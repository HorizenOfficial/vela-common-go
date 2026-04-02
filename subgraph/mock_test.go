package subgraph

import (
	"context"
	"testing"

	"github.com/HorizenOfficial/vela-common-go/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMockClient_HealthCheck verifies that the mock always returns nil (healthy).
func TestMockClient_HealthCheck(t *testing.T) {
	m := NewMockClient()
	err := m.HealthCheck(context.Background())
	require.NoError(t, err)
}

// TestMockClient_GetRequestCompleted_NotFound verifies that querying for an
// unknown request ID returns nil without error.
func TestMockClient_GetRequestCompleted_NotFound(t *testing.T) {
	m := NewMockClient()
	rc, err := m.GetRequestCompletedByID(context.Background(), common.RequestIdType{})
	require.NoError(t, err)
	assert.Nil(t, rc)
}

// TestMockClient_GetRequestCompleted_Found verifies that a registered
// RequestCompleted is returned for the matching request ID.
func TestMockClient_GetRequestCompleted_Found(t *testing.T) {
	var reqID common.RequestIdType
	reqID[31] = 1

	rc := &RequestCompleted{
		RequestID: reqID,
		Status:    common.RequestResultOK,
	}
	m := NewMockClient().WithRequestCompleted(rc)

	result, err := m.GetRequestCompletedByID(context.Background(), reqID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, common.RequestResultOK, result.Status)
}

// TestMockClient_WithRequestCompleted_Nil verifies that passing nil does not
// panic or add an entry.
func TestMockClient_WithRequestCompleted_Nil(t *testing.T) {
	m := NewMockClient().WithRequestCompleted(nil)
	rc, err := m.GetRequestCompletedByID(context.Background(), common.RequestIdType{})
	require.NoError(t, err)
	assert.Nil(t, rc)
}

// TestMockClient_GetUserEvents_Empty verifies that querying an application
// with no registered events returns nil.
func TestMockClient_GetUserEvents_Empty(t *testing.T) {
	m := NewMockClient()
	events, err := m.GetUserEvents(context.Background(), common.NewApplicationId(1), "", 10, nil)
	require.NoError(t, err)
	assert.Nil(t, events)
}

// TestMockClient_GetUserEvents_ReturnsAll verifies that all registered events
// are returned when limit is large enough and no filter is applied.
func TestMockClient_GetUserEvents_ReturnsAll(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 1, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 2, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 11, LogIndex: 0, EventSubType: "a"},
	})

	events, err := m.GetUserEvents(context.Background(), appID, "", 100, nil)
	require.NoError(t, err)
	require.Len(t, events, 3)
}

// TestMockClient_GetUserEvents_SortedDescending verifies that events are
// returned in descending sort key order (highest block/logIndex first).
func TestMockClient_GetUserEvents_SortedDescending(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 5, LogIndex: 0, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 0, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 7, LogIndex: 0, EventSubType: "a"},
	})

	events, err := m.GetUserEvents(context.Background(), appID, "", 100, nil)
	require.NoError(t, err)
	require.Len(t, events, 3)

	assert.Equal(t, uint64(10), events[0].BlockNumber)
	assert.Equal(t, uint64(7), events[1].BlockNumber)
	assert.Equal(t, uint64(5), events[2].BlockNumber)
}

// TestMockClient_GetUserEvents_Limit verifies that the mock respects the
// limit parameter and returns at most limit events.
func TestMockClient_GetUserEvents_Limit(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 1, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 2, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 3, EventSubType: "a"},
	})

	events, err := m.GetUserEvents(context.Background(), appID, "", 2, nil)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

// TestMockClient_GetUserEvents_Before verifies that the before parameter
// filters out events with a sort key >= the given value, enabling pagination.
func TestMockClient_GetUserEvents_Before(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 5, LogIndex: 0, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 0, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 15, LogIndex: 0, EventSubType: "a"},
	})

	// Set before to block 10's sort key — should exclude blocks 10 and 15.
	before := ComputeSortKey(UserEvent{BlockNumber: 10, LogIndex: 0})
	events, err := m.GetUserEvents(context.Background(), appID, "", 100, before)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, uint64(5), events[0].BlockNumber)
}

// TestMockClient_GetUserEvents_EventSubTypeFilter verifies that events are
// filtered by eventSubType when a non-empty value is provided.
func TestMockClient_GetUserEvents_EventSubTypeFilter(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 1, EventSubType: "deposit"},
		{ApplicationID: appID, BlockNumber: 2, EventSubType: "withdrawal"},
		{ApplicationID: appID, BlockNumber: 3, EventSubType: "deposit"},
	})

	events, err := m.GetUserEvents(context.Background(), appID, "deposit", 100, nil)
	require.NoError(t, err)
	require.Len(t, events, 2)
	for _, ev := range events {
		assert.Equal(t, "deposit", ev.EventSubType)
	}
}

