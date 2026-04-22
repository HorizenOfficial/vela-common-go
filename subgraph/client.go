package subgraph

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/HorizenOfficial/vela-common-go/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type graphError struct {
	Message string `json:"message"`
}

type graphResponse[T any] struct {
	Data   T            `json:"data"`
	Errors []graphError `json:"errors"`
}

type client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient builds a subgraph client pointing to the given endpoint.
func NewClient(endpoint string) Client {
	return &client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type healthCheckResponse struct {
	Meta *struct {
		HasIndexingErrors bool `json:"hasIndexingErrors"`
	} `json:"_meta"`
}

type completedEntity struct {
	ApplicationID   string      `json:"applicationId"`
	RequestID       string      `json:"requestId"`
	Status          json.Number `json:"status"`
	ErrorCode       json.Number `json:"errorCode"`
	ErrorMessage    string      `json:"errorMessage"`
	ApplicationFees json.Number `json:"applicationFees"`
	BlockNumber     json.Number `json:"blockNumber"`
}

func (c *client) GetRequestCompletedByID(ctx context.Context, requestID common.RequestIdType) (*RequestCompleted, error) {
	query := `
query($requestId: Bytes!) {
  requestCompleteds(where: { requestId: $requestId }, first: 1) {
    applicationId
    requestId
    status
    errorCode
    errorMessage
    applicationFees
    blockNumber
  }
}`

	type response struct {
		RequestCompleteds []completedEntity `json:"requestCompleteds"`
	}
	var resp graphResponse[response]
	if err := c.doGraphQL(ctx, query, map[string]interface{}{"requestId": "0x" + requestID.String()}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}
	if len(resp.Data.RequestCompleteds) == 0 {
		return nil, nil
	}

	return parseCompletedEntity(resp.Data.RequestCompleteds[0], requestID)
}

func (c *client) GetDeployRequestCompletedByID(ctx context.Context, requestID common.RequestIdType) (*RequestCompleted, error) {
	query := `
query($requestId: Bytes!) {
  deployRequestCompleteds(where: { requestId: $requestId }, first: 1) {
    applicationId
    requestId
    status
    errorCode
    errorMessage
    applicationFees
    blockNumber
  }
}`

	type response struct {
		DeployRequestCompleteds []completedEntity `json:"deployRequestCompleteds"`
	}
	var resp graphResponse[response]
	if err := c.doGraphQL(ctx, query, map[string]interface{}{"requestId": "0x" + requestID.String()}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}
	if len(resp.Data.DeployRequestCompleteds) == 0 {
		return nil, nil
	}

	return parseCompletedEntity(resp.Data.DeployRequestCompleteds[0], requestID)
}

func parseCompletedEntity(entity completedEntity, requestID common.RequestIdType) (*RequestCompleted, error) {
	var appID uint64
	if entity.ApplicationID != "" {
		var err error
		appID, err = strconv.ParseUint(entity.ApplicationID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid applicationId %q: %w", entity.ApplicationID, err)
		}
	}

	statusUint, err := strconv.ParseUint(entity.Status.String(), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid status value %q: %w", entity.Status, err)
	}
	status, err := uint8ToRequestResultStatus(uint8(statusUint))
	if err != nil {
		return nil, err
	}

	errorCodeUint, err := strconv.ParseUint(entity.ErrorCode.String(), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid errorCode %q: %w", entity.ErrorCode, err)
	}

	appFees, ok := stringToBigInt(entity.ApplicationFees.String())
	if !ok {
		return nil, fmt.Errorf("invalid applicationFees %q", entity.ApplicationFees)
	}

	blockNumber, err := strconv.ParseUint(entity.BlockNumber.String(), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid blockNumber %q: %w", entity.BlockNumber, err)
	}

	return &RequestCompleted{
		ApplicationID:   common.ApplicationIdType(appID),
		RequestID:       requestID,
		Status:          status,
		ErrorCode:       uint8(errorCodeUint),
		ErrorMessage:    entity.ErrorMessage,
		ApplicationFees: appFees,
		BlockNumber:     blockNumber,
	}, nil
}

func (c *client) HealthCheck(ctx context.Context) error {
	query := `
query HealthCheck {
  _meta {
    hasIndexingErrors
  }
}`

	var resp graphResponse[healthCheckResponse]
	if err := c.doGraphQL(ctx, query, map[string]interface{}{}, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}
	if resp.Data.Meta == nil {
		return fmt.Errorf("subgraph health check returned empty meta")
	}
	if resp.Data.Meta.HasIndexingErrors {
		return fmt.Errorf("subgraph reports indexing errors")
	}
	return nil
}

type userEventEntity struct {
	ApplicationID string `json:"applicationId"`
	RequestID     string `json:"requestId"`
	EventSubType  string `json:"eventSubType"`
	EncryptedData string `json:"encryptedData"`
	BlockNumber   string `json:"blockNumber"`
	LogIndex      string `json:"logIndex"`
	SortKey       string `json:"sortKey"`
}

type userEventsResponse struct {
	UserEvents []userEventEntity `json:"userEvents"`
}

func (c *client) GetUserEvents(ctx context.Context, applicationID common.ApplicationIdType, eventSubType [32]byte, limit int, before *big.Int) ([]UserEvent, error) {
	var subTypes [][32]byte
	if eventSubType != ([32]byte{}) {
		subTypes = [][32]byte{eventSubType}
	}
	return c.GetUserEventsBySubTypes(ctx, applicationID, subTypes, limit, before)
}

func (c *client) GetUserEventsBySubTypes(ctx context.Context, applicationID common.ApplicationIdType, eventSubTypes [][32]byte, limit int, before *big.Int) ([]UserEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	variables := map[string]interface{}{
		"applicationId": fmt.Sprintf("%d", uint64(applicationID)),
		"limit":         limit,
	}

	varDefs := ""
	whereParts := []string{"applicationId: $applicationId"}
	if len(eventSubTypes) > 0 {
		hexSubTypes := make([]string, len(eventSubTypes))
		for i, st := range eventSubTypes {
			hexSubTypes[i] = "0x" + hex.EncodeToString(st[:])
		}
		varDefs += ", $eventSubTypes: [Bytes!]!"
		variables["eventSubTypes"] = hexSubTypes
		whereParts = append(whereParts, "eventSubType_in: $eventSubTypes")
	}
	if before != nil {
		varDefs += ", $before: BigInt!"
		variables["before"] = before.String()
		whereParts = append(whereParts, "sortKey_lt: $before")
	}

	query := fmt.Sprintf(`
query($applicationId: BigInt!, $limit: Int!%s) {
  userEvents(
    where: { %s }
    orderBy: sortKey
    orderDirection: desc
    first: $limit
  ) {
    applicationId
    requestId
    eventSubType
    encryptedData
    blockNumber
    logIndex
    sortKey
  }
}`, varDefs, strings.Join(whereParts, ", "))

	var resp graphResponse[userEventsResponse]
	if err := c.doGraphQL(ctx, query, variables, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}

	return parseUserEventEntities(applicationID, resp.Data.UserEvents)
}

func parseUserEventEntities(applicationID common.ApplicationIdType, entities []userEventEntity) ([]UserEvent, error) {
	events := make([]UserEvent, 0, len(entities))
	for _, entity := range entities {
		reqID, err := parseRequestID(entity.RequestID)
		if err != nil {
			return nil, fmt.Errorf("invalid requestId %q: %w", entity.RequestID, err)
		}

		data, err := decodeHex(entity.EncryptedData)
		if err != nil {
			return nil, fmt.Errorf("invalid encryptedData for request %s: %w", reqID.String(), err)
		}

		subType, err := decodeSubType(entity.EventSubType)
		if err != nil {
			return nil, fmt.Errorf("invalid eventSubType for request %s: %w", reqID.String(), err)
		}

		blockNumber, err := strconv.ParseUint(entity.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid blockNumber %q: %w", entity.BlockNumber, err)
		}

		logIndex, err := strconv.ParseUint(entity.LogIndex, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid logIndex %q: %w", entity.LogIndex, err)
		}

		sortKey, ok := stringToBigInt(entity.SortKey)
		if !ok {
			return nil, fmt.Errorf("invalid sortKey %q", entity.SortKey)
		}

		events = append(events, UserEvent{
			ApplicationID: applicationID,
			RequestID:     reqID,
			EventSubType:  subType,
			EncryptedData: data,
			BlockNumber:   blockNumber,
			LogIndex:      logIndex,
			SortKey:       sortKey,
		})
	}
	return events, nil
}

type appEventEntity struct {
	ApplicationID string `json:"applicationId"`
	RequestID     string `json:"requestId"`
	EventSubType  string `json:"eventSubType"`
	Data          string `json:"data"`
	BlockNumber   string `json:"blockNumber"`
	LogIndex      string `json:"logIndex"`
	SortKey       string `json:"sortKey"`
}

type appEventsResponse struct {
	AppEvents []appEventEntity `json:"appEvents"`
}

func (c *client) GetAppEvents(ctx context.Context, applicationID common.ApplicationIdType, eventSubType [32]byte, limit int, before *big.Int) ([]AppEvent, error) {
	var subTypes [][32]byte
	if eventSubType != ([32]byte{}) {
		subTypes = [][32]byte{eventSubType}
	}
	return c.GetAppEventsBySubTypes(ctx, applicationID, subTypes, limit, before)
}

func (c *client) GetAppEventsBySubTypes(ctx context.Context, applicationID common.ApplicationIdType, eventSubTypes [][32]byte, limit int, before *big.Int) ([]AppEvent, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 1000 {
		limit = 1000
	}

	variables := map[string]interface{}{
		"applicationId": fmt.Sprintf("%d", uint64(applicationID)),
		"limit":         limit,
	}

	varDefs := ""
	whereParts := []string{"applicationId: $applicationId"}
	if len(eventSubTypes) > 0 {
		hexSubTypes := make([]string, len(eventSubTypes))
		for i, st := range eventSubTypes {
			hexSubTypes[i] = "0x" + hex.EncodeToString(st[:])
		}
		varDefs += ", $eventSubTypes: [Bytes!]!"
		variables["eventSubTypes"] = hexSubTypes
		whereParts = append(whereParts, "eventSubType_in: $eventSubTypes")
	}
	if before != nil {
		varDefs += ", $before: BigInt!"
		variables["before"] = before.String()
		whereParts = append(whereParts, "sortKey_lt: $before")
	}

	query := fmt.Sprintf(`
query($applicationId: BigInt!, $limit: Int!%s) {
  appEvents(
    where: { %s }
    orderBy: sortKey
    orderDirection: desc
    first: $limit
  ) {
    applicationId
    requestId
    eventSubType
    data
    blockNumber
    logIndex
    sortKey
  }
}`, varDefs, strings.Join(whereParts, ", "))

	var resp graphResponse[appEventsResponse]
	if err := c.doGraphQL(ctx, query, variables, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}

	return parseAppEventEntities(applicationID, resp.Data.AppEvents)
}

func parseAppEventEntities(applicationID common.ApplicationIdType, entities []appEventEntity) ([]AppEvent, error) {
	events := make([]AppEvent, 0, len(entities))
	for _, entity := range entities {
		reqID, err := parseRequestID(entity.RequestID)
		if err != nil {
			return nil, fmt.Errorf("invalid requestId %q: %w", entity.RequestID, err)
		}

		data, err := decodeHex(entity.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid data for request %s: %w", reqID.String(), err)
		}

		subType, err := decodeSubType(entity.EventSubType)
		if err != nil {
			return nil, fmt.Errorf("invalid eventSubType for request %s: %w", reqID.String(), err)
		}

		blockNumber, err := strconv.ParseUint(entity.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid blockNumber %q: %w", entity.BlockNumber, err)
		}

		logIndex, err := strconv.ParseUint(entity.LogIndex, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid logIndex %q: %w", entity.LogIndex, err)
		}

		sortKey, ok := stringToBigInt(entity.SortKey)
		if !ok {
			return nil, fmt.Errorf("invalid sortKey %q", entity.SortKey)
		}

		events = append(events, AppEvent{
			ApplicationID: applicationID,
			RequestID:     reqID,
			EventSubType:  subType,
			Data:          data,
			BlockNumber:   blockNumber,
			LogIndex:      logIndex,
			SortKey:       sortKey,
		})
	}
	return events, nil
}

// decodeSubType parses an "0x"-prefixed hex string (up to 32 bytes, per the
// on-chain bytes32 topic) into a [32]byte. Shorter hex values are left-aligned
// and zero-padded on the right, matching how Solidity emits bytes32.
func decodeSubType(hexString string) ([32]byte, error) {
	var out [32]byte
	if hexString == "" {
		return out, nil
	}
	b, err := decodeHex(hexString)
	if err != nil {
		return out, err
	}
	if len(b) > 32 {
		return out, fmt.Errorf("eventSubType exceeds 32 bytes: got %d", len(b))
	}
	copy(out[:], b)
	return out, nil
}

func (c *client) doGraphQL(ctx context.Context, query string, variables map[string]interface{}, dest any) error {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal graphql payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		blob, _ := io.ReadAll(res.Body)
		return fmt.Errorf("graphql request returned status %d: %s", res.StatusCode, string(blob))
	}

	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("failed to decode graphql response: %w", err)
	}

	return nil
}

