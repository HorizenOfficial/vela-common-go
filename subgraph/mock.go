package subgraph

import (
	"context"
	"math/big"
	"sort"

	"github.com/HorizenOfficial/vela-common-go/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

// MockClient provides canned responses for tests.
type MockClient struct {
	requests       map[common.RequestIdType]*RequestCompleted
	deployRequests map[common.RequestIdType]*RequestCompleted
	events         map[common.ApplicationIdType][]UserEvent
	appEvents      map[common.ApplicationIdType][]AppEvent
}

func NewMockClient() *MockClient {
	return &MockClient{
		requests:       make(map[common.RequestIdType]*RequestCompleted),
		deployRequests: make(map[common.RequestIdType]*RequestCompleted),
		events:         make(map[common.ApplicationIdType][]UserEvent),
		appEvents:      make(map[common.ApplicationIdType][]AppEvent),
	}
}

func (m *MockClient) WithRequestCompleted(rc *RequestCompleted) *MockClient {
	if rc != nil {
		m.requests[rc.RequestID] = rc
	}
	return m
}

func (m *MockClient) WithDeployRequestCompleted(rc *RequestCompleted) *MockClient {
	if rc != nil {
		m.deployRequests[rc.RequestID] = rc
	}
	return m
}

func (m *MockClient) WithUserEvents(appID common.ApplicationIdType, events []UserEvent) *MockClient {
	m.events[appID] = events
	return m
}

func (m *MockClient) WithAppEvents(appID common.ApplicationIdType, events []AppEvent) *MockClient {
	m.appEvents[appID] = events
	return m
}

func (m *MockClient) GetRequestCompletedByID(_ context.Context, requestID common.RequestIdType) (*RequestCompleted, error) {
	if rc, ok := m.requests[requestID]; ok {
		return rc, nil
	}
	return nil, nil
}

func (m *MockClient) GetDeployRequestCompletedByID(_ context.Context, requestID common.RequestIdType) (*RequestCompleted, error) {
	if rc, ok := m.deployRequests[requestID]; ok {
		return rc, nil
	}
	return nil, nil
}

func (m *MockClient) HealthCheck(context.Context) error {
	return nil
}

func (m *MockClient) GetUserEvents(_ context.Context, applicationID common.ApplicationIdType, eventSubType [32]byte, limit int, before *big.Int) ([]UserEvent, error) {
	var subTypes [][32]byte
	if eventSubType != ([32]byte{}) {
		subTypes = [][32]byte{eventSubType}
	}
	return m.GetUserEventsBySubTypes(context.Background(), applicationID, subTypes, limit, before)
}

func (m *MockClient) GetUserEventsBySubTypes(_ context.Context, applicationID common.ApplicationIdType, eventSubTypes [][32]byte, limit int, before *big.Int) ([]UserEvent, error) {
	all, ok := m.events[applicationID]
	if !ok {
		return nil, nil
	}

	subTypeSet := make(map[[32]byte]bool, len(eventSubTypes))
	for _, s := range eventSubTypes {
		subTypeSet[s] = true
	}

	var filtered []UserEvent
	for _, ev := range all {
		if len(subTypeSet) > 0 && !subTypeSet[ev.EventSubType] {
			continue
		}
		if before != nil && ComputeSortKey(ev).Cmp(before) >= 0 {
			continue
		}
		filtered = append(filtered, ev)
	}

	return mockApplySortAndLimit(filtered, limit), nil
}

func (m *MockClient) GetAppEvents(_ context.Context, applicationID common.ApplicationIdType, eventSubType [32]byte, limit int, before *big.Int) ([]AppEvent, error) {
	var subTypes [][32]byte
	if eventSubType != ([32]byte{}) {
		subTypes = [][32]byte{eventSubType}
	}
	return m.GetAppEventsBySubTypes(context.Background(), applicationID, subTypes, limit, before)
}

func (m *MockClient) GetAppEventsBySubTypes(_ context.Context, applicationID common.ApplicationIdType, eventSubTypes [][32]byte, limit int, before *big.Int) ([]AppEvent, error) {
	all, ok := m.appEvents[applicationID]
	if !ok {
		return nil, nil
	}

	subTypeSet := make(map[[32]byte]bool, len(eventSubTypes))
	for _, s := range eventSubTypes {
		subTypeSet[s] = true
	}

	var filtered []AppEvent
	for _, ev := range all {
		if len(subTypeSet) > 0 && !subTypeSet[ev.EventSubType] {
			continue
		}
		if before != nil && ComputeAppEventSortKey(ev).Cmp(before) >= 0 {
			continue
		}
		filtered = append(filtered, ev)
	}

	return mockApplySortAndLimitAppEvents(filtered, limit), nil
}

func (m *MockClient) GetRefunds(_ context.Context, _ common.ApplicationIdType, _ *common.RequestIdType, _ int) ([]OnChainRefund, error) {
	return nil, nil
}

func (m *MockClient) GetWithdrawals(_ context.Context, _ common.ApplicationIdType, _ *common.RequestIdType, _ int) ([]OnChainWithdrawal, error) {
	return nil, nil
}

func (m *MockClient) GetClaimsExecuted(_ context.Context, _ ethCommon.Address, _ *ethCommon.Address, _ int) ([]ClaimExecuted, error) {
	return nil, nil
}

func mockApplySortAndLimit(events []UserEvent, limit int) []UserEvent {
	sort.Slice(events, func(i, j int) bool {
		return ComputeSortKey(events[i]).Cmp(ComputeSortKey(events[j])) > 0
	})

	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}

func mockApplySortAndLimitAppEvents(events []AppEvent, limit int) []AppEvent {
	sort.Slice(events, func(i, j int) bool {
		return ComputeAppEventSortKey(events[i]).Cmp(ComputeAppEventSortKey(events[j])) > 0
	})

	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}
	if len(events) <= limit {
		return events
	}
	return events[:limit]
}
