// Package types implements a 256-bit unsigned integer type.
package types

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
)

// Uint256 represents a 256-bit unsigned integer using 4 uint64 values.
// Words are stored in little-endian order (words[0] is least-significant).
//
// All arithmetic operations are performed modulo 2^256 unless stated otherwise.
type Uint256 [4]uint64

// maxDecimalDigits is the maximum number of decimal digits in a 256-bit number.
// Calculated as ceil(256 * log10(2)) = 78.
const maxDecimalDigits = 78

const hexDigits = "0123456789abcdef"

// NewUint256 creates a new Uint256 from a uint64 value.
func NewUint256(v uint64) *Uint256 {
	return &Uint256{v, 0, 0, 0}
}

// SetBytes interprets bytes as a big-endian unsigned integer.
// If b is longer than 32 bytes, only the last 32 bytes are used (left-truncation).
// This matches the semantics of (*big.Int).Bytes() round-trips, where leading
// zeros or a single high byte beyond 32 are safely discarded.
func (z *Uint256) SetBytes(b []byte) *Uint256 {
	*z = Uint256{}
	if b == nil {
		return z
	}
	if len(b) == 0 {
		return z
	}
	if len(b) > 32 {
		b = b[len(b)-32:]
	}

	var tmp [32]byte
	copy(tmp[32-len(b):], b)

	z[3] = binary.BigEndian.Uint64(tmp[0:8])
	z[2] = binary.BigEndian.Uint64(tmp[8:16])
	z[1] = binary.BigEndian.Uint64(tmp[16:24])
	z[0] = binary.BigEndian.Uint64(tmp[24:32])

	return z
}

// Bytes returns the value as a 32-byte big-endian slice.
// Leading zero bytes are included.
func (z Uint256) Bytes() []byte {
	var buf [32]byte
	binary.BigEndian.PutUint64(buf[0:8], z[3])
	binary.BigEndian.PutUint64(buf[8:16], z[2])
	binary.BigEndian.PutUint64(buf[16:24], z[1])
	binary.BigEndian.PutUint64(buf[24:32], z[0])
	return buf[:]
}

// SetHex parses a hex string with "0x" prefix into z.
// Returns an error if the prefix is missing, the string is invalid, or exceeds 256 bits.
func (z *Uint256) SetHex(s string) error {
	if len(s) < 2 || s[0] != '0' || s[1] != 'x' {
		return fmt.Errorf("invalid Uint256 prefix: only lowercase 0x is accepted")
	}
	s = s[2:]
	if len(s) == 0 {
		return fmt.Errorf("invalid Uint256 format: empty hex string after 0x prefix")
	}
	return z.parseHex(s)
}

// Add sets z = x + y (mod 2^256) and returns z.
func (z *Uint256) Add(x, y Uint256) *Uint256 {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], _ = bits.Add64(x[3], y[3], carry)
	return z
}

// AddOverflow sets z = x + y and reports whether overflow occurred.
func (z *Uint256) AddOverflow(x, y Uint256) (overflow bool) {
	var carry uint64
	z[0], carry = bits.Add64(x[0], y[0], 0)
	z[1], carry = bits.Add64(x[1], y[1], carry)
	z[2], carry = bits.Add64(x[2], y[2], carry)
	z[3], carry = bits.Add64(x[3], y[3], carry)
	return carry != 0
}

// Sub sets z = x - y (mod 2^256) and returns z.
func (z *Uint256) Sub(x, y Uint256) *Uint256 {
	var borrow uint64
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], _ = bits.Sub64(x[3], y[3], borrow)
	return z
}

// SubOverflow sets z = x - y and reports whether underflow occurred.
func (z *Uint256) SubOverflow(x, y Uint256) (underflow bool) {
	var borrow uint64
	z[0], borrow = bits.Sub64(x[0], y[0], 0)
	z[1], borrow = bits.Sub64(x[1], y[1], borrow)
	z[2], borrow = bits.Sub64(x[2], y[2], borrow)
	z[3], borrow = bits.Sub64(x[3], y[3], borrow)
	return borrow != 0
}

// Cmp compares z and y and returns:
//
//	-1 if z < y
//	 0 if z == y
//	+1 if z > y
func (z Uint256) Cmp(y Uint256) int {
	for i := 3; ; i-- {
		if z[i] > y[i] {
			return 1
		}
		if z[i] < y[i] {
			return -1
		}
		if i == 0 {
			break
		}
	}
	return 0
}

// Eq returns true if z == y.
func (z Uint256) Eq(y Uint256) bool {
	return z == y
}

// IsZero returns true if z == 0.
func (z Uint256) IsZero() bool {
	return (z[0] | z[1] | z[2] | z[3]) == 0
}

// String returns the decimal representation of z.
func (z Uint256) String() string {
	if z.IsZero() {
		return "0"
	}

	val := z
	res := make([]byte, 0, maxDecimalDigits)

	const ten = uint64(10)
	for !val.IsZero() {
		var rem uint64
		val, rem = val.divModWord(ten)
		res = append(res, byte(rem)+'0')
	}

	// reverse
	for i, j := 0, len(res)-1; i < j; i, j = i+1, j-1 {
		res[i], res[j] = res[j], res[i]
	}
	return string(res)
}

