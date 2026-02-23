# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Common Go utility library for the Horizen Confidential Compute Environment (HCCE). Contains shared types and utilities used across the Horizen blockchain system. Wallet/app-side packages live under `wallet/` (e.g., `wallet/common`, `wallet/subgraph`, `wallet/blockchain`). The `wasm/` domain stays separate for guest-side shared code.

**Origin:** Migrated from `horizen-pes/nova` repository as a shared library.
**Downstream consumers:**
- `horizen-pes/app/simple` (simple app) - imports this as a dependency
- `horizen-pes-nova/runtime/wasm-go` (payment app) - imports this as a dependency
- `horizen-pes-nova/wallet` (wallet CLI) - imports this as a dependency, bridges go-ethereum types to wasm types, owns `FetchAndDecryptUserEvents` (in `cmd/user_events.go`)

**Scope principle:** This library contains only code that is genuinely shared across multiple consumers. For WASM packages, that means what *any* WASM guest needs — primitive types (`Uint256`, `Address`), framework result types, memory management, and logging. The same principle applies to any new top-level domain: only extract code here when multiple projects need it. App-specific types (event schemas, account models, instruction types, report structures) stay in each app even if two apps happen to define identical types, because a different consumer may not need them at all.

## Build & Test Commands

```bash
# Run all tests (all packages)
go test ./... -v

# Run tests for a specific top-level domain
go test ./wasm/... -v

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

### Repository Layout

Top-level domains in this repository:

```
horizen-cce-common-go/
├── wallet/
│   ├── common/    # Shared wallet/app-side types (ApplicationIdType, RequestIdType, RequestType, etc.)
│   ├── subgraph/  # GraphQL client for The Graph subgraph
│   └── blockchain/# App-side blockchain client (submit + tee key reads)
├── wasm/          # WASM guest types and utilities
│   ├── types/     # Uint256, Address, result types, helpers
│   └── utils/     # Memory allocation, logging
```

Keep wallet/app-side shared functionality under `wallet/`; keep WASM guest shared functionality under `wasm/`.

### Common Package Structure

**`wallet/common/`** - Shared wallet/app-side framework types used by both `horizen-pes` and `horizen-pes-nova`:
- `ApplicationIdType` - Application identifier (`uint64`). `NewApplicationId` takes `uint64`. `horizen-pes/pkg/common` re-exports this as a type alias for backward compatibility.
- `RequestIdType` - 32-byte request identifier with hex JSON serialization
- `RequestResultStatus` - Request outcome enum (`RequestResultOK`, `RequestResultFailed`, `RequestResultUnknown`)

### Subgraph Package Structure

**`wallet/subgraph/`** - GraphQL client for querying The Graph subgraph:
- `Client` interface - `HealthCheck`, `GetRequestCompletedByID`, `GetUserEvents`
- `RequestCompleted`, `UserEvent` - Projection types returned by queries
- `NewClient` - Client constructor
- `MockClient` - Test double with builder pattern (`WithRequestCompleted`, `WithUserEvents`)
- `ComputeSortKey` - Sort key computation for event pagination

### WASM Sandbox Design

The module is completely self-contained for WASM sandbox isolation. Host ↔ Guest communication happens through JSON serialization only - no direct type imports between environments.

```
Host (Go runtime)          Guest (WASM module)
     |                            |
     |  --- JSON bytes --->       |
     |  <-- JSON bytes ---        |
```

The host uses `math/big.Int` and `go-ethereum/common.Address`; the guest uses `Uint256` and `Address`. Both serialize to identical JSON hex format (`"0x..."`). This compatibility is validated by tests in the downstream consumer (`horizen-pes/app/simple/app/app_test.go`).

### WASM Package Structure

**`wasm/types/`** - Core data types for WASM guest environment:
- `Uint256` - 256-bit unsigned integer as `[4]uint64` (little-endian words), with overflow-detecting arithmetic
- `Address` - Ethereum-style 20-byte address
- `PlainEvent`, `Withdrawal` - Domain types replacing host equivalents
- Result types (`LoadModuleResult`, `DepositResult`, `ProcessResult`, `DeanonymizationResult`) - WASM operation returns
- `helpers.go` - WASM pointer ↔ type conversion (`PtrToUint256`, `PtrToAddress`, `SerializeAndWriteResult`)

**`wasm/utils/`** - Runtime utilities:
- `memory.go` - WASM linear memory allocation via global map (prevents GC collection), plus `BytesToPtr`/`PtrToString` for data translation
- `logger.go` - WASI pipe logging with prefixes (`TRC`, `DBG`, `INF`, `WRN`, `ERR`)

### WASM Key Constraints

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
- **WASM ABI boundary** - WASM has no unsigned integer types; `i32`/`i64` are just 32/64 bits with signedness only in operations. The host passes `ApplicationIdType` (`uint64`) as `int64` via `ToWasmType()` — a bit-preserving reinterpret cast. Guest exports receive `int64` and cast back with `uint64(appId)`. This round-trip preserves the full `uint64` range including values above `MaxInt64`.

### Go 1.22+ Loop Variable Scoping

With Go 1.22+, each iteration of a `for range` loop creates a new scope. Using `:=` inside a loop body shadows outer variables — the outer variable is never updated. Use `newVar := ...; outerVar = newVar` pattern when accumulating state across iterations.

### Host vs Guest Type Boundary

The framework (`horizen-pes`) and the WASM apps live in separate type worlds connected only by JSON:

- **Host-side types** (framework): `ethCommon.Address`, `*common.Big`, `common.Event`, `common.Withdrawal`
- **Guest-side types** (this library): `types.Address`, `*types.Uint256`, `types.PlainEvent`, `types.Withdrawal`
- **App-specific event types** (each app): `DepositEvent`, `SenderEvent`, `RecipientEvent`, `WithdrawalEvent` — defined locally in each WASM app's `app/types.go` using guest-side types. Host-side test code defines its own mirror types using `*common.Big` / `ethCommon.Address` for JSON deserialization of the same events.

The framework never imports app-specific types. Framework test helpers (`pkg/testutil`) validate events as opaque JSON (`json.Valid()`), not by deserializing into app-specific structs. App-specific event validation belongs in each app's own system tests.

### System Test Patterns

All three full-flow system tests (simple app, payment app, mock runtime) follow the same sequence:
1. **Deploy** — submit WASM bytecode, wait for state in DB + blockchain
2. **Register keys** — user key + auditor key via `AssociateKey` requests
3. **Deposit** — submit deposit, validate encrypted event fields + update payload signature
4. **Withdrawal** — submit withdrawal, validate encrypted event fields + on-chain withdrawal + signature
5. **Deanonymization report** — submit as auditor, decrypt report, validate framework envelope (`applicationId`, `requestId`), decode base64 `reportDataBytes`, verify expected account balances

The deanonymization report comes **last** so it serves as a final state audit verifying the cumulative effect of all prior operations (e.g., 2 ETH deposited - 0.5 ETH withdrawn = 1.5 ETH balance in report).

### Integration Test Patterns

Both downstream consumers include memory-aware integration tests that validate the WASM memory round-trip:
- **MemoryCleanBetweenOps** — calls `GetAllocatedMemoryStats2` after each operation to verify no leaks
- **ErrorPathMemory** — verifies error results (still serialized via `BytesToPtr`) don't leak memory
- **LargeResultRoundTrip** — exercises `BytesToPtr` with large payloads (100+ accounts)
