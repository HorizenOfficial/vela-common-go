package bip32

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHMACSHA512_RFC4231_TestCase1 is the baseline HMAC-SHA512 vector from
// RFC 4231 §4.2. CKDpub's correctness depends on HMAC-SHA512 producing the
// reference output byte-for-byte; this test catches an environment where the
// primitive is broken (e.g., a TinyGo regression) before BIP-32 derivation
// would surface the failure as a confusing pubkey mismatch.
func TestHMACSHA512_RFC4231_TestCase1(t *testing.T) {
	key := make([]byte, 20)
	for i := range key {
		key[i] = 0x0b
	}
	data := []byte("Hi There")

	const want = "87aa7cdea5ef619d4ff0b4241a1d6cb02379f4e2ce4ec2787ad0b30545e17cdedaa833b7d6b8a702038b274eaea3f4e4be9d914eeb61f1702e696c203a126854"

	mac := hmac.New(sha512.New, key)
	mac.Write(data)
	got := hex.EncodeToString(mac.Sum(nil))
	assert.Equal(t, want, got)
}

// TestParseXpub_BIP32Vector1Master parses the master extended public key from
// BIP-32 Test Vector 1 (seed = 000102030405060708090a0b0c0d0e0f).
// Reference: https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
func TestParseXpub_BIP32Vector1Master(t *testing.T) {
	const xpub = "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet8"

	k, err := ParseXpub(xpub)
	require.NoError(t, err)

	assert.Equal(t, MainnetXpubVersion, k.Version)
	assert.Equal(t, uint8(0), k.Depth)
	assert.Equal(t, [4]byte{0, 0, 0, 0}, k.ParentFP)
	assert.Equal(t, uint32(0), k.ChildNumber)
	assert.Equal(t, "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508", hex.EncodeToString(k.ChainCode[:]))
	assert.Equal(t, "0339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2", hex.EncodeToString(k.PubKey[:]))
}

func TestParseXpub_RejectsBadChecksum(t *testing.T) {
	// Flip the last character; checksum will no longer validate.
	const broken = "xpub661MyMwAqRbcFtXgS5sYJABqqG9YLmC4Q1Rdap9gSE8NqtwybGhePY2gZ29ESFjqJoCu1Rupje8YtGqsefD265TMg7usUDFdp6W1EGMcet9"
	_, err := ParseXpub(broken)
	assert.ErrorIs(t, err, ErrInvalidXpub)
}

// TestCKDpub_BIP32Vector1_m0H_to_m0H1 derives m/0H/1 from m/0H using
// non-hardened CKDpub (index = 1) and verifies the child pubkey + chain code
// against the BIP-32 Test Vector 1 expectations.
func TestCKDpub_BIP32Vector1_m0H_to_m0H1(t *testing.T) {
	parent := ExtendedKey{
		PubKey:    hex33(t, "035a784662a4a20a65bf6aab9ae98a6c068a81c52e4b032c0fb5400c706cfccc56"),
		ChainCode: hex32(t, "47fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae6236141"),
		Depth:     1,
		Version:   MainnetXpubVersion,
	}

	child, err := CKDpub(parent, 1)
	require.NoError(t, err)

	assert.Equal(t, "03501e454bf00751f24b1b489aa925215d66af2234e3891c3b21a52bedb3cd711c", hex.EncodeToString(child.PubKey[:]))
	assert.Equal(t, "2a7857631386ba23dacac34180dd1983734e444fdbf774041578e9b6adb37c19", hex.EncodeToString(child.ChainCode[:]))
	assert.Equal(t, uint8(2), child.Depth)
	assert.Equal(t, uint32(1), child.ChildNumber)
}

func TestCKDpub_RejectsHardenedIndex(t *testing.T) {
	parent := ExtendedKey{PubKey: hex33(t, "0339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2")}
	_, err := CKDpub(parent, HardenedKeyOffset)
	assert.ErrorIs(t, err, ErrHardenedFromPublic)

	_, err = CKDpub(parent, HardenedKeyOffset+5)
	assert.ErrorIs(t, err, ErrHardenedFromPublic)
}

// TestCompressedPubkeyToAddress_PrivKeyOne checks the canonical Ethereum
// address for the secp256k1 generator point (private key = 1):
//
//	addr = 0x7E5F4552091A69125d5DfCb7b8C2659029395Bdf
//
// This is a well-known test value used across Ethereum tooling.
func TestCompressedPubkeyToAddress_PrivKeyOne(t *testing.T) {
	// Compressed generator point (priv = 1): 0x02 || X
	pub := hex33(t, "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798")

	addr, err := CompressedPubkeyToAddress(pub)
	require.NoError(t, err)
	assert.Equal(t, "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf", addr.Hex())
}

