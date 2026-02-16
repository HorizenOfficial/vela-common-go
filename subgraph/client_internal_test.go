package subgraph

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/horizen-cce-common-go/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseRequestID ---

// TestParseRequestID_Valid parses a well-formed 0x-prefixed 64-char hex string
// into a 32-byte RequestIdType.
func TestParseRequestID_Valid(t *testing.T) {
	hex := "0x0000000000000000000000000000000000000000000000000000000000000001"
	id, err := parseRequestID(hex)
	require.NoError(t, err)

	var expected common.RequestIdType
	expected[31] = 1
	assert.Equal(t, expected, id)
}

// TestParseRequestID_Without0xPrefix verifies that the 0x prefix is optional.
func TestParseRequestID_Without0xPrefix(t *testing.T) {
	hex := "0000000000000000000000000000000000000000000000000000000000000002"
	id, err := parseRequestID(hex)
	require.NoError(t, err)

	var expected common.RequestIdType
	expected[31] = 2
	assert.Equal(t, expected, id)
}

// TestParseRequestID_TooLong rejects hex strings that decode to more than 32 bytes.
func TestParseRequestID_TooLong(t *testing.T) {
	hex := "0x" + "ff" + "0000000000000000000000000000000000000000000000000000000000000001"
	_, err := parseRequestID(hex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be more than 32 bytes")
}

// TestParseRequestID_InvalidHex rejects non-hex characters.
func TestParseRequestID_InvalidHex(t *testing.T) {
	_, err := parseRequestID("0xZZZZ")
	require.Error(t, err)
}

// --- decodeHex ---

// TestDecodeHex_WithPrefix decodes a 0x-prefixed hex string.
func TestDecodeHex_WithPrefix(t *testing.T) {
	result, err := decodeHex("0xcafe")
	require.NoError(t, err)
	assert.Equal(t, []byte{0xca, 0xfe}, result)
}

// TestDecodeHex_WithoutPrefix decodes a bare hex string.
func TestDecodeHex_WithoutPrefix(t *testing.T) {
	result, err := decodeHex("abcd")
	require.NoError(t, err)
	assert.Equal(t, []byte{0xab, 0xcd}, result)
}

// TestDecodeHex_Empty returns nil for an empty string without error.
func TestDecodeHex_Empty(t *testing.T) {
	result, err := decodeHex("")
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestDecodeHex_Only0x returns nil when the string is just the 0x prefix.
func TestDecodeHex_Only0x(t *testing.T) {
	result, err := decodeHex("0x")
	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestDecodeHex_Invalid rejects non-hex characters.
func TestDecodeHex_Invalid(t *testing.T) {
	_, err := decodeHex("0xGHIJ")
	require.Error(t, err)
}

// --- uint8ToRequestResultStatus ---

// TestUint8ToRequestResultStatus_OK maps 0 to RequestResultOK.
func TestUint8ToRequestResultStatus_OK(t *testing.T) {
	s, err := uint8ToRequestResultStatus(0)
	require.NoError(t, err)
	assert.Equal(t, common.RequestResultOK, s)
}

// TestUint8ToRequestResultStatus_Failed maps 1 to RequestResultFailed.
func TestUint8ToRequestResultStatus_Failed(t *testing.T) {
	s, err := uint8ToRequestResultStatus(1)
	require.NoError(t, err)
	assert.Equal(t, common.RequestResultFailed, s)
}

// TestUint8ToRequestResultStatus_Unknown returns an error for any value >= 2.
func TestUint8ToRequestResultStatus_Unknown(t *testing.T) {
	for _, v := range []uint8{2, 5, 255} {
		s, err := uint8ToRequestResultStatus(v)
		require.Error(t, err)
		assert.Equal(t, common.RequestResultUnknown, s)
	}
}

// --- stringToBigInt ---

// TestStringToBigInt_Valid parses a decimal string into a big.Int.
func TestStringToBigInt_Valid(t *testing.T) {
	val, ok := stringToBigInt("12345")
	require.True(t, ok)
	assert.Equal(t, big.NewInt(12345), val)
}

// TestStringToBigInt_Zero parses "0" correctly.
func TestStringToBigInt_Zero(t *testing.T) {
	val, ok := stringToBigInt("0")
	require.True(t, ok)
	assert.Equal(t, big.NewInt(0), val)
}

// TestStringToBigInt_Invalid returns false for non-numeric strings.
func TestStringToBigInt_Invalid(t *testing.T) {
	_, ok := stringToBigInt("not-a-number")
	assert.False(t, ok)
}

// TestStringToBigInt_HexRejected rejects hex strings since it parses base-10 only.
func TestStringToBigInt_HexRejected(t *testing.T) {
	_, ok := stringToBigInt("0xff")
	assert.False(t, ok)
}

// --- requestIdStringTo32Byte ---

// TestRequestIdStringTo32Byte_Valid converts a 64-char hex string into [32]byte.
func TestRequestIdStringTo32Byte_Valid(t *testing.T) {
	hex := "0000000000000000000000000000000000000000000000000000000000000001"
	arr, err := requestIdStringTo32Byte(hex)
	require.NoError(t, err)

	var expected [32]byte
	expected[31] = 1
	assert.Equal(t, expected, arr)
}

// TestRequestIdStringTo32Byte_Short accepts shorter hex strings (left-padded with zeros).
func TestRequestIdStringTo32Byte_Short(t *testing.T) {
	arr, err := requestIdStringTo32Byte("ff")
	require.NoError(t, err)

	var expected [32]byte
	expected[0] = 0xff
	assert.Equal(t, expected, arr)
}

// TestRequestIdStringTo32Byte_TooLong rejects hex strings decoding to > 32 bytes.
func TestRequestIdStringTo32Byte_TooLong(t *testing.T) {
	hex := "00" + "0000000000000000000000000000000000000000000000000000000000000001"
	_, err := requestIdStringTo32Byte(hex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be more than 32 bytes")
}

// TestRequestIdStringTo32Byte_InvalidHex rejects non-hex characters.
func TestRequestIdStringTo32Byte_InvalidHex(t *testing.T) {
	_, err := requestIdStringTo32Byte("ZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid hex string")
}

// --- HTTP client via httptest ---

// fakeSubgraph starts a test HTTP server that responds to GraphQL queries
// with the given JSON body and status code.
func fakeSubgraph(t *testing.T, statusCode int, body interface{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestClient_HealthCheck_OK verifies that a healthy subgraph response is accepted.
func TestClient_HealthCheck_OK(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"_meta": map[string]interface{}{
				"hasIndexingErrors": false,
			},
		},
	})

	c := NewClient(srv.URL)
	err := c.HealthCheck(context.Background())
	require.NoError(t, err)
}

// TestClient_HealthCheck_IndexingErrors verifies that indexing errors are reported.
func TestClient_HealthCheck_IndexingErrors(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"_meta": map[string]interface{}{
				"hasIndexingErrors": true,
			},
		},
	})

	c := NewClient(srv.URL)
	err := c.HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "indexing errors")
}

