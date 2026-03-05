package types

// --- Local replacements for Host types ---
// This is a deliberate design choice required by the WebAssembly architecture.
// The application communicates by serializing the host-side struct to JSON, passing it to the
// Wasm module, which then deserializes it into its own identical local struct.
// This maintains a clean separation between the two environments.
// The Wasm module is a separate, sandboxed program and should not import types directly from
// the host application's packages, even if they are defined exactly the same way.
// Moreover we do use analogous but different types, for instance ethereum addresses in the Host
// and [20]byte array type in the guest (this is because tinygo does not support the full standard
// go runtime needed by go-ethereum).
// Similarly we use math/big.Int in the host and Uint256 type in the guest.
// ---

// LoadModuleResult is a local replacement for wasmCommon.LoadModuleResult
type LoadModuleResult struct {
	State []byte   `json:"state"`
	Fuel  *Uint256 `json:"fuel"`
	Error string   `json:"error,omitempty"`
}

// DepositResult is a local replacement for wasmCommon.DepositResult
type DepositResult struct {
	State  []byte       `json:"state"`
	Events []PlainEvent `json:"events"`
	Fuel   *Uint256     `json:"fuel"`
	Error  string       `json:"error,omitempty"`
}

// ProcessResult is a local replacement for wasmCommon.ProcessResult
type ProcessResult struct {
	State       []byte       `json:"state"`
	Events      []PlainEvent `json:"events"`
	Withdrawals []Withdrawal `json:"withdrawals"`
	Report      []byte       `json:"report,omitempty"` // Optional deanonymization report
	Fuel        *Uint256     `json:"fuel"`
	Error       string       `json:"error,omitempty"`
}
