package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/horizen-cce-common-go/wallet/blockchain/contracts/processorendpoint"
	"github.com/horizen-cce-common-go/wallet/blockchain/contracts/tee"
	"github.com/horizen-cce-common-go/wallet/common"
)

// ChainClient captures the go-ethereum capabilities needed by the app-side client.
type ChainClient interface {
	ethereum.BlockNumberReader
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer
	ethereum.GasPricer1559
	ethereum.FeeHistoryReader
	ethereum.LogFilterer
	ethereum.PendingStateReader
	ethereum.PendingContractCaller
	ethereum.TransactionReader
	ethereum.TransactionSender
	ethereum.ChainIDReader
}

// BlockChainClient is the app-side blockchain client used by wallet applications.
type BlockChainClient struct {
	mu                     sync.RWMutex
	connected              bool
	processorAddress       ethCommon.Address
	teeAuthAddress         ethCommon.Address
	rpcURL                 string
	processorBoundContract *bind.BoundContract
	processorEndpoint      *processorendpoint.ProcessorEndpoint
	teeAuthBoundContract   *bind.BoundContract
	teeAuthEndpoint        *tee.TeeAuthenticator
	client                 ChainClient
	account                *bind.TransactOpts
	privKey                *ecdsa.PrivateKey
	dial                   func(string) (ChainClient, error)
}

func defaultDial(url string) (ChainClient, error) {
	return ethclient.Dial(url)
}

func NewBlockChainClient(processor ethCommon.Address, teeAuthenticator ethCommon.Address, rpcURL string, key *ecdsa.PrivateKey) *BlockChainClient {
	return &BlockChainClient{
		processorAddress:  processor,
		teeAuthAddress:    teeAuthenticator,
		rpcURL:            rpcURL,
		processorEndpoint: processorendpoint.NewProcessorEndpoint(),
		teeAuthEndpoint:   tee.NewTeeAuthenticator(),
		privKey:           key,
		dial:              defaultDial,
	}
}

// SetupNewBlockChainClientConnected wires a client to an existing chain backend for tests.
func SetupNewBlockChainClientConnected(client ChainClient, processorContractAddress ethCommon.Address, teeSignerAddress ethCommon.Address, account *bind.TransactOpts) *BlockChainClient {
	blockchainClient := NewBlockChainClient(processorContractAddress, teeSignerAddress, "", nil)
	blockchainClient.client = client
	blockchainClient.processorBoundContract = blockchainClient.processorEndpoint.Instance(blockchainClient.client, processorContractAddress)
	blockchainClient.teeAuthBoundContract = blockchainClient.teeAuthEndpoint.Instance(blockchainClient.client, teeSignerAddress)
	blockchainClient.account = account
	blockchainClient.connected = true
	return blockchainClient
}

func (c *BlockChainClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		// Idempotent reconnect: if already initialized, keep current state.
		return nil
	}

	client, err := c.dial(c.rpcURL)
	if err != nil {
		return fmt.Errorf("cannot connect to chain: %w", err)
	}
	c.client = client

	c.processorBoundContract = c.processorEndpoint.Instance(c.client, c.processorAddress)
	if c.teeAuthEndpoint != nil && c.teeAuthAddress != (ethCommon.Address{}) {
		c.teeAuthBoundContract = c.teeAuthEndpoint.Instance(c.client, c.teeAuthAddress)
	}

	if c.privKey != nil {
		chainID, err := c.client.ChainID(ctx)
		if err != nil {
			return fmt.Errorf("failed to retrieve chain ID: %w", err)
		}
		c.account = bind.NewKeyedTransactor(c.privKey, chainID)
	}

	c.connected = true
	return nil
}

func (c *BlockChainClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.client = nil
	c.processorBoundContract = nil
	c.teeAuthBoundContract = nil
	c.account = nil
	c.connected = false
	return nil
}

func (c *BlockChainClient) unpackProcessorEndpointError(chainErr error) error {
	if chainErr == nil {
		return nil
	}

	raw, hasRevertErrorData := ethclient.RevertErrorData(chainErr)
	if !hasRevertErrorData || len(raw) == 0 {
		return fmt.Errorf("call returned error: %w", chainErr)
	}

	rawUnpackedErr, unpackErr := c.processorEndpoint.UnpackError(raw)
	if unpackErr != nil {
		return fmt.Errorf("call returned unknown error: %w", chainErr)
	}

	return fmt.Errorf("contract revert: %T", rawUnpackedErr)
}

func (c *BlockChainClient) SubmitRequest(ctx context.Context, protocolVersion uint8, applicationID common.ApplicationIdType, requestType common.RequestType, payload []byte, depositAmount *big.Int, maxFeeValue *big.Int) (common.RequestIdType, uint64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return common.RequestIdType{}, 0, fmt.Errorf("client not connected, call Connect first")
	}
	if c.account == nil {
		return common.RequestIdType{}, 0, fmt.Errorf("client not configured for signing transactions")
	}

	if depositAmount == nil {
		depositAmount = big.NewInt(0)
	}
	if maxFeeValue == nil {
		maxFeeValue = big.NewInt(0)
	}

	data := c.processorEndpoint.PackSubmitRequest(
		protocolVersion,
		processorendpoint.ApplicationIdToBindingType(applicationID),
		uint8(requestType),
		payload,
		depositAmount,
		maxFeeValue,
	)

	c.account.Value = new(big.Int).Add(depositAmount, maxFeeValue)
	tx, err := bind.Transact(c.processorBoundContract, c.account, data)
	c.account.Value = nil
	if err != nil {
		return common.RequestIdType{}, 0, fmt.Errorf("failed to submit transaction: %w", c.unpackProcessorEndpointError(err))
	}

	receipt, err := bind.WaitMined(ctx, c.client, tx.Hash())
	if err != nil {
		return common.RequestIdType{}, 0, fmt.Errorf("error waiting for tx inclusion: %w", err)
	}
	if receipt.Status != 1 {
		return common.RequestIdType{}, 0, fmt.Errorf("transaction failed")
	}

	for _, vLog := range receipt.Logs {
		event, err := c.processorEndpoint.UnpackRequestSubmittedEvent(vLog)
		if err == nil {
			return common.RequestIdType(event.RequestId), receipt.BlockNumber.Uint64(), nil
		}
	}

	return common.RequestIdType{}, 0, fmt.Errorf("requestId not found in logs")
}

func (c *BlockChainClient) GetTeePublicKey(ctx context.Context) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("client not connected, call Connect first")
	}
	if c.teeAuthBoundContract == nil || c.teeAuthEndpoint == nil {
		return nil, fmt.Errorf("tee authenticator contract not configured")
	}

	pubSecp521r1, err := bind.Call(
		c.teeAuthBoundContract,
		&bind.CallOpts{Pending: false},
		c.teeAuthEndpoint.PackGetPubSecp521r1(),
		c.teeAuthEndpoint.UnpackGetPubSecp521r1,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve pubSecp521r1: %w", err)
	}

	pub := make([]byte, len(pubSecp521r1))
	copy(pub, pubSecp521r1)
	return pub, nil
}
