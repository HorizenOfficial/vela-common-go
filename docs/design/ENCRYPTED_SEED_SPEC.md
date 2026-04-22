# Encrypted Seed & Privacy-Preserving Event Subtypes

## Overview

This feature adds opt-in privacy for event subtypes. Users can submit a **seed** (a 65-byte random value, transmitted as a 93-byte AES-256-GCM encrypted envelope) alongside their P521 public key during the `ASSOCIATEKEY` request. When a seed is registered, the executor replaces the WASM-provided `EventSubType` with a randomly chosen value from a deterministic 50-element set, preventing event linkability.

> **Scope: `PlainEvent` / `Event` only — does not apply to `AppEvent`.**
> `AppEvent` is an application-level, non-user-directed, non-encrypted event; it has no `UserID` and therefore no per-user seed to drive subtype rotation. Its `EventSubType` is always the raw `[32]byte` value the WASM app emits, passed through unchanged to the on-chain `bytes32` topic.

---

## 1. ASSOCIATEKEY Payload Format

The `ASSOCIATEKEY` request now supports two payload sizes:

| Payload size | Contents |
|---|---|
| **133 bytes** | P521 public key only (no privacy) |
| **226 bytes** | P521 public key (133) + encrypted seed (93) |

The encrypted seed is 93 bytes = AES-256-GCM envelope:
- **12 bytes**: nonce
- **65 bytes**: seed ciphertext
- **16 bytes**: GCM authentication tag

The shared key for AES-256-GCM is derived via **ECDH(user_P521_private, enclave_P521_public)**.

The smart contract (`ProcessorEndpoint.sol`) validates the payload length on-chain:
```solidity
if (requestType == REQUEST_TYPE_ASSOCIATEKEY) {
    if (payload.length != 133 && payload.length != 226) revert InvalidPayload();
}
```

---

## 2. Seed Definition

A **seed** is a 65-byte random value chosen by the client. No specific format is required.

