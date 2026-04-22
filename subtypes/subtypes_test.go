package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSubtypes_ReturnsDefaultCount(t *testing.T) {
	seed := []byte("test-seed")
	subtypes := GenerateSubtypes(seed)
	assert.Len(t, subtypes, DefaultSubtypeN)
}

func TestGenerateSubtypesN_ReturnsRequestedCount(t *testing.T) {
	seed := []byte("test-seed")
	for _, n := range []int{1, 5, 10, 100} {
		subtypes := GenerateSubtypesN(seed, n)
		assert.Len(t, subtypes, n)
	}
}

func TestGenerateSubtypesN_NonZero(t *testing.T) {
	seed := []byte("test-seed")
	subtypes := GenerateSubtypesN(seed, 5)
	for _, s := range subtypes {
		assert.NotEqual(t, [32]byte{}, s, "subtype should not be all zeros")
	}
}

func TestGenerateSubtypesN_Deterministic(t *testing.T) {
	seed := []byte("deterministic-seed")
	a := GenerateSubtypesN(seed, 10)
	b := GenerateSubtypesN(seed, 10)
	assert.Equal(t, a, b)
}

func TestGenerateSubtypesN_DifferentSeedsDifferentResults(t *testing.T) {
	a := GenerateSubtypesN([]byte("seed-a"), 10)
	b := GenerateSubtypesN([]byte("seed-b"), 10)
	assert.NotEqual(t, a, b)
}

func TestGenerateSubtypesN_AllUnique(t *testing.T) {
	seed := []byte("unique-test-seed")
	subtypes := GenerateSubtypes(seed)
	seen := make(map[[32]byte]bool, len(subtypes))
	for _, s := range subtypes {
		assert.False(t, seen[s], "duplicate subtype: 0x%x", s)
		seen[s] = true
	}
}

func TestGenerateSubtypesN_MatchesManualHMAC(t *testing.T) {
	seed := []byte("verify-seed")
	subtypes := GenerateSubtypesN(seed, 3)

	for i := range 3 {
		mac := hmac.New(sha256.New, seed)
		mac.Write([]byte{byte(i + 1)})
		var expected [32]byte
		copy(expected[:], mac.Sum(nil))
		assert.Equal(t, expected, subtypes[i], "subtype[%d] mismatch", i)
	}
}

func TestGenerateSubtypesN_Zero(t *testing.T) {
	subtypes := GenerateSubtypesN([]byte("seed"), 0)
	assert.Empty(t, subtypes)
}
