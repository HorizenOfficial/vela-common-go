// Package subtypes is the cross-repo home of the privacy-preserving event
// subtype primitives: the seed-derivation message constant and the
// HMAC-SHA-256-based deterministic subtype-set generator. The framework's
// `encryptEvents` picks a random subtype from this set per event; off-chain
// consumers (wallets, schedulers) recompute the set from the seed to filter
// their own events on the subgraph.
//
// Cross-repo position: `vela/pkg/executor/subtype.go` keeps only
// `GenerateRandomSubtype` (host-only — uses `crypto/rand`) and imports the
// constants + deterministic primitives from here; `vela-nova/wallet` and
// `vela-ned/scheduler` import the deterministic primitives directly. There
// is only one source of truth for `SubtypeKeyMessage` and `DefaultSubtypeN`
// — this file.
package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
)

// SubtypeKeyMessage is the fixed bytestring signed by a user's secp256k1 EOA
// key (after `keccak256`-hashing) to deterministically derive the user's
// 65-byte seed. The seed is what the framework's `encryptEvents` consumes for
// per-event subtype randomization, and what off-chain consumers feed into
// `AllSubtypes` to recover the filter set.
//
// Changing the constant rotates every user's subtype set globally — existing
// seeds become invalid and every user has to re-ASSOCIATEKEY. Treat as
// load-bearing for the protocol's wire format.
const SubtypeKeyMessage = "subtype-key-v1"

// DefaultSubtypeN is the number of privacy-preserving subtypes generated per seed.
const DefaultSubtypeN = 50

// GenerateSubtype returns HMAC-SHA256(key=seed, data=[]byte{index}) as a
// 32-byte value, matching the on-chain `bytes32` event subtype. `index`
// should be in the range [1, n] where n is the consumer's chosen anonymity
// set size (`DefaultSubtypeN` for the framework default).
func GenerateSubtype(seed []byte, index int) [32]byte {
	mac := hmac.New(sha256.New, seed)
	mac.Write([]byte{byte(index)})
	var out [32]byte
	copy(out[:], mac.Sum(nil))
	return out
}

// AllSubtypes returns `GenerateSubtype(seed, i)` for `i` in `[1, n]`. The
// returned slice has length n; index `i` maps to `result[i-1]`. This is the
// deterministic filter set off-chain consumers feed to the subgraph's
// `GetUserEventsBySubTypes` to discover events `encryptEvents` rotated via
// `GenerateRandomSubtype(seed, n)`.
func AllSubtypes(seed []byte, n int) [][32]byte {
	out := make([][32]byte, n)
	for i := 1; i <= n; i++ {
		out[i-1] = GenerateSubtype(seed, i)
	}
	return out
}
