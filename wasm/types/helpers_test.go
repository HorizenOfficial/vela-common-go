package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Note: SerializeAndWriteResult relies on WASM memory functions that use 32-bit pointers.
// These cannot be properly tested in native 64-bit Go. The function is tested indirectly
// through WASM module integration tests.

func TestPtrToUint256(t *testing.T) {
	tests := []struct {
		name           string
		inputBytes     []byte
		length         int32
		expectedNil    bool
		expectedString string
	}{
		{
			name:           "nil ptr with zero length returns zero",
			inputBytes:     nil,
			length:         0,
			expectedNil:    false,
			expectedString: "0",
		},
		{
			name:        "nil ptr with non-zero length returns nil",
			inputBytes:  nil,
			length:      10,
			expectedNil: true,
		},
		{
			name:        "negative length returns nil",
			inputBytes:  []byte{1, 2, 3},
			length:      -1,
			expectedNil: true,
		},
		{
			name:        "non-nil ptr with zero length returns nil",
			inputBytes:  []byte{1, 2, 3},
			length:      0,
			expectedNil: true,
		},
		{
			name:           "valid single byte",
			inputBytes:     []byte{0xff},
			length:         1,
			expectedNil:    false,
			expectedString: "255",
		},
		{
			name:           "valid multiple bytes big-endian",
			inputBytes:     []byte{0x01, 0x00}, // 256 in big-endian
			length:         2,
			expectedNil:    false,
			expectedString: "256",
		},
		{
			name:           "32 bytes (max without truncation)",
			inputBytes:     make([]byte, 32),
			length:         32,
			expectedNil:    false,
			expectedString: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ptr *byte
			if tt.inputBytes != nil {
				ptr = &tt.inputBytes[0]
			}

			result := PtrToUint256(ptr, tt.length)

			if tt.expectedNil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tt.expectedString, result.String())
			}
		})
	}

	t.Run("large value", func(t *testing.T) {
		// big.Int bytes for 2^64 (18446744073709551616)
		bytes := []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		result := PtrToUint256(&bytes[0], int32(len(bytes)))
		require.NotNil(t, result)
		require.Equal(t, "18446744073709551616", result.String())
	})
}

func TestPtrToAddress(t *testing.T) {
	tests := []struct {
		name        string
		inputBytes  []byte
		length      int32
		expectedNil bool
		expectedHex string
	}{
		{
			name:        "nil ptr returns nil",
			inputBytes:  nil,
			length:      20,
			expectedNil: true,
		},
		{
			name:        "zero length returns nil",
			inputBytes:  make([]byte, 20),
			length:      0,
			expectedNil: true,
		},
		{
			name:        "negative length returns nil",
			inputBytes:  make([]byte, 20),
			length:      -1,
			expectedNil: true,
		},
		{
			name:        "length exceeds max returns nil",
			inputBytes:  make([]byte, 25),
			length:      25,
			expectedNil: true,
		},
		{
			name:        "valid 20 bytes",
			inputBytes:  []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			length:      20,
			expectedNil: false,
			expectedHex: "0x0102030405060708090a0b0c0d0e0f1011121314",
		},
		{
			name:        "valid shorter than 20 bytes (right-aligned)",
			inputBytes:  []byte{0xde, 0xad, 0xbe, 0xef},
			length:      4,
			expectedNil: false,
			expectedHex: "0x00000000000000000000000000000000deadbeef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ptr *byte
			if tt.inputBytes != nil {
				ptr = &tt.inputBytes[0]
			}

			result := PtrToAddress(ptr, tt.length)

			if tt.expectedNil {
				require.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, tt.expectedHex, result.Hex())
			}
		})
	}
}

func TestResultTypesJSON(t *testing.T) {
	t.Run("DepositResult", func(t *testing.T) {
		depositSubType := [32]byte{0x01, 0x02}
		depositReceivedSubType := [32]byte{0x03, 0x04}
		result := DepositResult{
			State: []byte("state"),
			Events: []PlainEvent{
				{
					UserID:       BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}),
					EventSubType: depositSubType,
					Data:         []byte("event data"),
				},
			},
			AppEvents: []AppEvent{
				{
					EventSubType: depositReceivedSubType,
					Data:         []byte(`{"tokenAddress":"0x0000","amount":"0x3e8"}`),
				},
			},
			Fuel:  NewUint256(100),
			Error: "",
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed DepositResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		require.Len(t, parsed.Events, 1)
		require.Equal(t, depositSubType, parsed.Events[0].EventSubType)
		require.Len(t, parsed.AppEvents, 1)
		require.Equal(t, depositReceivedSubType, parsed.AppEvents[0].EventSubType)
		require.Equal(t, []byte(`{"tokenAddress":"0x0000","amount":"0x3e8"}`), parsed.AppEvents[0].Data)
	})

	t.Run("DepositResult_NoAppEvents", func(t *testing.T) {
		result := DepositResult{
			State: []byte("state"),
			Fuel:  NewUint256(100),
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed DepositResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		require.Empty(t, parsed.AppEvents)
	})

	t.Run("ProcessResult", func(t *testing.T) {
		transferReceiptSubType := [32]byte{0x05, 0x06}
		result := ProcessResult{
			State:  []byte("state"),
			Events: []PlainEvent{},
			AppEvents: []AppEvent{
				{
					EventSubType: transferReceiptSubType,
					Data:         []byte{0xab, 0xcd},
				},
			},
			Withdrawals: []Withdrawal{
				{
					DestinationAddress: BytesToAddress([]byte{0xde, 0xad, 0xbe, 0xef}),
					Amount:             NewUint256(1000000),
				},
			},
			Report: []byte("report data"),
			Fuel:   NewUint256(200),
			Error:  "",
		}

		data, err := json.Marshal(result)
		require.NoError(t, err)

		var parsed ProcessResult
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		require.Len(t, parsed.Withdrawals, 1)
		require.Equal(t, "1000000", parsed.Withdrawals[0].Amount.String())
		require.Len(t, parsed.AppEvents, 1)
		require.Equal(t, transferReceiptSubType, parsed.AppEvents[0].EventSubType)
		require.Equal(t, []byte{0xab, 0xcd}, parsed.AppEvents[0].Data)
	})

}
