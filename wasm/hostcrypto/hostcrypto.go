// Package hostcrypto exposes a build-tag-gated facade for the hash
// primitives Vela apps need from inside a TinyGo+WASI guest. The WASM build
// routes calls through host imports (env::host_sha256, env::host_hmac_sha512)
// implemented by vela's executor; native builds delegate to the Go stdlib
// directly so the same package compiles cleanly on the host side (vela's
// Scheduler, manager, integration tests).
//
// Why the bridge exists: Go 1.24's stdlib crypto packages depend on
// `crypto/internal/fips140`'s per-goroutine indicator state, which lives in
// TinyGo's runtime scheduler and is only populated by `_start`. vela's
// executor invokes named exports (process_request, deposit, ...) via
// wasmtime-go's Linker.Instantiate WITHOUT calling _start, so the FIPS
// indicator-init never runs and the first crypto call nil-derefs. The host
// bridge sidesteps this by performing the hash natively (no stdlib crypto
// inside the WASM guest), keeping vela apps on TinyGo without the runtime-
// lifecycle fight. The spike at vela-ned/spikes/007-wasmtime-go-init/ is the
// regression artifact.
package hostcrypto

// SHA256 returns the SHA-256 digest of data.
func SHA256(data []byte) [32]byte {
	return sha256Sum(data)
}

// HMACSHA512 returns the 64-byte HMAC-SHA-512 of msg with key.
func HMACSHA512(key, msg []byte) [64]byte {
	return hmacSHA512Sum(key, msg)
}