func parseRequestID(hexString string) (common.RequestIdType, error) {
	trimmed := strings.TrimPrefix(hexString, "0x")
	b, err := requestIdStringTo32Byte(trimmed)
	if err != nil {
		return common.RequestIdType{}, err
	}
	return common.RequestIdType(b), nil
}

func decodeHex(hexString string) ([]byte, error) {
	if hexString == "" {
		return nil, nil
	}
	trimmed := strings.TrimPrefix(hexString, "0x")
	return hex.DecodeString(trimmed)
}

// Unexported helpers inlined from vela/pkg/common/utils.go.

func uint8ToRequestResultStatus(i uint8) (common.RequestResultStatus, error) {
	switch i {
	case 0:
		return common.RequestResultOK, nil
	case 1:
		return common.RequestResultFailed, nil
	default:
		return common.RequestResultUnknown, fmt.Errorf("unknown request status value %d", i)
	}
}

func stringToBigInt(s string) (*big.Int, bool) {
	return new(big.Int).SetString(s, 10)
}

func requestIdStringTo32Byte(s string) ([32]byte, error) {
	arr, err := hex.DecodeString(s)
	if err != nil {
		return [32]byte{}, fmt.Errorf("requestId string is not a valid hex string: %w", err)
	}
	if len(arr) > 32 {
		return [32]byte{}, fmt.Errorf("requestId string must not be more than 32 bytes long, got %d", len(arr))
	}

	var arr32 [32]byte
	copy(arr32[:], arr)
	return arr32, nil
}