// divModWord computes z / divisor and z % divisor. We assume the divisor has been checked by the caller to be != 0
func (z Uint256) divModWord(divisor uint64) (Uint256, uint64) {
	var quot Uint256
	var r uint64

	for i := 3; ; i-- {
		q, rr := bits.Div64(r, z[i], divisor)
		quot[i] = q
		r = rr
		if i == 0 {
			break
		}
	}
	return quot, r
}

// ToHex returns the hex representation of z with "0x" prefix.
// Leading zeros are removed, except for zero values which return "0x0".
func (z Uint256) ToHex() string {
	if z.IsZero() {
		return "0x0"
	}

	// Preallocate buffer for "0x" + up to 64 hex digits
	buf := make([]byte, 2, 66)
	buf[0], buf[1] = '0', 'x'

	// Encode each nibble, skipping leading zeros
	leadingZero := true
	for i := 3; i >= 0; i-- {
		word := z[i]
		for j := 60; j >= 0; j -= 4 {
			b := byte((word >> j) & 0xf)
			if leadingZero && b == 0 {
				continue
			}
			leadingZero = false
			buf = append(buf, hexDigits[b])
		}
	}

	return string(buf)
}

// MarshalJSON implements json.Marshaler.
// It marshals the Uint256 as a hex string with 0x prefix.
func (z Uint256) MarshalJSON() ([]byte, error) {
	return json.Marshal(z.ToHex())
}

// UnmarshalJSON implements json.Unmarshaler.
// Accepts hex strings with "0x" prefix, or JSON null.
func (z *Uint256) UnmarshalJSON(data []byte) error {
	if z == nil {
		return fmt.Errorf("Uint256: UnmarshalJSON on nil pointer")
	}
	if string(data) == "null" {
		*z = Uint256{}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid Uint256 format: %w", err)
	}

	if len(s) < 2 || s[0] != '0' || s[1] != 'x' {
		return fmt.Errorf("invalid Uint256 prefix: %s (only lowercase 0x is accepted)", s)
	}

	s = s[2:]
	if len(s) == 0 {
		return fmt.Errorf("invalid Uint256 format: empty hex string after 0x prefix")
	}

	if err := z.parseHex(s); err != nil {
		return fmt.Errorf("invalid Uint256 hex: %w", err)
	}

	return nil
}

// parseHex parses a hex string (without prefix) into z.
// Returns an error if the string is invalid or exceeds 64 hex digits.
func (z *Uint256) parseHex(s string) error {
	if len(s) > 64 {
		return errors.New("hex string exceeds 256 bits")
	}

	*z = Uint256{}

	// Pad to 64 characters using a stack-allocated buffer
	var buf [64]byte
	for i := range buf {
		buf[i] = '0'
	}
	copy(buf[64-len(s):], s)

	// Parse each 16-character chunk as a uint64
	for i := range 4 {
		word, err := parseHexWord(string(buf[16*i : 16*(i+1)]))
		if err != nil {
			return err
		}
		z[3-i] = word
	}

	return nil
}

// parseHexWord parses exactly 16 hex characters into a uint64.
func parseHexWord(s string) (uint64, error) {
	var result uint64
	for i := range 16 {
		c := s[i]
		var nibble uint64
		switch {
		case c >= '0' && c <= '9':
			nibble = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			nibble = uint64(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			nibble = uint64(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
		result = (result << 4) | nibble
	}
	return result, nil
}

// Mul64 sets z = z * y (mod 2^256).
func (z *Uint256) Mul64(y uint64) {
	_ = z.Mul64Overflow(y)
}

// Mul64Overflow sets z = z * y and reports whether overflow occurred.
func (z *Uint256) Mul64Overflow(y uint64) (overflow bool) {
	var carry uint64
	for i := range 4 {
		hi, lo := bits.Mul64(z[i], y)
		var c uint64
		z[i], c = bits.Add64(lo, carry, 0)
		// Note: hi + c cannot overflow uint64 here because bits.Mul64(a, b)
		// has a maximum 'hi' value of 0xfffffffffffffffe (MaxUint64 - 1).
		// Since c is at most 1, the sum hi + c is at most MaxUint64.
		carry = hi + c
	}
	return carry != 0
}

// Add64 adds y to z (mod 2^256).
func (z *Uint256) Add64(y uint64) {
	_ = z.Add64Overflow(y)
}

// Add64Overflow adds y to z and reports whether overflow occurred.
func (z *Uint256) Add64Overflow(y uint64) (overflow bool) {
	var carry uint64
	z[0], carry = bits.Add64(z[0], y, 0)
	for i := 1; i < 4 && carry != 0; i++ {
		z[i], carry = bits.Add64(z[i], 0, carry)
	}
	return carry != 0
}
