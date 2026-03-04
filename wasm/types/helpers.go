package types

import (
	"encoding/json"
	"unsafe"

	"github.com/HorizenOfficial/vela-common-go/wasm/utils"
)

// SerializeAndWriteResult handles common serialization and returns a WASM pointer.
func SerializeAndWriteResult(result any) *byte {
	reportJSON, err := json.Marshal(result)
	if err != nil {
		return utils.BytesToPtr([]byte(WasmSerializationError))
	}
	return utils.BytesToPtr(reportJSON)
}

// PtrToUint256 converts a WASM pointer and length into a Uint256.
//
// The input bytes must come from (*big.Int).Bytes(), i.e.:
//   - big-endian
//   - absolute value only (non-negative)
//
// Semantics:
//   - (nil, 0) represents the value 0
//   - Any other (ptr, length) combination with ptr == nil or length < 0 is invalid
//   - If length > 32, a warning is logged and the input is truncated before passing to SetBytes
//
// The function returns nil on invalid input.
func PtrToUint256(ptr *byte, length int32) *Uint256 {
	// Validate length
	if length < 0 {
		return nil
	}

	// (nil, 0) => zero
	if ptr == nil && length == 0 {
		return NewUint256(0)
	}

	// Any other nil/non-nil mismatch is invalid
	if ptr == nil || length == 0 {
		return nil
	}

	// just to be on the very safe side and avoid panics. Should never happen
	if length > MaxBigIntBytes {
		utils.LogWarn("Unexpected length for a big.Int ptr mem: truncating from %d to %d", length, MaxBigIntBytes)
		length = MaxBigIntBytes
	}

	return new(Uint256).SetBytes(unsafe.Slice(ptr, length))
}

// PtrToAddress converts a WASM pointer and length to a ethereum address.
func PtrToAddress(ptr *byte, length int32) *Address {
	if ptr == nil || length <= 0 || length > MaxAddressBytes {
		return nil
	}
	var address Address
	address.SetBytes(unsafe.Slice(ptr, length))
	return &address
}
