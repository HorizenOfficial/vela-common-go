// Native (non-TinyGo-WASM) build: delegate to stdlib crypto directly.
// Used by host-side consumers (Scheduler, manager, integration tests) and
// for any non-WASM unit-test path that exercises bip32 / app code natively.

//go:build !tinygo.wasm

package hostcrypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
)

func sha256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func hmacSHA512Sum(key, msg []byte) [64]byte {
	mac := hmac.New(sha512.New, key)
	mac.Write(msg)
	var out [64]byte
	copy(out[:], mac.Sum(nil))
	return out
}