// TestMockClient_GetUserEvents_WrongAppID verifies that events for a different
// application ID are not returned.
func TestMockClient_GetUserEvents_WrongAppID(t *testing.T) {
	appID1 := common.NewApplicationId(1)
	appID2 := common.NewApplicationId(2)
	m := NewMockClient().WithUserEvents(appID1, []UserEvent{
		{ApplicationID: appID1, BlockNumber: 1, EventSubType: "a"},
	})

	events, err := m.GetUserEvents(context.Background(), appID2, "", 100, nil)
	require.NoError(t, err)
	assert.Nil(t, events)
}

// TestMockClient_GetUserEvents_Pagination simulates a two-page fetch: first
// page returns the newest event, second page uses before to get the older one.
func TestMockClient_GetUserEvents_Pagination(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 5, LogIndex: 0, EventSubType: "a"},
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 0, EventSubType: "a"},
	})

	// First page: limit 1.
	page1, err := m.GetUserEvents(context.Background(), appID, "", 1, nil)
	require.NoError(t, err)
	require.Len(t, page1, 1)
	assert.Equal(t, uint64(10), page1[0].BlockNumber)

	// Second page: before the first page's sort key.
	before := ComputeSortKey(page1[0])
	page2, err := m.GetUserEvents(context.Background(), appID, "", 1, before)
	require.NoError(t, err)
	require.Len(t, page2, 1)
	assert.Equal(t, uint64(5), page2[0].BlockNumber)

	// Third page: before the second page's sort key — should be empty.
	before = ComputeSortKey(page2[0])
	page3, err := m.GetUserEvents(context.Background(), appID, "", 1, before)
	require.NoError(t, err)
	assert.Empty(t, page3)
}

// TestMockClient_GetUserEventsBySubTypes_FiltersMultiple verifies that events
// are filtered to those matching any of the provided subtypes.
func TestMockClient_GetUserEventsBySubTypes_FiltersMultiple(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 1, EventSubType: "0xaaa"},
		{ApplicationID: appID, BlockNumber: 2, EventSubType: "0xbbb"},
		{ApplicationID: appID, BlockNumber: 3, EventSubType: "0xccc"},
		{ApplicationID: appID, BlockNumber: 4, EventSubType: "0xddd"},
	})

	events, err := m.GetUserEventsBySubTypes(context.Background(), appID, []string{"0xaaa", "0xccc"}, 100, nil)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "0xccc", events[0].EventSubType) // descending order
	assert.Equal(t, "0xaaa", events[1].EventSubType)
}

// TestMockClient_GetUserEventsBySubTypes_EmptySlice returns all events when
// no subtypes are specified.
func TestMockClient_GetUserEventsBySubTypes_EmptySlice(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 1, EventSubType: "0xaaa"},
		{ApplicationID: appID, BlockNumber: 2, EventSubType: "0xbbb"},
	})

	events, err := m.GetUserEventsBySubTypes(context.Background(), appID, nil, 100, nil)
	require.NoError(t, err)
	assert.Len(t, events, 2)
}

// TestMockClient_GetUserEventsBySubTypes_Before verifies pagination with subtypes.
func TestMockClient_GetUserEventsBySubTypes_Before(t *testing.T) {
	appID := common.NewApplicationId(1)
	m := NewMockClient().WithUserEvents(appID, []UserEvent{
		{ApplicationID: appID, BlockNumber: 5, LogIndex: 0, EventSubType: "0xaaa"},
		{ApplicationID: appID, BlockNumber: 10, LogIndex: 0, EventSubType: "0xaaa"},
		{ApplicationID: appID, BlockNumber: 15, LogIndex: 0, EventSubType: "0xbbb"},
	})

	before := ComputeSortKey(UserEvent{BlockNumber: 15, LogIndex: 0})
	events, err := m.GetUserEventsBySubTypes(context.Background(), appID, []string{"0xaaa", "0xbbb"}, 100, before)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, uint64(10), events[0].BlockNumber)
	assert.Equal(t, uint64(5), events[1].BlockNumber)
}

// TestMockClient_ImplementsClientInterface verifies that *MockClient satisfies
// the Client interface at compile time.
func TestMockClient_ImplementsClientInterface(t *testing.T) {
	var _ Client = NewMockClient()
}
