# horizen-cce-common-go/wasm

Common Go library for WebAssembly (WASM) guest modules running within the Horizen Confidential Compute Environment (HCCE).

Provides shared types and utilities that WASM guest applications use to communicate with the host runtime. Compiled with [TinyGo](https://tinygo.org/) to target `wasm32-wasi`.

## Architecture

WASM modules run in a sandboxed environment. The host and guest communicate exclusively through JSON-serialized messages over WASM linear memory — no direct type imports cross the boundary.

```
   Host (Go runtime)                           Guest (WASM module)

  math/big.Int ───────────────┐              ┌── Uint256 [4]uint64
  go-ethereum common.Address ─┤  JSON bytes  ├── Address [20]byte
  common.PlainEvent ──────────┤ ◄──────────► ├── PlainEvent
  wasmCommon.*Result ─────────┘              └── *Result types
```

The guest-side types are intentionally separate from host types. TinyGo does not support the full Go standard library (e.g., `go-ethereum` is not available), so lightweight equivalents are provided here.

## Packages

### `types`

Core data types for the WASM guest environment.

| Type | Description |
|------|-------------|
| `Uint256` | 256-bit unsigned integer as `[4]uint64` (little-endian word order). Overflow-detecting arithmetic via `Add`/`AddOverflow`, `Sub`/`SubOverflow`, `Mul64`/`Mul64Overflow`, `Add64`/`Add64Overflow`. JSON serializes as lowercase `0x`-prefixed hex. |
| `Address` | Ethereum-style 20-byte address. JSON serializes as `0x`-prefixed hex. |
| `PlainEvent` | Guest-side event with user ID, sub-type, and data payload. |
| `Withdrawal` | Withdrawal request with destination address and amount. |
| `LoadModuleResult`, `DepositResult`, `ProcessResult`, `DeanonymizationResult` | Result types returned by WASM-exported functions. |
| `MemoryStats` | Allocation statistics from the memory manager. |

### `utils`

Runtime utilities for the WASM guest.

| Component | Description |
|-----------|-------------|
| `Allocate` / `Deallocate` | WASM linear memory allocator. Pins allocations in a global map to prevent GC collection. Exported as `allocate` / `deallocate` for host calls. |
| `BytesToPtr` | Converts a Go byte slice into a WASM-compatible pointer with a 4-byte little-endian length prefix. |
| `PtrToString` | Converts a WASM pointer and length into a Go string. |
| `GetAllocatedMemoryStats` | Returns current allocation count and cumulative size. Exported as `get_allocated_memory_stats`. |
| `LogTrace` .. `LogError` | Leveled logging via WASI stdout pipe. The host parses prefixes (`TRC`, `DBG`, `INF`, `WRN`, `ERR`) to route to appropriate log levels. |

## Usage

Import the module in your WASM guest application:

```go
import (
    "github.com/horizen-cce-common-go/wasm/types"
    "github.com/horizen-cce-common-go/wasm/utils"
)
```

### Uint256

```go
// From uint64
v := types.NewUint256(1000)

// From hex
v = new(types.Uint256)
v.SetHex("0xff")

// Arithmetic with overflow detection
sum := new(types.Uint256)
if sum.AddOverflow(a, b) {
    // handle overflow
}

// JSON round-trip: serializes as "0xff"
data, _ := json.Marshal(v)
```

### Address

```go
addr, err := types.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
addr.Hex()    // "0x1234567890abcdef1234567890abcdef12345678"
addr.IsZero() // false
```

### WASM Memory

```go
// Return a result to the host
func processRequest(payloadPtr *byte, payloadLen int32) *byte {
    payload := utils.PtrToString(payloadPtr, payloadLen)
    result := doWork(payload)
    return types.SerializeAndWriteResult(result)
}
```

## Conventions

- **Strict `0x` prefix**: all hex parsing rejects uppercase `0X` for consistency across types.
- **Zero-on-set**: `SetBytes` methods zero the receiver before writing, preventing stale data.
- **Overflow-aware API**: arithmetic methods come in pairs (`Add`/`AddOverflow`). The non-overflow variant silently wraps modulo 2^256; the overflow variant returns a boolean. Callers handling balances or funds must use the overflow-detecting variants.

## Testing

```bash
# Run all tests
go test github.com/horizen-cce-common-go/wasm/types -v
go test github.com/horizen-cce-common-go/wasm/utils -v

# Run a single test
go test github.com/horizen-cce-common-go/wasm/types -run TestUint256Add
```

Tests use `testify` with table-driven and property-based patterns. Random tests use fixed seeds for CI reproducibility.

Note: functions that involve 32-bit WASM pointer arithmetic (`BytesToPtr`, `SerializeAndWriteResult`) cannot be unit-tested in native 64-bit Go. They are exercised through integration tests in the downstream consumers (e.g., `horizen-pes/app/simple/integration_test.go`) that compile to WASM and validate the full `allocate` → `BytesToPtr` → `extractResultBytes` → `deallocate` round-trip.

## Constraints

- **TinyGo**: limited `sync` support, no `reflect` in some cases, linear memory model.
- **Single-threaded guest**: concurrency is managed on the host side.
- **32-bit WASM pointers**: maximum data size is `math.MaxInt32 - 4` bytes.
- **Memory layout**: TinyGo configures 2 initial pages (128KB); the Go heap grows beyond via `memory.grow`.

## License

See repository root for license information.
