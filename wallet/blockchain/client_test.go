package blockchain

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/horizen-cce-common-go/wallet/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectAlreadyConnectedIsIdempotent(t *testing.T) {
	c := SetupNewBlockChainClientConnected(nil, ethCommon.Address{}, ethCommon.Address{}, &bind.TransactOpts{})
	err := c.Connect(context.Background())
	require.NoError(t, err)
}

func TestSubmitRequestRequiresConnected(t *testing.T) {
	c := NewBlockChainClient(ethCommon.Address{}, ethCommon.Address{}, "", nil)
	_, _, err := c.SubmitRequest(context.Background(), 0, common.NewApplicationId(1), common.Process, nil, big.NewInt(0), big.NewInt(0))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected")
}

func TestGetTeePublicKeyRequiresConnected(t *testing.T) {
	c := NewBlockChainClient(ethCommon.Address{}, ethCommon.Address{}, "", nil)
	_, err := c.GetTeePublicKey(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client not connected")
}

func TestGetTeePublicKeyRequiresConfiguredContract(t *testing.T) {
	c := SetupNewBlockChainClientConnected(nil, ethCommon.Address{}, ethCommon.Address{}, &bind.TransactOpts{})
	c.teeAuthBoundContract = nil
	_, err := c.GetTeePublicKey(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestCloseResetsClient(t *testing.T) {
	c := SetupNewBlockChainClientConnected(nil, ethCommon.Address{}, ethCommon.Address{}, &bind.TransactOpts{})
	require.NoError(t, c.Close())
	assert.False(t, c.connected)
}

func TestMockClientDefaultAndOverrides(t *testing.T) {
	m := NewMockClient().WithTeePublicKey([]byte{1, 2, 3})
	reqID, _, err := m.SubmitRequest(context.Background(), 0, common.NewApplicationId(1), common.Process, nil, big.NewInt(0), big.NewInt(0))
	require.NoError(t, err)
	assert.NotEqual(t, common.RequestIdType{}, reqID)

	pub, err := m.GetTeePublicKey(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3}, pub)

	expected := errors.New("submit failed")
	m.SubmitRequestFn = func(context.Context, uint8, common.ApplicationIdType, common.RequestType, []byte, *big.Int, *big.Int) (common.RequestIdType, uint64, error) {
		return common.RequestIdType{}, 0, expected
	}
	_, _, err = m.SubmitRequest(context.Background(), 0, common.NewApplicationId(1), common.Process, nil, big.NewInt(0), big.NewInt(0))
	require.ErrorIs(t, err, expected)
}
