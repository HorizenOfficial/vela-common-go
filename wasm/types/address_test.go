package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexToAddress(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errContains string
	}{
		{
			name:     "valid address with 0x prefix",
			input:    "0x1234567890abcdef1234567890abcdef12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "valid address without prefix",
			input:    "1234567890abcdef1234567890abcdef12345678",
			expected: "0x1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "valid address uppercase hex digits",
			input:    "0xABCDEF1234567890ABCDEF1234567890ABCDEF12",
			expected: "0xabcdef1234567890abcdef1234567890abcdef12",
		},
		{
			name:     "zero address",
			input:    "0x0000000000000000000000000000000000000000",
			expected: "0x0000000000000000000000000000000000000000",
		},
		{
			name:        "uppercase 0X prefix rejected",
			input:       "0X1234567890abcdef1234567890abcdef12345678",
			expectError: true,
			errContains: "only lowercase 0x is accepted",
		},
		{
			name:        "too short",
			input:       "0x12345678",
			expectError: true,
			errContains: "invalid address length",
		},
		{
			name:        "too long",
			input:       "0x1234567890abcdef1234567890abcdef1234567890",
			expectError: true,
			errContains: "invalid address length",
		},
		{
			name:        "invalid hex character",
			input:       "0x1234567890abcdef1234567890abcdef1234567g",
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expectError: true,
			errContains: "invalid address length",
		},
		{
			name:        "only prefix",
			input:       "0x",
			expectError: true,
			errContains: "invalid address length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := HexToAddress(tt.input)
			if tt.expectError {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, addr.Hex())
			}
		})
	}
}

func TestBytesToAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "exact 20 bytes",
			input:    []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			expected: "0x0102030405060708090a0b0c0d0e0f1011121314",
		},
		{
			name:     "shorter than 20 bytes (right-aligned)",
			input:    []byte{1, 2, 3, 4},
			expected: "0x0000000000000000000000000000000001020304",
		},
		{
			name:     "longer than 20 bytes (left-truncated)",
			input:    []byte{0xff, 0xff, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			expected: "0x0102030405060708090a0b0c0d0e0f1011121314",
		},
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: "0x0000000000000000000000000000000000000000",
		},
		{
			name:     "nil bytes",
			input:    nil,
			expected: "0x0000000000000000000000000000000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := BytesToAddress(tt.input)
			require.Equal(t, tt.expected, addr.Hex())
		})
	}
}

func TestAddressSetBytes(t *testing.T) {
	t.Run("shorter bytes are right-aligned with zero padding", func(t *testing.T) {
		addr, _ := HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
		addr.SetBytes([]byte{1, 2, 3, 4})
		// SetBytes zeros first, then copies right-aligned
		require.Equal(t, "0x0000000000000000000000000000000001020304", addr.Hex())
	})

	t.Run("exact length replaces all bytes", func(t *testing.T) {
		addr, _ := HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
		addr.SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20})
		require.Equal(t, "0x0102030405060708090a0b0c0d0e0f1011121314", addr.Hex())
	})
}

func TestAddressBytes(t *testing.T) {
	addr, _ := HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	b := addr.Bytes()

	require.Len(t, b, 20)
	require.Equal(t, byte(0x12), b[0])
	require.Equal(t, byte(0x78), b[19])

	// Verify it's a copy (modifying returned slice doesn't affect address)
	b[0] = 0xff
	require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", addr.Hex())
}

func TestAddressJSON(t *testing.T) {
	type wrapper struct {
		Addr Address `json:"addr"`
	}

	t.Run("marshal", func(t *testing.T) {
		addr, _ := HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		w := wrapper{Addr: addr}

		data, err := json.Marshal(w)
		require.NoError(t, err)
		require.Contains(t, string(data), `"addr":"0x1234567890abcdef1234567890abcdef12345678"`)
	})

	t.Run("unmarshal valid", func(t *testing.T) {
		jsonStr := `{"addr":"0x1234567890abcdef1234567890abcdef12345678"}`
		var w wrapper
		err := json.Unmarshal([]byte(jsonStr), &w)
		require.NoError(t, err)
		require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", w.Addr.Hex())
	})

	t.Run("unmarshal uppercase 0X rejected", func(t *testing.T) {
		jsonStr := `{"addr":"0X1234567890abcdef1234567890abcdef12345678"}`
		var w wrapper
		err := json.Unmarshal([]byte(jsonStr), &w)
		require.Error(t, err)
		require.Contains(t, err.Error(), "only lowercase 0x is accepted")
	})

	t.Run("unmarshal invalid length", func(t *testing.T) {
		jsonStr := `{"addr":"0x1234"}`
		var w wrapper
		err := json.Unmarshal([]byte(jsonStr), &w)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid address")
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		jsonStr := `{"addr":12345}`
		var w wrapper
		err := json.Unmarshal([]byte(jsonStr), &w)
		require.Error(t, err)
	})

	t.Run("round trip", func(t *testing.T) {
		original, _ := HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		w := wrapper{Addr: original}

		data, err := json.Marshal(w)
		require.NoError(t, err)

		var w2 wrapper
		err = json.Unmarshal(data, &w2)
		require.NoError(t, err)
		require.Equal(t, original, w2.Addr)
	})
}

func TestAddressJSON_Null(t *testing.T) {
	type wrapper struct {
		Addr Address `json:"addr"`
	}

	t.Run("null sets zero address", func(t *testing.T) {
		jsonStr := `{"addr":null}`
		var w wrapper
		// Set to non-zero first to verify it gets zeroed
		w.Addr[0] = 0xff
		err := json.Unmarshal([]byte(jsonStr), &w)
		require.NoError(t, err)
		require.True(t, w.Addr.IsZero())
	})
}

func TestAddressIsZero(t *testing.T) {
	t.Run("zero address", func(t *testing.T) {
		var addr Address
		require.True(t, addr.IsZero())
	})

	t.Run("non-zero address", func(t *testing.T) {
		addr, _ := HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
		require.False(t, addr.IsZero())
	})

	t.Run("address with single non-zero byte", func(t *testing.T) {
		var addr Address
		addr[19] = 1
		require.False(t, addr.IsZero())
	})

	t.Run("BytesToAddress empty is zero", func(t *testing.T) {
		addr := BytesToAddress([]byte{})
		require.True(t, addr.IsZero())
	})
}

func TestAddressString(t *testing.T) {
	addr, _ := HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	assert.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", addr.String())
}
