package subgraph

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeSortKey_FromBlockAndLogIndex verifies the sort key formula:
// BlockNumber * SortKeyBase + LogIndex.
func TestComputeSortKey_FromBlockAndLogIndex(t *testing.T) {
	ev := UserEvent{BlockNumber: 100, LogIndex: 5}
	key := ComputeSortKey(ev)

	expected := new(big.Int).SetUint64(100*SortKeyBase + 5)
	assert.Equal(t, expected, key)
}

// TestComputeSortKey_ZeroBlockAndLogIndex ensures a zero-valued event produces
// a zero sort key.
func TestComputeSortKey_ZeroBlockAndLogIndex(t *testing.T) {
	ev := UserEvent{BlockNumber: 0, LogIndex: 0}
	key := ComputeSortKey(ev)
	assert.Equal(t, big.NewInt(0), key)
}

// TestComputeSortKey_LargeBlockNumber checks that high block numbers and log
// indices don't overflow the big.Int computation.
func TestComputeSortKey_LargeBlockNumber(t *testing.T) {
	ev := UserEvent{BlockNumber: 1_000_000, LogIndex: 999}
	key := ComputeSortKey(ev)

	expected := new(big.Int).SetUint64(1_000_000*SortKeyBase + 999)
	assert.Equal(t, expected, key)
}

// TestComputeSortKey_UsesPresetSortKey verifies that when the event already
// carries a SortKey the computation is skipped and the preset value is returned.
func TestComputeSortKey_UsesPresetSortKey(t *testing.T) {
	preset := big.NewInt(42)
	ev := UserEvent{BlockNumber: 100, LogIndex: 5, SortKey: preset}
	key := ComputeSortKey(ev)

	assert.Equal(t, big.NewInt(42), key)
}

// TestComputeSortKey_ReturnsCopyOfPresetSortKey ensures that mutating the
// returned sort key does not affect the original event's SortKey field.
func TestComputeSortKey_ReturnsCopyOfPresetSortKey(t *testing.T) {
	preset := big.NewInt(42)
	ev := UserEvent{SortKey: preset}

	key := ComputeSortKey(ev)
	key.SetInt64(999)

	assert.Equal(t, big.NewInt(42), preset)
}

// TestComputeSortKey_HigherBlockComesFirst confirms that a higher block number
// produces a larger sort key, which determines descending order.
func TestComputeSortKey_HigherBlockComesFirst(t *testing.T) {
	evA := UserEvent{BlockNumber: 10, LogIndex: 0}
	evB := UserEvent{BlockNumber: 11, LogIndex: 0}

	require.True(t, ComputeSortKey(evB).Cmp(ComputeSortKey(evA)) > 0)
}

// TestComputeSortKey_HigherLogIndexWithinBlock confirms that within the same
// block, a higher log index produces a larger sort key.
func TestComputeSortKey_HigherLogIndexWithinBlock(t *testing.T) {
	evA := UserEvent{BlockNumber: 10, LogIndex: 1}
	evB := UserEvent{BlockNumber: 10, LogIndex: 2}

	require.True(t, ComputeSortKey(evB).Cmp(ComputeSortKey(evA)) > 0)
}
