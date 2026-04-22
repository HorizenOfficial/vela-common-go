package common

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationIdType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    ApplicationIdType
		expected string
	}{
		{"zero", 0, `"0"`},
		{"small", 42, `"42"`},
		{"below 2^53", ApplicationIdType(1<<53 - 1), `"9007199254740991"`},
		{"above 2^53", ApplicationIdType(1<<53 + 1), `"9007199254740993"`},
		{"bug report value", ApplicationIdType(4270397216330557770), `"4270397216330557770"`},
		{"max uint64", ApplicationIdType(math.MaxUint64), `"18446744073709551615"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(got))
		})
	}
}

func TestApplicationIdType_UnmarshalJSON_QuotedString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ApplicationIdType
	}{
		{"zero", `"0"`, 0},
		{"small", `"42"`, 42},
		{"above 2^53", `"9007199254740993"`, ApplicationIdType(1<<53 + 1)},
		{"bug report value", `"4270397216330557770"`, ApplicationIdType(4270397216330557770)},
		{"max uint64", `"18446744073709551615"`, ApplicationIdType(math.MaxUint64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ApplicationIdType
			err := json.Unmarshal([]byte(tt.input), &got)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestApplicationIdType_UnmarshalJSON_RawNumber(t *testing.T) {
	// Backward compatibility: accept raw JSON numbers.
	tests := []struct {
		name     string
		input    string
		expected ApplicationIdType
	}{
		{"zero", `0`, 0},
		{"small", `42`, 42},
		{"below 2^53", `9007199254740991`, ApplicationIdType(1<<53 - 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ApplicationIdType
			err := json.Unmarshal([]byte(tt.input), &got)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestApplicationIdType_UnmarshalJSON_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", `""`},
		{"negative", `"-1"`},
		{"overflow", `"18446744073709551616"`},
		{"non-numeric", `"abc"`},
		{"hex", `"0xff"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ApplicationIdType
			err := json.Unmarshal([]byte(tt.input), &got)
			assert.Error(t, err)
		})
	}
}

func TestApplicationIdType_RoundTrip(t *testing.T) {
	// The exact value from the bug report: 4270397216330557770 (0x3b437fa48933294a).
	// This value is above 2^53 and would lose precision through float64.
	original := ApplicationIdType(4270397216330557770)

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded ApplicationIdType
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func TestApplicationIdType_RoundTripInStruct(t *testing.T) {
	type request struct {
		ApplicationID ApplicationIdType `json:"applicationId"`
		Name          string            `json:"name"`
	}

	original := request{
		ApplicationID: ApplicationIdType(4270397216330557770),
		Name:          "test",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Verify the JSON contains a quoted string, not a raw number.
	assert.Contains(t, string(data), `"applicationId":"4270397216330557770"`)

	var decoded request
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}