The seed is transmitted encrypted inside the `ASSOCIATEKEY` payload; the AES-256-GCM encryption (using a key derived from the user's own P521 key pair) already proves that only the legitimate key-holder could have submitted it. No additional signature verification is performed.

### Client-side seed generation (pseudocode)
```
seed = random_bytes(65)
```

---

## 3. Executor Flow for ASSOCIATEKEY

```
1. Validate payload length: 133 or 226 bytes
2. Parse P521 public key from payload[0:133]
3. Store key: appData.AddKey(sender, key)
4. IF payload is 226 bytes:
   a. Extract encrypted seed: payload[133:226]
   b. Decrypt: ECDH(user_P521_pub, enclave_P521_priv) → AES-256-GCM decrypt → 65-byte seed
   c. Store seed: appData.AddSeed(sender, seed)
5. Add fuel cost (10 units)
```

On decryption or verification failure, the executor returns an on-chain error with `CodeParsingKeyError`.

---

## 4. AppData Serialization Format (Version 1)

The binary format for persisted application state:

```
┌──────────────────────────────────────────────────────────┐
│ Version          │ 1 byte  (uint8)                       │
│ Nonce            │ 8 bytes (uint64, big-endian)          │
│ WASM Fingerprint │ 32 bytes (SHA-256)                    │
├──────────────────────────────────────────────────────────┤
│ Key Count        │ 4 bytes (uint32, big-endian)          │
│ Key Entries      │ repeat Key Count times:               │
│   Address        │   20 bytes (Ethereum address)         │
│   P521 PubKey    │   133 bytes                           │
├──────────────────────────────────────────────────────────┤
│ Seed Count       │ 4 bytes (uint32, big-endian)          │
│ Seed Entries     │ repeat Seed Count times:              │
│   Address        │   20 bytes (Ethereum address)         │
│   Seed           │   65 bytes (random value)             │
├──────────────────────────────────────────────────────────┤
│ App State        │ variable length (remaining bytes)     │
└──────────────────────────────────────────────────────────┘
```

Constants:
- `Store_KeySize = 20` (shared by both maps)
- `KeyStore_ValSize = 133`
- `SeedStore_ValSize = 65`

Go types:
```go
type KeyStore  map[ethCommon.Address]*cryptotypes.PublicKeyP521
type SeedStore map[ethCommon.Address][]byte
```

The seed store section was added in Version 1. It is serialized **after** the key store and **before** the app state. If there are no seeds, the seed count is 0 and the section is just the 4-byte zero count.

---

## 5. Privacy-Preserving Event Subtype Generation

### Subtype generation algorithm

Each user's seed deterministically defines a set of 50 subtypes:

```
for index in [1, 50]:
    subtype[index] = "0x" + hex(HMAC-SHA256(key=seed, data=byte(index)))
```

At event emission time, the executor picks a **cryptographically random** index in [1, 50] and uses the corresponding subtype.

### Event encryption flow

This flow applies **only to `PlainEvent`** (user-directed events). `AppEvent`s bypass it entirely: their subtype is never rewritten, since there is no `UserID` to look up a seed for.

When `encryptEvents()` processes each `PlainEvent`:

```
1. Look up user's P521 public key in keyStore (fail if missing)
2. Encrypt event data: ECDH(enclave_priv, user_pub) → AES-256-GCM
3. Determine subtype:
   IF user has a seed in seedStore:
     → Generate random subtype from the 50-element HMAC set
     → On failure: log warning, fall back to WASM-provided subtype
   ELSE:
     → Use WASM-provided EventSubType as-is
4. Emit Event { ApplicationID, UserID, EventSubType, EncryptedData }
```

### Privacy properties

- **Anonymity set size**: 50 (configurable via `DefaultSubtypeN`)
- **Deterministic**: same seed always produces the same 50 subtypes
- **Unlinkable**: random selection prevents observers from correlating events to a specific user beyond 1/50 probability
- **Opt-in**: users without a seed keep the original WASM-provided subtype
- **Rotatable**: changing `SubtypeKeyMessage` rotates all users' subtype sets

---

## 6. Data Structures

The structures below are the ones affected by this feature. `AppEvent` is intentionally omitted: it is out of scope (see Overview) and its `EventSubType` is emitted unchanged.

### PlainEvent (pre-encryption, output from WASM)
```go
type PlainEvent struct {
    UserID       ethCommon.Address  // recipient
    EventSubType [32]byte           // original subtype from WASM (bytes32 on-chain)
    Data         []byte             // plaintext event data
}
```

### Event (post-encryption, submitted on-chain)
```go
type Event struct {
    ApplicationID ApplicationIdType  // app ID
    UserID        ethCommon.Address   // recipient
    EventSubType  [32]byte            // privacy-preserving or original (bytes32 on-chain)
    EncryptedData []byte              // AES-256-GCM encrypted data
}
```

### On-chain event (Solidity)
```solidity
event UserEvent(
    uint64 indexed applicationId,
    bytes32 indexed requestId,
    bytes32 indexed eventSubType,
    bytes encryptedData
);
```

### Subgraph entity (GraphQL)
```graphql
type UserEvent @entity(immutable: true) {
    id: ID!
    applicationId: BigInt!
    requestId: Bytes!
    eventSubType: Bytes!
    encryptedData: Bytes!
    blockNumber: BigInt!
    logIndex: BigInt!
    sortKey: BigInt!
    blockTimestamp: BigInt!
}
```

---

## 7. Client Integration Checklist

To be compatible with this feature, a client must:

1. **Generate 65 random bytes** to use as the seed
2. **Encrypt the seed** using ECDH with the user's P521 private key and the enclave's P521 public key, producing a 93-byte AES-256-GCM envelope (12-byte nonce + 65-byte ciphertext + 16-byte tag)
3. **Submit ASSOCIATEKEY** with 226-byte payload: `P521_public_key (133) || encrypted_seed (93)`
4. **Query events** using `eventSubType` — the client must be aware that with a seed, subtypes are no longer application-defined but are the raw 32-byte HMAC digests. The client can reconstruct all 50 possible subtypes using `HMAC-SHA256(seed, byte(i))` for `i` in `[1, 50]` to filter events.

If the client does **not** want privacy-preserving subtypes, it submits the 133-byte payload (key only) as before — fully backward compatible.
