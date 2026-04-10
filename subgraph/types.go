package subgraph

import (
	"context"
	"math/big"

	"github.com/HorizenOfficial/vela-common-go/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

// Client defines the subgraph operations used by the services.
type Client interface {
	HealthCheck(ctx context.Context) error
	GetRequestCompletedByID(ctx context.Context, requestID common.RequestIdType) (*RequestCompleted, error)
	GetDeployRequestCompletedByID(ctx context.Context, requestID common.RequestIdType) (*RequestCompleted, error)
	GetUserEvents(ctx context.Context, applicationID common.ApplicationIdType, eventSubType string, limit int, before *big.Int) ([]UserEvent, error)
	GetUserEventsBySubTypes(ctx context.Context, applicationID common.ApplicationIdType, eventSubTypes []string, limit int, before *big.Int) ([]UserEvent, error)
	GetRefunds(ctx context.Context, applicationID common.ApplicationIdType, requestID *common.RequestIdType, limit int) ([]OnChainRefund, error)
	GetWithdrawals(ctx context.Context, applicationID common.ApplicationIdType, requestID *common.RequestIdType, limit int) ([]OnChainWithdrawal, error)
	GetClaimsExecuted(ctx context.Context, payee ethCommon.Address, tokenAddress *ethCommon.Address, limit int) ([]ClaimExecuted, error)
}

// RequestCompleted is the projection returned by the subgraph
// for both RequestCompleted and DeployRequestCompleted entities.
type RequestCompleted struct {
	ApplicationID   common.ApplicationIdType
	RequestID       common.RequestIdType
	Status          common.RequestResultStatus
	ErrorCode       uint8
	ErrorMessage    string
	ApplicationFees *big.Int
	BlockNumber     uint64
}

// UserEvent is the projection returned by the subgraph.
type UserEvent struct {
	ApplicationID common.ApplicationIdType
	RequestID     common.RequestIdType
	EventSubType  string
	EncryptedData []byte
	BlockNumber   uint64
	LogIndex      uint64
	SortKey       *big.Int
}

// OnChainRefund represents a Refund event indexed by the subgraph.
type OnChainRefund struct {
	ApplicationID common.ApplicationIdType
	RequestID     common.RequestIdType
	To            ethCommon.Address
	TokenAddress  ethCommon.Address
	Amount        *big.Int
	BlockNumber   uint64
}

// OnChainWithdrawal represents a Withdrawal event indexed by the subgraph.
type OnChainWithdrawal struct {
	ApplicationID common.ApplicationIdType
	RequestID     common.RequestIdType
	To            ethCommon.Address
	TokenAddress  ethCommon.Address
	Amount        *big.Int
	BlockNumber   uint64
}

// ClaimExecuted represents a PaymentWithdrawn event indexed by the subgraph.
type ClaimExecuted struct {
	TokenAddress ethCommon.Address
	Payee        ethCommon.Address
	Amount       *big.Int
	BlockNumber  uint64
}
