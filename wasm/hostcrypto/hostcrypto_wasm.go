// TinyGo+WASI build: route SHA-256 and HMAC-SHA-512 through host imports
// provided by vela's executor. See the package doc for the underlying TinyGo
// + Go-1.24 FIPS-stdlib lifecycle issue this bridge sidesteps.

//go:build tinygo.wasm

package hostcrypto

import "unsafe"

// hostSHA256 is implemented by vela/pkg/wasm/wasmtime_runtime.go via
// linker.DefineFunc("env", "host_sha256", ...). Reads inLen bytes at inPtr,
// writes the 32-byte digest at outPtr.
//
//go:wasmimport env host_sha256
func hostSHA256(inPtr unsafe.Pointer, inLen uint32, outPtr unsafe.Pointer)

// hostHMACSHA512 is implemented by vela/pkg/wasm/wasmtime_runtime.go via
// linker.DefineFunc("env", "host_hmac_sha512", ...). Reads keyLen bytes at
// keyPtr and msgLen bytes at msgPtr, writes the 64-byte HMAC-SHA-512 digest
// at outPtr.
//
//go:wasmimport env host_hmac_sha512
func hostHMACSHA512(keyPtr unsafe.Pointer, keyLen uint32, msgPtr unsafe.Pointer, msgLen uint32, outPtr unsafe.Pointer)

func sha256Sum(data []byte) [32]byte {
	var out [32]byte
	// Empty input: handled by the host (sha256 of empty has a well-known
	// digest). We still need a non-nil pointer to satisfy WASM ABI; point at
	// out itself with length 0 — the host reads zero bytes.
	if len(data) == 0 {
		hostSHA256(unsafe.Pointer(&out[0]), 0, unsafe.Pointer(&out[0]))
		return out
	}
	hostSHA256(unsafe.Pointer(&data[0]), uint32(len(data)), unsafe.Pointer(&out[0]))
	return out
}

func hmacSHA512Sum(key, msg []byte) [64]byte {
	var out [64]byte
	// Empty key or msg: same non-nil-pointer trick as above. The host
	// implementation must handle zero-length key / msg correctly (RFC 2104
	// permits both).
	var keyPtr, msgPtr unsafe.Pointer
	if len(key) > 0 {
		keyPtr = unsafe.Pointer(&key[0])
	} else {
		keyPtr = unsafe.Pointer(&out[0])
	}
	if len(msg) > 0 {
		msgPtr = unsafe.Pointer(&msg[0])
	} else {
		msgPtr = unsafe.Pointer(&out[0])
	}
	hostHMACSHA512(keyPtr, uint32(len(key)), msgPtr, uint32(len(msg)), unsafe.Pointer(&out[0]))
	return out
}
