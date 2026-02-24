package blockchain

import (
	"context"
	"math/big"

	"github.com/horizen-cce-common-go/wallet/common"
)

// Client defines the app-side blockchain operations used by wallets.
type Client interface {
	SubmitRequest(ctx context.Context, protocolVersion uint8, applicationID common.ApplicationIdType, requestType common.RequestType, payload []byte, depositAmount *big.Int, maxFeeValue *big.Int) (common.RequestIdType, uint64, error)
	GetTeePublicKey(ctx context.Context) ([]byte, error)
	Connect(ctx context.Context) error
	Close() error
}
