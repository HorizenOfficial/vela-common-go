// Package bip32 implements the subset of BIP-32 needed by vela-ned: non-hardened
// public-side child key derivation (CKDpub), BIP-32 Base58Check xpub parsing,
// and derivation of Ethereum-style addresses from child compressed public keys.
//
// The package is pure Go and compiles under TinyGo (no CGo, no unsafe). It
// depends on:
//   - crypto/hmac + crypto/sha512    (stdlib, BIP-32 derivation)
//   - crypto/sha256                  (stdlib, Base58Check checksum)
//   - dcrec/secp256k1/v4             (pure-Go secp256k1 — TinyGo-verified by spike 005)
//   - golang.org/x/crypto/sha3       (pure-Go Keccak-256 for Ethereum addresses)
//
// Scope: non-hardened CKDpub only. Hardened derivation (index >= 2^31) requires
// the parent private key and is intentionally out of scope — vela-ned's WASM
// guest only ever holds public material.
//
// Reference: https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
package bip32

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// ExtendedKey is a BIP-32 extended public key.
//
// PubKey is the SEC1-compressed (33-byte) secp256k1 public key.
// ChainCode is the 32-byte chain code used to derive children.
// Depth, ChildNumber, ParentFP, Version are metadata populated by ParseXpub
// and propagated through CKDpub for serialization fidelity. CKDpub does not
// recompute ParentFP on derived children (it would require RIPEMD-160 and is
// not needed for sub-address derivation); String() on a CKDpub-derived child
// therefore emits a zero ParentFP. Re-serializing parsed parents is exact.
type ExtendedKey struct {
	PubKey      [33]byte
	ChainCode   [32]byte
	Depth       uint8
	ChildNumber uint32
	ParentFP    [4]byte
	Version     [4]byte
}

// Standard BIP-32 mainnet xpub version bytes.
var MainnetXpubVersion = [4]byte{0x04, 0x88, 0xB2, 0x1E}

// HardenedKeyOffset is the BIP-32 child-index threshold above which derivation
// is hardened (requires the parent private key). CKDpub rejects indices at or
// above this value.
const HardenedKeyOffset uint32 = 1 << 31

var (
	// ErrHardenedFromPublic is returned when CKDpub is called with an index
	// >= 2^31 (hardened range). Public-side derivation cannot produce
	// hardened children by BIP-32 definition.
	ErrHardenedFromPublic = errors.New("bip32: cannot derive hardened child from public key")

	// ErrInvalidDerivation indicates the derived intermediate scalar is
	// invalid (zero or >= curve order, or the resulting point is at
	// infinity). BIP-32 recommends "proceed with the next value of i";
	// the caller chooses how to handle.
	ErrInvalidDerivation = errors.New("bip32: invalid derived child (proceed with next index)")
)

// CKDpub derives a non-hardened child public key from a parent extended public
// key per BIP-32 §"Public parent key → public child key". The child's
// PubKey and ChainCode are set; Depth is parent.Depth + 1; ChildNumber is the
// input index; ParentFP is left as zeros (see ExtendedKey doc).
func CKDpub(parent ExtendedKey, index uint32) (ExtendedKey, error) {
	if index >= HardenedKeyOffset {
		return ExtendedKey{}, ErrHardenedFromPublic
	}

	// I = HMAC-SHA512(Key = parent.ChainCode, Data = parent.PubKey || ser32(i))
	mac := hmac.New(sha512.New, parent.ChainCode[:])
	mac.Write(parent.PubKey[:])
	var idxBE [4]byte
	binary.BigEndian.PutUint32(idxBE[:], index)
	mac.Write(idxBE[:])
	I := mac.Sum(nil)

	IL, IR := I[:32], I[32:]

	// Parse IL as a scalar mod n. Reject overflow (IL >= n) or zero.
	var ilScalar secp256k1.ModNScalar
	overflow := ilScalar.SetByteSlice(IL)
	if overflow || ilScalar.IsZero() {
		return ExtendedKey{}, ErrInvalidDerivation
	}

	// Parse parent pubkey as a curve point.
	parentPub, err := secp256k1.ParsePubKey(parent.PubKey[:])
	if err != nil {
		return ExtendedKey{}, fmt.Errorf("bip32: parse parent pubkey: %w", err)
	}

	// childPoint = IL*G + parentPoint
	var ilG, parentJ, childJ secp256k1.JacobianPoint
	secp256k1.ScalarBaseMultNonConst(&ilScalar, &ilG)
	parentPub.AsJacobian(&parentJ)
	secp256k1.AddNonConst(&ilG, &parentJ, &childJ)

	// Reject point-at-infinity (Z == 0 after addition).
	if childJ.Z.IsZero() {
		return ExtendedKey{}, ErrInvalidDerivation
	}

	childJ.ToAffine()
	childPub := secp256k1.NewPublicKey(&childJ.X, &childJ.Y)

	child := ExtendedKey{
		Depth:       parent.Depth + 1,
		ChildNumber: index,
		Version:     parent.Version,
	}
	copy(child.PubKey[:], childPub.SerializeCompressed())
	copy(child.ChainCode[:], IR)

	return child, nil
}
