package subgraph

import "math/big"

// SortKeyBase is the multiplier for sort key computation.
// Must match SORT_BASE in the subgraph mapping.
const SortKeyBase = uint64(1000000000000)

// ComputeSortKey returns the sort key for a UserEvent.
// If the event already has a SortKey set, a copy is returned.
// Otherwise it is computed as BlockNumber * SortKeyBase + LogIndex.
func ComputeSortKey(ev UserEvent) *big.Int {
	if ev.SortKey != nil {
		return new(big.Int).Set(ev.SortKey)
	}

	block := new(big.Int).SetUint64(ev.BlockNumber)
	base := new(big.Int).SetUint64(SortKeyBase)
	block.Mul(block, base)
	block.Add(block, new(big.Int).SetUint64(ev.LogIndex))
	return block
}