// TestClient_HealthCheck_EmptyMeta verifies that a missing _meta field is an error.
func TestClient_HealthCheck_EmptyMeta(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{},
	})

	c := NewClient(srv.URL)
	err := c.HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty meta")
}

// TestClient_HealthCheck_GraphQLError verifies that a subgraph-level error is surfaced.
func TestClient_HealthCheck_GraphQLError(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data":   map[string]interface{}{},
		"errors": []map[string]interface{}{{"message": "something broke"}},
	})

	c := NewClient(srv.URL)
	err := c.HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "something broke")
}

// TestClient_HealthCheck_HTTPError verifies that non-2xx status codes are reported.
func TestClient_HealthCheck_HTTPError(t *testing.T) {
	srv := fakeSubgraph(t, 500, "internal server error")

	c := NewClient(srv.URL)
	err := c.HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

// TestClient_GetRequestCompletedByID_Found verifies successful parsing of a
// RequestCompleted entity from the subgraph response.
func TestClient_GetRequestCompletedByID_Found(t *testing.T) {
	var reqID common.RequestIdType
	reqID[31] = 1

	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"requestCompleteds": []map[string]interface{}{
				{
					"requestId":       "0x0000000000000000000000000000000000000000000000000000000000000001",
					"status":          "0",
					"errorCode":       "0",
					"errorMessage":    "",
					"applicationFees": "1000",
					"blockNumber":     "42",
				},
			},
		},
	})

	c := NewClient(srv.URL)
	rc, err := c.GetRequestCompletedByID(context.Background(), reqID)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, reqID, rc.RequestID)
	assert.Equal(t, common.RequestResultOK, rc.Status)
	assert.Equal(t, uint8(0), rc.ErrorCode)
	assert.Equal(t, big.NewInt(1000), rc.ApplicationFees)
	assert.Equal(t, uint64(42), rc.BlockNumber)
}

