package types

const (
	MaxBigIntBytes  = 32
	MaxAddressBytes = AddressLength
)

// PlainEvent is a local replacement for common.PlainEvent.
//
// RecipientPubKey is an optional override for the executor's recipient-key
// resolution. When non-empty, the host's encryptEvents parses it as a P521
// SEC1-uncompressed public key (133 bytes, prefix 0x04) and uses it as the
// ECIES receiver, ignoring the standard keyStore[UserID] lookup. When nil
// or empty, the existing keyStore lookup runs unchanged. The field carries
// json omitempty semantics — events that don't set it serialize identically
// to the pre-field shape.
type PlainEvent struct {
	UserID          Address  `json:"userId"`
	EventSubType    [32]byte `json:"eventSubType"`
	Data            []byte   `json:"data"`
	RecipientPubKey []byte   `json:"recipientPubKey,omitempty"`
}

// AppEvent is a local replacement for common.AppEvent (application-level, non-encrypted event)
type AppEvent struct {
	EventSubType [32]byte `json:"eventSubType"`
	Data         []byte   `json:"data"`
}

// Withdrawal is a local replacement for common.Withdrawal
type Withdrawal struct {
	TokenAddress       Address  `json:"tokenAddress"`
	DestinationAddress Address  `json:"destinationAddress"`
	Amount             *Uint256 `json:"amount"`
}

type MemoryStats struct {
	MapSize              int64 `json:"mapSize"`
	CumulativeMemorySize int64 `json:"cumulativeMemorySize"`
}

const (
	WasmSerializationError = `{"error":"wasm serialization error"}`
)