// --- ERC-20 entity queries ---

type refundEntity struct {
	ApplicationID string `json:"applicationId"`
	RequestID     string `json:"requestId"`
	To            string `json:"to"`
	TokenAddress  string `json:"tokenAddress"`
	Amount        string `json:"amount"`
	BlockNumber   string `json:"blockNumber"`
}

func (c *client) GetRefunds(ctx context.Context, applicationID common.ApplicationIdType, requestID *common.RequestIdType, limit int) ([]OnChainRefund, error) {
	if limit <= 0 {
		limit = 100
	}
	vars := map[string]interface{}{
		"appId": fmt.Sprintf("%d", applicationID),
		"first": limit,
	}

	where := `applicationId: $appId`
	if requestID != nil {
		vars["reqId"] = "0x" + hex.EncodeToString(requestID[:])
		where += `, requestId: $reqId`
	}

	query := fmt.Sprintf(`query($appId: String!, $first: Int!%s) {
		onChainRefunds(first: $first, where: {%s}, orderBy: blockNumber, orderDirection: desc) {
			applicationId requestId to tokenAddress amount blockNumber
		}
	}`, func() string {
		if requestID != nil {
			return ", $reqId: String!"
		}
		return ""
	}(), where)

	type response struct {
		OnChainRefunds []refundEntity `json:"onChainRefunds"`
	}
	var resp graphResponse[response]
	if err := c.doGraphQL(ctx, query, vars, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}

	result := make([]OnChainRefund, 0, len(resp.Data.OnChainRefunds))
	for _, e := range resp.Data.OnChainRefunds {
		reqID, err := parseRequestID(e.RequestID)
		if err != nil {
			return nil, fmt.Errorf("parsing refund requestId: %w", err)
		}
		appID, err := strconv.ParseUint(e.ApplicationID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing refund applicationId: %w", err)
		}
		amount, ok := stringToBigInt(e.Amount)
		if !ok {
			return nil, fmt.Errorf("parsing refund amount %q", e.Amount)
		}
		bn, err := strconv.ParseUint(e.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing refund blockNumber: %w", err)
		}
		result = append(result, OnChainRefund{
			ApplicationID: common.NewApplicationId(appID),
			RequestID:     reqID,
			To:            ethCommon.HexToAddress(e.To),
			TokenAddress:  ethCommon.HexToAddress(e.TokenAddress),
			Amount:        amount,
			BlockNumber:   bn,
		})
	}
	return result, nil
}