// TestClient_GetRequestCompletedByID_NotFound verifies that an empty result
// set returns nil without error.
func TestClient_GetRequestCompletedByID_NotFound(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"requestCompleteds": []map[string]interface{}{},
		},
	})

	c := NewClient(srv.URL)
	rc, err := c.GetRequestCompletedByID(context.Background(), common.RequestIdType{})
	require.NoError(t, err)
	assert.Nil(t, rc)
}

// TestClient_GetRequestCompletedByID_FailedStatus verifies that status=1
// is mapped to RequestResultFailed with error details.
func TestClient_GetRequestCompletedByID_FailedStatus(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"requestCompleteds": []map[string]interface{}{
				{
					"requestId":       "0x0000000000000000000000000000000000000000000000000000000000000001",
					"status":          "1",
					"errorCode":       "5",
					"errorMessage":    "something went wrong",
					"applicationFees": "0",
					"blockNumber":     "10",
				},
			},
		},
	})

	c := NewClient(srv.URL)
	var reqID common.RequestIdType
	reqID[31] = 1
	rc, err := c.GetRequestCompletedByID(context.Background(), reqID)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, common.RequestResultFailed, rc.Status)
	assert.Equal(t, uint8(5), rc.ErrorCode)
	assert.Equal(t, "something went wrong", rc.ErrorMessage)
}

// TestClient_GetUserEvents_ParsesResponse verifies that the client correctly
// parses a subgraph userEvents response into UserEvent structs.
func TestClient_GetUserEvents_ParsesResponse(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"userEvents": []map[string]interface{}{
				{
					"applicationId": "1",
					"requestId":     "0x0000000000000000000000000000000000000000000000000000000000000001",
					"eventSubType":  "deposit",
					"encryptedData": "0xcafe",
					"blockNumber":   "100",
					"logIndex":      "3",
					"sortKey":       "100000000000003",
				},
			},
		},
	})

	c := NewClient(srv.URL)
	appID := common.NewApplicationId(1)
	events, err := c.GetUserEvents(context.Background(), appID, "", 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	ev := events[0]
	assert.Equal(t, appID, ev.ApplicationID)
	assert.Equal(t, "deposit", ev.EventSubType)
	assert.Equal(t, []byte{0xca, 0xfe}, ev.EncryptedData)
	assert.Equal(t, uint64(100), ev.BlockNumber)
	assert.Equal(t, uint64(3), ev.LogIndex)

	var expectedReqID common.RequestIdType
	expectedReqID[31] = 1
	assert.Equal(t, expectedReqID, ev.RequestID)
}

// TestClient_GetUserEvents_Empty verifies that an empty result set returns
// an empty slice without error.
func TestClient_GetUserEvents_Empty(t *testing.T) {
	srv := fakeSubgraph(t, 200, map[string]interface{}{
		"data": map[string]interface{}{
			"userEvents": []map[string]interface{}{},
		},
	})

	c := NewClient(srv.URL)
	events, err := c.GetUserEvents(context.Background(), common.NewApplicationId(1), "", 10, nil)
	require.NoError(t, err)
	assert.Empty(t, events)
}
