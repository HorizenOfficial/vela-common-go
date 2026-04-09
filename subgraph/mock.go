package subgraph

import (
	"context"
	"math/big"
	"sort"
	"strings"

	"github.com/HorizenOfficial/vela-common-go/common"
)

// MockClient provides canned responses for tests.
type MockClient struct {
	requests       map[common.RequestIdType]*RequestCompleted
	deployRequests map[common.RequestIdType]*RequestCompleted
	events         map[common.ApplicationIdType][]UserEvent
}

func NewMockClient() *MockClient {
	return &MockClient{
		requests:       make(map[common.RequestIdType]*RequestCompleted),
		deployRequests: make(map[common.RequestIdType]*RequestCompleted),
		events:         make(map[common.ApplicationIdType][]UserEvent),
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

func (m *MockClient) GetUserEvents(_ context.Context, applicationID common.ApplicationIdType, eventSubType string, limit int, before *big.Int) ([]UserEvent, error) {
	var subTypes []string
	if strings.TrimSpace(eventSubType) != "" {
		subTypes = []string{eventSubType}
	}
	return m.GetUserEventsBySubTypes(context.Background(), applicationID, subTypes, limit, before)
}

func (m *MockClient) GetUserEventsBySubTypes(_ context.Context, applicationID common.ApplicationIdType, eventSubTypes []string, limit int, before *big.Int) ([]UserEvent, error) {
	all, ok := m.events[applicationID]
	if !ok {
		return nil, nil
	}

	subTypeSet := make(map[string]bool, len(eventSubTypes))
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
