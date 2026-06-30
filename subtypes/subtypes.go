// Package subtypes is the cross-repo home of the privacy-preserving event
// subtype primitives: the seed-derivation message constant and the HMAC-SHA-256
// subtype-set generator. The framework's `encryptEvents` picks a random subtype
// from this set per event; off-chain consumers (wallets, schedulers) recompute
// the set from the seed to filter their own events on the subgraph.
//
// MIGRATION TARGET (b-full): today the same constants live duplicated in
// `vela/pkg/executor/subtype.go` (`SubtypeKeyMessage`, `DefaultSubtypeN`,
// `GenerateSubtype`/`AllSubtypes`/`GenerateRandomSubtype`) and in
// `vela-nova/wallet/cmd/seed.go` (`SubtypeKeyMessage`, `DefaultSubtypeN` and a
// local seed-derivation function). End state: those duplicates collapse into
// this package — vela's executor and vela-nova's wallet both import the
// constants and the generator from here, and vela's `pkg/executor/subtype.go`
// keeps only `GenerateRandomSubtype` (which is host-only and depends on
// crypto/rand). Tracked as a follow-up; this package is currently the
// **incremental** target (additive only — vela and vela-nova continue working
// with their existing duplicates until the dedup PRs land).
package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
)

// SubtypeKeyMessage is the fixed bytestring signed by a user's secp256k1 EOA
// key (after `keccak256`-hashing) to deterministically derive the user's
// 65-byte seed. The seed is what the framework's `encryptEvents` consumes for
// per-event subtype randomization, and what off-chain consumers feed into
// `GenerateSubtypesN` to recover the filter set.
//
// Changing the constant rotates every user's subtype set globally — existing
// seeds become invalid and every user has to re-ASSOCIATEKEY. Treat as
// load-bearing for the protocol's wire format.
const SubtypeKeyMessage = "subtype-key-v1"

// DefaultSubtypeN is the number of privacy-preserving subtypes generated per seed.
const DefaultSubtypeN = 50

// GenerateSubtypes returns DefaultSubtypeN privacy-preserving event subtypes
// derived from the given seed using HMAC-SHA256.
func GenerateSubtypes(seed []byte) [][32]byte {
	return GenerateSubtypesN(seed, DefaultSubtypeN)
}

// GenerateSubtypesN returns n privacy-preserving event subtypes derived from
// the given seed. Each subtype is HMAC-SHA256(key=seed, data=byte(index))
// packed as a 32-byte value, for index in [1, n]. The raw 32 bytes match the
// on-chain bytes32 event subtype — no hex re-encoding at the boundary.
func GenerateSubtypesN(seed []byte, n int) [][32]byte {
	subtypes := make([][32]byte, n)
	for i := range n {
		mac := hmac.New(sha256.New, seed)
		mac.Write([]byte{byte(i + 1)})
		copy(subtypes[i][:], mac.Sum(nil))
	}
	return subtypes
}
