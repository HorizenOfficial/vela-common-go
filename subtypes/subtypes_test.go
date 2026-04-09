package subtypes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGenerateSubtypesN_HexFormat(t *testing.T) {
	seed := []byte("test-seed")
	subtypes := GenerateSubtypesN(seed, 5)
	for _, s := range subtypes {
		assert.True(t, strings.HasPrefix(s, "0x"), "subtype should start with 0x: %s", s)
		// 0x + 64 hex chars (32 bytes SHA-256)
		assert.Len(t, s, 66, "subtype should be 66 chars (0x + 64 hex): %s", s)
		// Verify it's valid hex
		_, err := hex.DecodeString(s[2:])
		require.NoError(t, err, "subtype should be valid hex: %s", s)
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
	seen := make(map[string]bool, len(subtypes))
	for _, s := range subtypes {
		assert.False(t, seen[s], "duplicate subtype: %s", s)
		seen[s] = true
	}
}

func TestGenerateSubtypesN_MatchesManualHMAC(t *testing.T) {
	seed := []byte("verify-seed")
	subtypes := GenerateSubtypesN(seed, 3)

	for i := 0; i < 3; i++ {
		mac := hmac.New(sha256.New, seed)
		mac.Write([]byte{byte(i + 1)})
		expected := "0x" + hex.EncodeToString(mac.Sum(nil))
		assert.Equal(t, expected, subtypes[i], "subtype[%d] mismatch", i)
	}
}

func TestGenerateSubtypesN_Zero(t *testing.T) {
	subtypes := GenerateSubtypesN([]byte("seed"), 0)
	assert.Empty(t, subtypes)
}
