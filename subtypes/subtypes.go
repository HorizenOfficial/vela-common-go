package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
)

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