// TestDeriveAddress_RoundTrip is a smoke test: the convenience wrapper must
// equal CKDpub then CompressedPubkeyToAddress applied separately.
func TestDeriveAddress_RoundTrip(t *testing.T) {
	parent := ExtendedKey{
		PubKey:    hex33(t, "035a784662a4a20a65bf6aab9ae98a6c068a81c52e4b032c0fb5400c706cfccc56"),
		ChainCode: hex32(t, "47fdacbd0f1097043b78c63c20c34ef4ed9a111d980047ad16282c7ae6236141"),
	}

	addr, err := DeriveAddress(parent, 1)
	require.NoError(t, err)

	child, err := CKDpub(parent, 1)
	require.NoError(t, err)
	addrManual, err := CompressedPubkeyToAddress(child.PubKey)
	require.NoError(t, err)

	assert.Equal(t, addrManual, addr)
}

func hex33(t *testing.T, s string) [33]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	require.Len(t, b, 33)
	var out [33]byte
	copy(out[:], b)
	return out
}

func hex32(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	require.Len(t, b, 32)
	var out [32]byte
	copy(out[:], b)
	return out
}

// TestMasterKey_BIP32Vector1 derives the master extended public key from
// BIP-32 Test Vector 1's 16-byte seed (000102030405060708090a0b0c0d0e0f)
// and verifies the resulting pubkey + chain code match the spec's master.
// Reference: https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki
func TestMasterKey_BIP32Vector1(t *testing.T) {
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	require.NoError(t, err)
	require.Len(t, seed, 16)

	master, err := MasterKey(seed)
	require.NoError(t, err)

	assert.Equal(t, uint8(0), master.Depth, "master depth must be 0")
	assert.Equal(t, uint32(0), master.ChildNumber, "master child number must be 0")
	assert.Equal(t, [4]byte{0, 0, 0, 0}, master.ParentFP, "master parent fingerprint must be zero")
	assert.Equal(t, MainnetXpubVersion, master.Version)
	assert.Equal(t, "0339a36013301597daef41fbe593a02cc513d0b55527ec2df1050e2e8ff49c85c2", hex.EncodeToString(master.PubKey[:]))
	assert.Equal(t, "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508", hex.EncodeToString(master.ChainCode[:]))
}

// TestMasterKey_SeedLengthBounds covers the BIP-32 spec's 16–64 byte seed
// length rule. Lengths outside that range are rejected.
func TestMasterKey_SeedLengthBounds(t *testing.T) {
	_, err := MasterKey(make([]byte, 15))
	assert.Error(t, err, "15-byte seed must be rejected")
	_, err = MasterKey(make([]byte, 65))
	assert.Error(t, err, "65-byte seed must be rejected")

	// Boundary values pass length check (independent of derivation outcome).
	_, err = MasterKey(make([]byte, 16))
	assert.NoError(t, err, "16-byte seed must be accepted")
	_, err = MasterKey(make([]byte, 64))
	assert.NoError(t, err, "64-byte seed must be accepted")
}

// TestParseXpub_RejectsOffCurvePubkey covers the fail-fast on-curve branch in
// ParseXpub. offCurveXpub is a fixed, externally-generated Base58Check xpub:
// a well-formed envelope (mainnet version, valid 4-byte checksum) whose 33-byte
// pubkey field has a valid 0x02 prefix but an X coordinate that is not on the
// secp256k1 curve. It therefore passes every earlier check (length, prefix,
// checksum) and exercises only the curve validation, which must reject it.
//
// The fixture is a constant rather than synthesized at runtime so the test
// input is independent of the code under test (base58Decode / doubleSHA256) —
// the same property that makes the positive BIP-32 vectors trustworthy.
func TestParseXpub_RejectsOffCurvePubkey(t *testing.T) {
	const offCurveXpub = "xpub661MyMwAqRbcEYS8w7XLSVeEsBXy79zSzH1J8vCdxAZningWLdN3zgtU6Q5JXayek4PRsn35jii4veMimro1xefsM58PgBMrvdYrdxDSid5"

	_, err := ParseXpub(offCurveXpub)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidXpub)
	assert.Contains(t, err.Error(), "not on curve")
}
