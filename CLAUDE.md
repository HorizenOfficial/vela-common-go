# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Common Go utility library for the Horizen Confidential Compute Environment (HCCE), focused on WebAssembly (WASM) support. Contains shared types and utilities used by WASM modules running within the Horizen blockchain system.

**Origin:** Migrated from `horizen-pes/nova` repository as a shared library.
**Downstream consumers:**
- `horizen-pes/app` (simple app) - imports this as a dependency
- `horizen-pes-nova/runtime/wasm-go` (payment app) - imports this as a dependency
- `horizen-pes-nova/wallet` (wallet CLI) - imports this as a dependency, bridges go-ethereum types to wasm types

## Build & Test Commands

```bash
# Run all tests (both packages)
go test ./... -v

# Run tests for a specific package
go test github.com/horizen-cce-common-go/wasm/types -v
go test github.com/horizen-cce-common-go/wasm/utils -v

# Run a single test
go test github.com/horizen-cce-common-go/wasm/types -run TestAdd64Overflow

# Manage dependencies
go mod tidy
```

Tests use `testify` (assert & require packages) with table-driven and property-based patterns. Random tests use fixed seeds (e.g., `rand.NewSource(1234)`) for CI stability.

## Architecture

### WASM Sandbox Design

The module is completely self-contained for WASM sandbox isolation. Host ↔ Guest communication happens through JSON serialization only - no direct type imports between environments.

```
Host (Go runtime)          Guest (WASM module)
     |                            |
     |  --- JSON bytes --->       |
     |  <-- JSON bytes ---        |
```

The host uses `math/big.Int` and `go-ethereum/common.Address`; the guest uses `Uint256` and `Address`. Both serialize to identical JSON hex format (`"0x..."`). This compatibility is validated by tests in the downstream consumer (`horizen-pes/app/simple/app/app_test.go`).

### Package Structure

**`wasm/types/`** - Core data types for WASM guest environment:
- `Uint256` - 256-bit unsigned integer as `[4]uint64` (little-endian words), with overflow-detecting arithmetic
- `Address` - Ethereum-style 20-byte address
- `PlainEvent`, `Withdrawal` - Domain types replacing host equivalents
- Result types (`LoadModuleResult`, `DepositResult`, `ProcessResult`, `DeanonymizationResult`) - WASM operation returns
- `helpers.go` - WASM pointer ↔ type conversion (`PtrToUint256`, `PtrToAddress`, `SerializeAndWriteResult`)

**`wasm/utils/`** - Runtime utilities:
- `memory.go` - WASM linear memory allocation via global map (prevents GC collection), plus `BytesToPtr`/`PtrToString` for data translation
- `logger.go` - WASI pipe logging with prefixes (`TRC`, `DBG`, `INF`, `WRN`, `ERR`)

### Key Constraints

- **TinyGo compilation** - Limited sync support, linear memory model
- **Single-threaded guest** - Concurrency handled on host side
- **32-bit WASM pointers** - `MaxWasmDataSize = MaxInt32 - 4` bytes; `utils/memory.go` functions that convert `int32` pointers cannot be unit-tested in native 64-bit Go (segfaults due to pointer truncation). These are exercised through integration tests in downstream consumers that compile to WASM and validate the full `allocate` → `BytesToPtr` → `extractResultBytes` → `deallocate` round-trip.
- **Memory base** - 2 pages (128KB), Go heap grows beyond via `memory.grow`

### Uint256 Implementation Notes

- Little-endian word order: `[4]uint64` where index 0 is least significant
- `SetBytes`/`Bytes` use big-endian byte order (compatible with `big.Int.Bytes()`)
- JSON serializes as hex with lowercase `0x` prefix (rejects uppercase `0X`)
- Uses `math/bits` for efficient carry/borrow propagation
- Max decimal digits: 78

### Design Conventions

- **Strict `0x` prefix** - All types (`Uint256`, `Address`) reject uppercase `0X` prefix for consistency. This applies to `UnmarshalJSON`, `HexToAddress`, and `SetHex`.
- **Zero-on-set** - Both `Uint256.SetBytes` and `Address.SetBytes` zero the receiver before writing, preventing stale data.
- **Overflow-aware API** - Arithmetic methods come in pairs: `Add`/`AddOverflow`, `Sub`/`SubOverflow`, `Mul64`/`Mul64Overflow`, `Add64`/`Add64Overflow`. The downstream consumer relies on overflow detection for financial safety.
- **Naming** - Use Go camelCase for all variables and return values (no snake_case). Use `LogWarn`/`LogError` from `utils` instead of `println` for all diagnostic output.
- **WASM exports** - Guest modules should export `get_memory_stats` (returning `MemoryStats` via `SerializeAndWriteResult`) to enable memory leak detection in integration tests.

### Go 1.22+ Loop Variable Scoping

With Go 1.22+, each iteration of a `for range` loop creates a new scope. Using `:=` inside a loop body shadows outer variables — the outer variable is never updated. Use `newVar := ...; outerVar = newVar` pattern when accumulating state across iterations.

### Integration Test Patterns

Both downstream consumers include memory-aware integration tests that validate the WASM memory round-trip:
- **MemoryCleanBetweenOps** — calls `GetAllocatedMemoryStats2` after each operation to verify no leaks
- **ErrorPathMemory** — verifies error results (still serialized via `BytesToPtr`) don't leak memory
- **LargeResultRoundTrip** — exercises `BytesToPtr` with large payloads (100+ accounts)
