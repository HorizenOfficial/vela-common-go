package bip32

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
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
// ExtendedKey. It verifies the 4-byte double-SHA256 checksum and the
// 33-byte SEC1-compressed pubkey shape (first byte must be 0x02 or 0x03).
// It does not enforce a specific Version — callers that require mainnet xpub
// can compare against MainnetXpubVersion after parsing.
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
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}
