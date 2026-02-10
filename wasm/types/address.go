package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

const AddressLength = 20

type Address [AddressLength]byte

// HexToAddress converts a hex string to Address with validation.
// Only lowercase "0x" prefix is accepted for consistency with Uint256.
func HexToAddress(s string) (Address, error) {
	if len(s) < 2 || s[0] != '0' || s[1] != 'x' {
		return Address{}, fmt.Errorf("invalid address prefix: only lowercase 0x is accepted")
	}
	s = s[2:]
	if len(s) != AddressLength*2 {
		return Address{}, fmt.Errorf("invalid address length: got %d hex chars, want %d", len(s), AddressLength*2)
	}

	data, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, err
	}

	var address Address
	copy(address[:], data)
	return address, nil
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// SetBytes sets the address from a byte slice.
// If b is longer than 20 bytes, only the last 20 bytes are used (left-truncation).
// If b is shorter than 20 bytes, it is right-aligned (zero-padded on the left).
func (a *Address) SetBytes(b []byte) {
	*a = Address{}
	if len(b) > AddressLength {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Bytes returns a copy of address bytes (safe, immutable to caller)
func (a Address) Bytes() []byte {
	b := make([]byte, AddressLength)
	copy(b, a[:])
	return b
}

// IsZero returns true if the address is all zeros.
func (a Address) IsZero() bool {
	return a == Address{}
}

func (a Address) Hex() string {
	return "0x" + hex.EncodeToString(a[:])
}

func (a *Address) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*a = Address{}
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	addr, err := HexToAddress(s)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	*a = addr
	return nil
}

func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Hex())
}

func (a Address) String() string {
	return a.Hex()
}
