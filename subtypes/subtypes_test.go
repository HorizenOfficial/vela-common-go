package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllSubtypes_DefaultCount(t *testing.T) {
	seed := []byte("test-seed")
	subs := AllSubtypes(seed, DefaultSubtypeN)
	assert.Len(t, subs, DefaultSubtypeN)
}

func TestAllSubtypes_ReturnsRequestedCount(t *testing.T) {
	seed := []byte("test-seed")
	for _, n := range []int{1, 5, 10, 100} {
		subs := AllSubtypes(seed, n)
		assert.Len(t, subs, n)
	}
}

func TestAllSubtypes_NonZero(t *testing.T) {
	seed := []byte("test-seed")
	subs := AllSubtypes(seed, 5)
	for _, s := range subs {
		assert.NotEqual(t, [32]byte{}, s, "subtype should not be all zeros")
	}
}

func TestAllSubtypes_Deterministic(t *testing.T) {
	seed := []byte("deterministic-seed")
	a := AllSubtypes(seed, 10)
	b := AllSubtypes(seed, 10)
	assert.Equal(t, a, b)
}

func TestAllSubtypes_DifferentSeedsDifferentResults(t *testing.T) {
	a := AllSubtypes([]byte("seed-a"), 10)
	b := AllSubtypes([]byte("seed-b"), 10)
	assert.NotEqual(t, a, b)
}

func TestAllSubtypes_AllUnique(t *testing.T) {
	seed := []byte("unique-test-seed")
	subs := AllSubtypes(seed, DefaultSubtypeN)
	seen := make(map[[32]byte]bool, len(subs))
	for _, s := range subs {
		assert.False(t, seen[s], "duplicate subtype: 0x%x", s)
		seen[s] = true
	}
}

func TestAllSubtypes_MatchesManualHMAC(t *testing.T) {
	seed := []byte("verify-seed")
	subs := AllSubtypes(seed, 3)

	for i := range 3 {
		mac := hmac.New(sha256.New, seed)
		mac.Write([]byte{byte(i + 1)})
		var expected [32]byte
		copy(expected[:], mac.Sum(nil))
		assert.Equal(t, expected, subs[i], "subtype[%d] mismatch", i)
	}
}

func TestAllSubtypes_Zero(t *testing.T) {
	subs := AllSubtypes([]byte("seed"), 0)
	assert.Empty(t, subs)
}

// TestGenerateSubtype_MatchesAllSubtypes confirms the single-index getter and
// the bulk generator agree — AllSubtypes(seed, n)[i-1] == GenerateSubtype(seed, i).
func TestGenerateSubtype_MatchesAllSubtypes(t *testing.T) {
	seed := []byte("getter-seed")
	all := AllSubtypes(seed, 5)
	for i := 1; i <= 5; i++ {
		assert.Equal(t, all[i-1], GenerateSubtype(seed, i), "index %d", i)
	}
}
