package bip32

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/HorizenOfficial/vela-common-go/wasm/hostcrypto"
)

// Base58Check (Bitcoin) alphabet — note the deliberate absence of 0, O, I, l.
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// xpubPayloadLen is the BIP-32 extended-key serialized payload length:
// version(4) | depth(1) | parentFP(4) | childNumber(4) | chainCode(32) | pubKey(33) = 78.
const xpubPayloadLen = 78

// ErrInvalidXpub is the umbrella error returned by ParseXpub for any
// malformed input (bad characters, wrong length, bad checksum, bad pubkey).
// The wrapped underlying error names the specific failure.
var ErrInvalidXpub = errors.New("bip32: invalid xpub")

// ParseXpub decodes a BIP-32 Base58Check-encoded extended public key into an
// ExtendedKey. It verifies the 4-byte double-SHA256 checksum, the SEC1
// compressed-pubkey prefix (0x02 or 0x03), and that the pubkey bytes
// decompress to a valid point on the secp256k1 curve (fail-fast — without
// this, off-curve junk only surfaces later in CKDpub / address derivation).
// It does not enforce a specific Version — callers that require mainnet
// xpub can compare against MainnetXpubVersion after parsing.
func ParseXpub(s string) (ExtendedKey, error) {
	raw, err := base58Decode(s)
	if err != nil {
		return ExtendedKey{}, fmt.Errorf("%w: %v", ErrInvalidXpub, err)
	}
	if len(raw) != xpubPayloadLen+4 {
		return ExtendedKey{}, fmt.Errorf("%w: decoded length %d, want %d", ErrInvalidXpub, len(raw), xpubPayloadLen+4)
	}

	payload, checksum := raw[:xpubPayloadLen], raw[xpubPayloadLen:]
	want := doubleSHA256(payload)[:4]
	if !bytes.Equal(checksum, want) {
		return ExtendedKey{}, fmt.Errorf("%w: checksum mismatch", ErrInvalidXpub)
	}

	var k ExtendedKey
	copy(k.Version[:], payload[0:4])
	k.Depth = payload[4]
	copy(k.ParentFP[:], payload[5:9])
	k.ChildNumber = uint32(payload[9])<<24 | uint32(payload[10])<<16 | uint32(payload[11])<<8 | uint32(payload[12])
	copy(k.ChainCode[:], payload[13:45])
	copy(k.PubKey[:], payload[45:78])

	if k.PubKey[0] != 0x02 && k.PubKey[0] != 0x03 {
		return ExtendedKey{}, fmt.Errorf("%w: pubkey prefix 0x%02x is not compressed (want 0x02 or 0x03)", ErrInvalidXpub, k.PubKey[0])
	}
	if _, err := secp256k1.ParsePubKey(k.PubKey[:]); err != nil {
		return ExtendedKey{}, fmt.Errorf("%w: pubkey not on curve: %v", ErrInvalidXpub, err)
	}

	return k, nil
}

// base58Decode decodes a Base58 string into raw bytes using the Bitcoin
// alphabet. Leading '1' characters in the input become leading 0x00 bytes in
// the output (canonical Base58 convention).
func base58Decode(s string) ([]byte, error) {
	leadingOnes := 0
	for leadingOnes < len(s) && s[leadingOnes] == '1' {
		leadingOnes++
	}

	digits := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		idx := indexInAlphabet(s[i])
		if idx < 0 {
			return nil, fmt.Errorf("invalid base58 char %q at position %d", s[i], i)
		}
		carry := uint32(idx)
		for j := range digits {
			carry += uint32(digits[j]) * 58
			digits[j] = byte(carry & 0xff)
			carry >>= 8
		}
		for carry > 0 {
			digits = append(digits, byte(carry&0xff))
			carry >>= 8
		}
	}

	// digits is little-endian; reverse and prepend zeros for leading '1's.
	out := make([]byte, leadingOnes+len(digits))
	for i, b := range digits {
		out[len(out)-1-i] = b
	}
	return out, nil
}

func indexInAlphabet(c byte) int {
	for i := 0; i < len(base58Alphabet); i++ {
		if base58Alphabet[i] == c {
			return i
		}
	}
	return -1
}

func doubleSHA256(b []byte) []byte {
	first := hostcrypto.SHA256(b)
	second := hostcrypto.SHA256(first[:])
	return second[:]
}

// Serialize encodes an ExtendedKey to its BIP-32 Base58Check string form
// (the canonical `xpub6…` representation). The output is the exact inverse
// of ParseXpub for any well-formed input: ParseXpub(Serialize(k)) == k.
func (k ExtendedKey) Serialize() string {
	payload := make([]byte, 0, xpubPayloadLen)
	payload = append(payload, k.Version[:]...)
	payload = append(payload, k.Depth)
	payload = append(payload, k.ParentFP[:]...)
	var childBE [4]byte
	binary.BigEndian.PutUint32(childBE[:], k.ChildNumber)
	payload = append(payload, childBE[:]...)
	payload = append(payload, k.ChainCode[:]...)
	payload = append(payload, k.PubKey[:]...)

	cksum := doubleSHA256(payload)[:4]
	full := append(payload, cksum...)
	return base58Encode(full)
}

// base58Encode is the inverse of base58Decode using the Bitcoin alphabet.
// Leading 0x00 bytes in the input produce leading '1' characters in the
// output (canonical Base58 convention).
func base58Encode(b []byte) string {
	zeros := 0
	for zeros < len(b) && b[zeros] == 0 {
		zeros++
	}

	src := append([]byte{}, b...)
	var encoded []byte
	for {
		allZero := true
		for _, x := range src {
			if x != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			break
		}
		var rem uint32
		for i, x := range src {
			acc := rem*256 + uint32(x)
			src[i] = byte(acc / 58)
			rem = acc % 58
		}
		encoded = append(encoded, base58Alphabet[rem])
	}

	for i := 0; i < zeros; i++ {
		encoded = append(encoded, '1')
	}
	for i, j := 0, len(encoded)-1; i < j; i, j = i+1, j-1 {
		encoded[i], encoded[j] = encoded[j], encoded[i]
	}
	return string(encoded)
}
