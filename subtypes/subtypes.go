package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// DefaultSubtypeN is the number of privacy-preserving subtypes generated per seed.
const DefaultSubtypeN = 50

// GenerateSubtypes returns DefaultSubtypeN privacy-preserving event subtypes
// derived from the given seed using HMAC-SHA256.
func GenerateSubtypes(seed []byte) []string {
	return GenerateSubtypesN(seed, DefaultSubtypeN)
}

// GenerateSubtypesN returns n privacy-preserving event subtypes derived from
// the given seed. Each subtype is "0x" + hex(HMAC-SHA256(key=seed, data=byte(index)))
// for index in [1, n].
func GenerateSubtypesN(seed []byte, n int) []string {
	subtypes := make([]string, n)
	for i := 0; i < n; i++ {
		mac := hmac.New(sha256.New, seed)
		mac.Write([]byte{byte(i + 1)})
		subtypes[i] = "0x" + hex.EncodeToString(mac.Sum(nil))
	}
	return subtypes
}
