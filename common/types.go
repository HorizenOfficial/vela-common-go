// Package common provides shared type definitions used across the Horizen
// Confidential Compute Environment.
package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
)

// ConstructorParams is an alias for raw JSON constructor parameters passed to the guest deploy function.
type ConstructorParams = json.RawMessage

// ApplicationIdType represents a unique application identifier.
type ApplicationIdType uint64

// NewApplicationId creates an ApplicationIdType from a uint64 value.
func NewApplicationId(id uint64) ApplicationIdType {
	return ApplicationIdType(id)
}

func (aid ApplicationIdType) String() string {
	return fmt.Sprintf("%d", uint64(aid))
}

// MarshalJSON serializes as a quoted decimal string to avoid float64 precision
// loss for values above 2^53 when decoded by standard JSON parsers.
func (aid ApplicationIdType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(aid), 10) + `"`), nil
}

// UnmarshalJSON accepts both a quoted decimal string ("123") and a raw JSON
// number (123) for backward compatibility. In both cases the value is parsed
// with strconv.ParseUint, avoiding the float64 intermediate that loses
// precision for large uint64 values.
func (aid *ApplicationIdType) UnmarshalJSON(data []byte) error {
	s := string(data)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ApplicationIdType: %w", err)
	}
	*aid = ApplicationIdType(v)
	return nil
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

// DeployModeArtifactRef is the deploy descriptor mode for off-chain artifact reference.
const DeployModeArtifactRef = "artifact_ref"

// DeployDescriptor defines the v1 deploy payload contract stored in Request.Payload.
// This is the wire protocol shared between the wallet (producer) and the framework (consumer).
type DeployDescriptor struct {
	Mode              string            `json:"mode"`
	ArtifactID        string            `json:"artifactId"`
	WasmSHA256        string            `json:"wasmSha256"`
	ConstructorParams ConstructorParams `json:"constructorParams,omitempty"`
}
