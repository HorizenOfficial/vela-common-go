# Horizen Vela Common Go Library

Shared Go types and utilities for the Horizen Vela platform. Contains only code that is genuinely shared across multiple consumers (the `vela` framework, specifi application wasms and wallet CLI, etc.) — app-specific types stay in each app.

## Packages

| Package | Purpose |
|---------|---------|
| [`common/`](./common) | Framework-level types shared between `vela` and `vela-nova` — `ApplicationIdType`, `RequestIdType`, `RequestResultStatus`, `DeployDescriptor`. |
| [`subgraph/`](./subgraph) | GraphQL client for The Graph subgraph, with projection types (`RequestCompleted`, `UserEvent`, `AppEvent`, `OnChainRefund`, `OnChainWithdrawal`, `ClaimExecuted`) and a `MockClient` test double. |
| [`subtypes/`](./subtypes) | Deterministic HMAC-SHA256 derivation of privacy-preserving `[32]byte` event subtypes from a seed. The raw bytes match the on-chain `bytes32` topic. |
| [`wasm/`](./wasm) | WASM guest-side primitives (`Uint256`, `Address`, result types), memory management, and logging. See [`wasm/README.md`](./wasm/README.md) for details. |

## Build & Test

```bash
# All tests
go test ./... -v

# Single package
go test github.com/HorizenOfficial/vela-common-go/wasm/types -v

# Manage dependencies
go mod tidy
```

## License

See [LICENSE](./LICENSE) and [NOTICES](./NOTICES).
