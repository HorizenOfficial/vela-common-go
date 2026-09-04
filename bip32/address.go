package bip32

import (
	"fmt"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"

	wasmtypes "github.com/HorizenOfficial/vela-common-go/wasm/types"
)

// CompressedPubkeyToAddress derives an Ethereum-style address from a 33-byte
// SEC1-compressed secp256k1 public key. The address is the last 20 bytes of
// Keccak-256 of the 64-byte uncompressed pubkey (without the 0x04 prefix).
// Returns wasmtypes.Address — the canonical address type used across
// vela-common-go consumers.
func CompressedPubkeyToAddress(pubKey [33]byte) (wasmtypes.Address, error) {
	pub, err := secp256k1.ParsePubKey(pubKey[:])
	if err != nil {
		return wasmtypes.Address{}, fmt.Errorf("bip32: parse pubkey: %w", err)
	}

	uncompressed := pub.SerializeUncompressed()
	if len(uncompressed) != 65 || uncompressed[0] != 0x04 {
		return wasmtypes.Address{}, fmt.Errorf("bip32: unexpected uncompressed pubkey shape (len=%d, prefix=0x%02x)", len(uncompressed), uncompressed[0])
	}

	h := sha3.NewLegacyKeccak256()
	h.Write(uncompressed[1:])
	digest := h.Sum(nil)

	var addr wasmtypes.Address
	copy(addr[:], digest[len(digest)-wasmtypes.AddressLength:])
	return addr, nil
}

// DeriveAddress is a convenience wrapper: derive the non-hardened child at
// `index` from `parent`, then convert its public key to an Ethereum address.
// Returns the same errors as CKDpub and CompressedPubkeyToAddress.
func DeriveAddress(parent ExtendedKey, index uint32) (wasmtypes.Address, error) {
	child, err := CKDpub(parent, index)
	if err != nil {
		return wasmtypes.Address{}, err
	}
	return CompressedPubkeyToAddress(child.PubKey)
}
