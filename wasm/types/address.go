package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const AddressLength = 20

type Address [AddressLength]byte

// HexToAddress converts a hex string to Address with validation.
func HexToAddress(s string) (Address, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
	}
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

func (a *Address) SetBytes(b []byte) {
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

func (a Address) Hex() string {
	return "0x" + hex.EncodeToString(a[:])
}

func (a *Address) UnmarshalJSON(data []byte) error {
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