type withdrawalEntity struct {
	ApplicationID string `json:"applicationId"`
	RequestID     string `json:"requestId"`
	To            string `json:"to"`
	TokenAddress  string `json:"tokenAddress"`
	Amount        string `json:"amount"`
	BlockNumber   string `json:"blockNumber"`
}

func (c *client) GetWithdrawals(ctx context.Context, applicationID common.ApplicationIdType, requestID *common.RequestIdType, limit int) ([]OnChainWithdrawal, error) {
	if limit <= 0 {
		limit = 100
	}
	vars := map[string]interface{}{
		"appId": fmt.Sprintf("%d", applicationID),
		"first": limit,
	}

	where := `applicationId: $appId`
	if requestID != nil {
		vars["reqId"] = "0x" + hex.EncodeToString(requestID[:])
		where += `, requestId: $reqId`
	}

	query := fmt.Sprintf(`query($appId: String!, $first: Int!%s) {
		onChainWithdrawals(first: $first, where: {%s}, orderBy: blockNumber, orderDirection: desc) {
			applicationId requestId to tokenAddress amount blockNumber
		}
	}`, func() string {
		if requestID != nil {
			return ", $reqId: String!"
		}
		return ""
	}(), where)

	type response struct {
		OnChainWithdrawals []withdrawalEntity `json:"onChainWithdrawals"`
	}
	var resp graphResponse[response]
	if err := c.doGraphQL(ctx, query, vars, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}

	result := make([]OnChainWithdrawal, 0, len(resp.Data.OnChainWithdrawals))
	for _, e := range resp.Data.OnChainWithdrawals {
		reqID, err := parseRequestID(e.RequestID)
		if err != nil {
			return nil, fmt.Errorf("parsing withdrawal requestId: %w", err)
		}
		appID, err := strconv.ParseUint(e.ApplicationID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing withdrawal applicationId: %w", err)
		}
		amount, ok := stringToBigInt(e.Amount)
		if !ok {
			return nil, fmt.Errorf("parsing withdrawal amount %q", e.Amount)
		}
		bn, err := strconv.ParseUint(e.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing withdrawal blockNumber: %w", err)
		}
		result = append(result, OnChainWithdrawal{
			ApplicationID: common.NewApplicationId(appID),
			RequestID:     reqID,
			To:            ethCommon.HexToAddress(e.To),
			TokenAddress:  ethCommon.HexToAddress(e.TokenAddress),
			Amount:        amount,
			BlockNumber:   bn,
		})
	}
	return result, nil
}

