package subgraph

import "math/big"

// SortKeyBase is the multiplier for sort key computation.
// Must match SORT_BASE in the subgraph mapping.
const SortKeyBase = uint64(1000000000000)

// ComputeSortKey returns the sort key for a UserEvent.
// If the event already has a SortKey set, a copy is returned.
// Otherwise it is computed as BlockNumber * SortKeyBase + LogIndex.
func ComputeSortKey(ev UserEvent) *big.Int {
	return computeSortKey(ev.SortKey, ev.BlockNumber, ev.LogIndex)
}

// ComputeAppEventSortKey returns the sort key for an AppEvent. Same formula
// as ComputeSortKey — kept separate because Go has no common parent type.
func ComputeAppEventSortKey(ev AppEvent) *big.Int {
	return computeSortKey(ev.SortKey, ev.BlockNumber, ev.LogIndex)
}

func computeSortKey(preset *big.Int, blockNumber, logIndex uint64) *big.Int {
	if preset != nil {
		return new(big.Int).Set(preset)
	}

	block := new(big.Int).SetUint64(blockNumber)
	base := new(big.Int).SetUint64(SortKeyBase)
	block.Mul(block, base)
	block.Add(block, new(big.Int).SetUint64(logIndex))
	return block
}
