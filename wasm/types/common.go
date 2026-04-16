package types

const (
	MaxBigIntBytes  = 32
	MaxAddressBytes = AddressLength
)

// PlainEvent is a local replacement for common.PlainEvent
type PlainEvent struct {
	UserID       Address `json:"userId"`
	EventSubType string  `json:"eventSubType"`
	Data         []byte  `json:"data"`
}

// AppEvent is a local replacement for common.AppEvent (application-level, non-encrypted event)
type AppEvent struct {
	EventSubType string `json:"eventSubType"`
	Data         []byte `json:"data"`
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