type claimEntity struct {
	TokenAddress string `json:"tokenAddress"`
	Payee        string `json:"payee"`
	Amount       string `json:"amount"`
	BlockNumber  string `json:"blockNumber"`
}

func (c *client) GetClaimsExecuted(ctx context.Context, payee ethCommon.Address, tokenAddress *ethCommon.Address, limit int) ([]ClaimExecuted, error) {
	if limit <= 0 {
		limit = 100
	}
	vars := map[string]interface{}{
		"payee": strings.ToLower(payee.Hex()),
		"first": limit,
	}

	where := `payee: $payee`
	if tokenAddress != nil {
		vars["token"] = strings.ToLower(tokenAddress.Hex())
		where += `, tokenAddress: $token`
	}

	query := fmt.Sprintf(`query($payee: String!, $first: Int!%s) {
		claimExecuteds(first: $first, where: {%s}, orderBy: blockNumber, orderDirection: desc) {
			tokenAddress payee amount blockNumber
		}
	}`, func() string {
		if tokenAddress != nil {
			return ", $token: String!"
		}
		return ""
	}(), where)

	type response struct {
		ClaimExecuteds []claimEntity `json:"claimExecuteds"`
	}
	var resp graphResponse[response]
	if err := c.doGraphQL(ctx, query, vars, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("subgraph returned errors: %v", resp.Errors[0].Message)
	}

	result := make([]ClaimExecuted, 0, len(resp.Data.ClaimExecuteds))
	for _, e := range resp.Data.ClaimExecuteds {
		amount, ok := stringToBigInt(e.Amount)
		if !ok {
			return nil, fmt.Errorf("parsing claim amount %q", e.Amount)
		}
		bn, err := strconv.ParseUint(e.BlockNumber, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing claim blockNumber: %w", err)
		}
		result = append(result, ClaimExecuted{
			TokenAddress: ethCommon.HexToAddress(e.TokenAddress),
			Payee:        ethCommon.HexToAddress(e.Payee),
			Amount:       amount,
			BlockNumber:  bn,
		})
	}
	return result, nil
}
