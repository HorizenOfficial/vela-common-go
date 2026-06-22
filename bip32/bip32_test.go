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
