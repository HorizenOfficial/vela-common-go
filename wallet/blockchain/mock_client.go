package blockchain

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/horizen-cce-common-go/wallet/common"
)

// MockClient is a lightweight app-side test double.
type MockClient struct {
	SubmitRequestFn   func(ctx context.Context, protocolVersion uint8, applicationID common.ApplicationIdType, requestType common.RequestType, payload []byte, depositAmount *big.Int, maxFeeValue *big.Int) (common.RequestIdType, uint64, error)
	GetTeePublicKeyFn func(ctx context.Context) ([]byte, error)
	ConnectFn         func(ctx context.Context) error
	CloseFn           func() error

	teePub []byte
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) WithTeePublicKey(pub []byte) *MockClient {
	if pub == nil {
		m.teePub = nil
		return m
	}
	m.teePub = append([]byte(nil), pub...)
	return m
}

func (m *MockClient) Connect(ctx context.Context) error {
	if m.ConnectFn != nil {
		return m.ConnectFn(ctx)
	}
	return nil
}

func (m *MockClient) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

func (m *MockClient) SubmitRequest(ctx context.Context, protocolVersion uint8, applicationID common.ApplicationIdType, requestType common.RequestType, payload []byte, depositAmount *big.Int, maxFeeValue *big.Int) (common.RequestIdType, uint64, error) {
	if m.SubmitRequestFn != nil {
		return m.SubmitRequestFn(ctx, protocolVersion, applicationID, requestType, payload, depositAmount, maxFeeValue)
	}

	var id common.RequestIdType
	if _, err := rand.Read(id[:]); err != nil {
		return common.RequestIdType{}, 0, fmt.Errorf("failed to generate request id: %w", err)
	}
	return id, 0, nil
}

func (m *MockClient) GetTeePublicKey(ctx context.Context) ([]byte, error) {
	if m.GetTeePublicKeyFn != nil {
		return m.GetTeePublicKeyFn(ctx)
	}
	if m.teePub == nil {
		return nil, fmt.Errorf("tee public key not configured")
	}
	return append([]byte(nil), m.teePub...), nil
}
