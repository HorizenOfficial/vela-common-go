# Changelog

## [0.3.0]

### Removed

- **BREAKING:** `wasm/types.LoadModuleResult` — the exported result type (`State`, `Fuel`, `Error`)
  was unused by the framework and by every downstream consumer, and has been dropped along with its
  JSON round-trip test. Consumers that still reference `types.LoadModuleResult` must define it
  locally. Because this removes an exported symbol, the next tag must be **v0.3.0**, not v0.2.1.

### Documentation

- `CLAUDE.md` and `wasm/README.md` no longer list `LoadModuleResult` among the WASM result types.

## [0.2.0] 

### Added

- `NOTICES` — dependency attribution for the full `go.mod` require set, grouped by license:
  a copyleft section for `go-ethereum` (LGPL-3.0-or-later) and permissive entries for the indirect
  deps `holiman/uint256` and `golang.org/x/sys`.
- `CLAUDE.md` — "Dependency Attribution (NOTICES)" contributor rule: every `go.mod` change must be
  reconciled against `NOTICES` (version, license, source URL, copyright), after `go mod tidy`.

### Changed

- Merged the `v0.1.0` and `v0.1.1` lines, which had been tagged on divergent branches, into a single
  release.

## [0.1.1]

### Added

- `common.ETH_TOKEN` — sentinel for the chain's native token (ETH / HZN), mirroring the
  `address constant ETH_TOKEN = address(0)` defined in `Structs.sol`, so callers stop open-coding
  `ethCommon.Address{}`. Declared as `var` (Go forbids const arrays); treat as immutable.

## [0.1.0] 

### Added

- ERC-20 integration support across the shared types and the subgraph client.

### Documentation

- README and `CLAUDE.md` updates covering the ERC-20 flow.

## [0.0.20] - [0.0.29] 

Pre-release series that established the library's current shape:

### Added

- `common/` — framework wire types: `ApplicationIdType`, `RequestIdType`, `RequestResultStatus`,
  `ConstructorParams`, `DeployDescriptor` / `DeployModeArtifactRef`.
- `subgraph/` — GraphQL client for The Graph (`Client` interface, projection types, `MockClient`
  with a builder API, `ComputeSortKey`).
- `subtypes/` — HMAC-SHA256 derived privacy-preserving event subtypes (`GenerateSubtypes`,
  `GenerateSubtypesN`, `DefaultSubtypeN`).
- `docs/design/ENCRYPTED_SEED_SPEC.md`, `LICENSE`, and the initial `NOTICES`.

### Changed

- `wasm/types` and `wasm/utils` refinements (result types, pointer helpers, memory manager) carried
  over from the `vela-nova` migration.

