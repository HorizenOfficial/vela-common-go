# Encrypted Seed & Privacy-Preserving Event Subtypes

## Overview

This feature adds opt-in privacy for event subtypes. Users can submit a **seed** (a secp256k1 signature) alongside their P521 public key during the `ASSOCIATEKEY` request. When a seed is registered, the executor replaces the WASM-provided `EventSubType` with a randomly chosen value from a deterministic 50-element set, preventing event linkability.

---

## 1. ASSOCIATEKEY Payload Format

The `ASSOCIATEKEY` request now supports two payload sizes:

| Payload size | Contents |
|---|---|
| **133 bytes** | P521 public key only (no privacy) |
| **226 bytes** | P521 public key (133) + encrypted seed (93) |

The encrypted seed is 93 bytes = AES-256-GCM envelope:
- **12 bytes**: nonce
- **65 bytes**: seed ciphertext (secp256k1 signature)
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

A **seed** is a 65-byte secp256k1 signature in `[R || S || V]` format (V in {0, 1}).

The signed message is:
```
keccak256("subtype-key-v1")
```

The constant `SubtypeKeyMessage = "subtype-key-v1"` is defined in the executor. Changing this string rotates all user subtype sets.

### Client-side seed generation (pseudocode)
```
msgHash = keccak256("subtype-key-v1")
seed = secp256k1_sign(msgHash, user_secp256k1_private_key)  // 65 bytes [R||S||V]
```

### Seed verification (executor-side)
The executor recovers the public key from the signature and checks that the corresponding address matches `req.Sender`:
```go
msgHash := ethCrypto.Keccak256([]byte("subtype-key-v1"))
recoveredPub, _ := ethCrypto.SigToPub(msgHash, seed)
recoveredAddr := ethCrypto.PubkeyToAddress(*recoveredPub)
// recoveredAddr must equal req.Sender
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
   c. Verify: recover secp256k1 signer from seed, must match sender address
   d. Store seed: appData.AddSeed(sender, seed)
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
│   Seed           │   65 bytes (secp256k1 signature)      │
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

### PlainEvent (pre-encryption, output from WASM)
```go
type PlainEvent struct {
    UserID       ethCommon.Address  // recipient
    EventSubType string             // original subtype from WASM
    Data         []byte             // plaintext event data
}
```

### Event (post-encryption, submitted on-chain)
```go
type Event struct {
    ApplicationID ApplicationIdType  // app ID
    UserID        ethCommon.Address   // recipient
    EventSubType  string              // privacy-preserving or original
    EncryptedData []byte              // AES-256-GCM encrypted data
}
```

### On-chain event (Solidity)
```solidity
event UserEvent(
    uint64 indexed applicationId,
    bytes32 indexed requestId,
    string indexed eventSubType,
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

1. **Generate a secp256k1 key pair** for the user (or reuse an existing one)
2. **Sign the message** `keccak256("subtype-key-v1")` with the secp256k1 private key → 65-byte seed
3. **Encrypt the seed** using ECDH with the user's P521 private key and the enclave's P521 public key, producing a 93-byte AES-256-GCM envelope (12-byte nonce + 65-byte ciphertext + 16-byte tag)
4. **Submit ASSOCIATEKEY** with 226-byte payload: `P521_public_key (133) || encrypted_seed (93)`
5. **Query events** using `eventSubType` — the client must be aware that with a seed, subtypes are no longer application-defined but are hex-encoded HMAC values. The client can reconstruct all 50 possible subtypes using `HMAC-SHA256(seed, byte(i))` for `i` in `[1, 50]` to filter events.

If the client does **not** want privacy-preserving subtypes, it submits the 133-byte payload (key only) as before — fully backward compatible.
