// Package common provides shared type definitions used across the Horizen
// Confidential Compute Environment.
package common

import (
	"encoding/hex"
	"fmt"
)

// ApplicationIdType represents a unique application identifier.
type ApplicationIdType uint64

// NewApplicationId creates an ApplicationIdType from a uint64 value.
func NewApplicationId(id uint64) ApplicationIdType {
	return ApplicationIdType(id)
}

func (aid ApplicationIdType) String() string {
	return fmt.Sprintf("%d", uint64(aid))
}

// RequestIdType represents a 32-byte request identifier.
type RequestIdType [32]byte

func (rt RequestIdType) String() string {
	return hex.EncodeToString(rt[:])
}

func (rt RequestIdType) MarshalJSON() ([]byte, error) {
	s := hex.EncodeToString(rt[:])
	return []byte(`"0x` + s + `"`), nil
}

func (rt *RequestIdType) UnmarshalJSON(data []byte) error {
	// data is expected to be a hex string with a "0x" prefix in quotes representing an array of exactly 32 bytes
	// e.g. "0xab12...a8" (68 chars in total, prefix and start-end quotes included)
	if len(data) != 68 || data[0] != '"' || data[1] != '0' || data[2] != 'x' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid RequestIdType format")
	}

	b, err := hex.DecodeString(string(data[3 : len(data)-1]))
	if err != nil {
		return err
	}
	if len(b) != 32 {
		return fmt.Errorf("invalid RequestIdType length")
	}
	copy(rt[:], b)
	return nil
}

// RequestResultStatus represents the final status of a request after execution.
type RequestResultStatus uint8

const (
	RequestResultOK RequestResultStatus = iota
	RequestResultFailed
	RequestResultUnknown
)
