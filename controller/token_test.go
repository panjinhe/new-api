package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type tokenAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type tokenPageResponse struct {
	Items []tokenResponseItem `json:"items"`
}

type tokenResponseItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Key    string `json:"key"`
	Status int    `json:"status"`
}

type tokenKeyResponse struct {
	Key string `json:"key"`
}

func setupTokenControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	if err := db.AutoMigrate(&model.Token{}); err != nil {
		t.Fatalf("failed to migrate token table: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedToken(t *testing.T, db *gorm.DB, userID int, name string, rawKey string) *model.Token {
	t.Helper()

	token := &model.Token{
		UserId:         userID,
		Name:           name,
		Key:            rawKey,
		Status:         common.TokenStatusEnabled,
		CreatedTime:    1,
		AccessedTime:   1,
		ExpiredTime:    -1,
		RemainQuota:    100,
		UnlimitedQuota: true,
		Group:          "default",
	}
	if err := db.Create(token).Error; err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	return token
}

func newAuthenticatedContext(t *testing.T, method string, target string, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var requestBody *bytes.Reader
	if body != nil {
		payload, err := common.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		requestBody = bytes.NewReader(payload)
	} else {
		requestBody = bytes.NewReader(nil)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, target, requestBody)
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	ctx.Set("id", userID)
	return ctx, recorder
}

func decodeAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) tokenAPIResponse {
	t.Helper()

	var response tokenAPIResponse
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode api response: %v", err)
	}
	return response
}

func TestGetAllTokensMasksKeyInResponse(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "list-token", "abcd1234efgh5678")
	seedToken(t, db, 2, "other-user-token", "zzzz1234yyyy5678")

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/?p=1&size=10", nil, 1)
	GetAllTokens(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var page tokenPageResponse
	if err := common.Unmarshal(response.Data, &page); err != nil {
		t.Fatalf("failed to decode token page response: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected exactly one token, got %d", len(page.Items))
	}
	if page.Items[0].Key != token.GetMaskedKey() {
		t.Fatalf("expected masked key %q, got %q", token.GetMaskedKey(), page.Items[0].Key)
	}
	if strings.Contains(recorder.Body.String(), token.Key) {
		t.Fatalf("list response leaked raw token key: %s", recorder.Body.String())
	}
}

func TestSearchTokensMasksKeyInResponse(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "searchable-token", "ijkl1234mnop5678")

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/search?keyword=searchable-token&p=1&size=10", nil, 1)
	SearchTokens(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var page tokenPageResponse
	if err := common.Unmarshal(response.Data, &page); err != nil {
		t.Fatalf("failed to decode search response: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected exactly one search result, got %d", len(page.Items))
	}
	if page.Items[0].Key != token.GetMaskedKey() {
		t.Fatalf("expected masked search key %q, got %q", token.GetMaskedKey(), page.Items[0].Key)
	}
	if strings.Contains(recorder.Body.String(), token.Key) {
		t.Fatalf("search response leaked raw token key: %s", recorder.Body.String())
	}
}

func TestGetTokenMasksKeyInResponse(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "detail-token", "qrst1234uvwx5678")

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/token/"+strconv.Itoa(token.Id), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	GetToken(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var detail tokenResponseItem
	if err := common.Unmarshal(response.Data, &detail); err != nil {
		t.Fatalf("failed to decode token detail response: %v", err)
	}
	if detail.Key != token.GetMaskedKey() {
		t.Fatalf("expected masked detail key %q, got %q", token.GetMaskedKey(), detail.Key)
	}
	if strings.Contains(recorder.Body.String(), token.Key) {
		t.Fatalf("detail response leaked raw token key: %s", recorder.Body.String())
	}
}

func TestUpdateTokenMasksKeyInResponse(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "editable-token", "yzab1234cdef5678")

	body := map[string]any{
		"id":                   token.Id,
		"name":                 "updated-token",
		"expired_time":         -1,
		"remain_quota":         100,
		"unlimited_quota":      true,
		"model_limits_enabled": false,
		"model_limits":         "",
		"group":                "default",
		"cross_group_retry":    false,
	}

	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/token/", body, 1)
	UpdateToken(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var detail tokenResponseItem
	if err := common.Unmarshal(response.Data, &detail); err != nil {
		t.Fatalf("failed to decode token update response: %v", err)
	}
	if detail.Key != token.GetMaskedKey() {
		t.Fatalf("expected masked update key %q, got %q", token.GetMaskedKey(), detail.Key)
	}
	if strings.Contains(recorder.Body.String(), token.Key) {
		t.Fatalf("update response leaked raw token key: %s", recorder.Body.String())
	}
}

func TestGetTokenKeyRequiresOwnershipAndReturnsFullKey(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	token := seedToken(t, db, 1, "owned-token", "owner1234token5678")

	authorizedCtx, authorizedRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/"+strconv.Itoa(token.Id)+"/key", nil, 1)
	authorizedCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	GetTokenKey(authorizedCtx)

	authorizedResponse := decodeAPIResponse(t, authorizedRecorder)
	if !authorizedResponse.Success {
		t.Fatalf("expected authorized key fetch to succeed, got message: %s", authorizedResponse.Message)
	}

	var keyData tokenKeyResponse
	if err := common.Unmarshal(authorizedResponse.Data, &keyData); err != nil {
		t.Fatalf("failed to decode token key response: %v", err)
	}
	if keyData.Key != token.GetFullKey() {
		t.Fatalf("expected full key %q, got %q", token.GetFullKey(), keyData.Key)
	}

	unauthorizedCtx, unauthorizedRecorder := newAuthenticatedContext(t, http.MethodPost, "/api/token/"+strconv.Itoa(token.Id)+"/key", nil, 2)
	unauthorizedCtx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	GetTokenKey(unauthorizedCtx)

	unauthorizedResponse := decodeAPIResponse(t, unauthorizedRecorder)
	if unauthorizedResponse.Success {
		t.Fatalf("expected unauthorized key fetch to fail")
	}
	if strings.Contains(unauthorizedRecorder.Body.String(), token.Key) {
		t.Fatalf("unauthorized key response leaked raw token key: %s", unauthorizedRecorder.Body.String())
	}
}

func TestMaybeFastResponsesTokenMetaLargeRequestUsesBodyEstimate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	largeInput := strings.Repeat("a", largeResponsesFastTokenBytes)
	body := fmt.Sprintf(`{"model":"gpt-5.5","input":"%s","max_output_tokens":123}`, largeInput)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req := &dto.OpenAIResponsesRequest{
		Model:           "gpt-5.5",
		Input:           []byte(fmt.Sprintf(`"%s"`, largeInput)),
		MaxOutputTokens: common.GetPointer[uint](123),
	}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
	}

	meta, ok := maybeFastResponsesTokenMeta(ctx, req, info, false)
	if !ok {
		t.Fatalf("expected fast token estimate for large responses request")
	}
	expected := int(math.Ceil(float64(len(body)) / 4.0))
	if meta.FastEstimatedTokens != expected {
		t.Fatalf("expected %d fast estimate tokens, got %d", expected, meta.FastEstimatedTokens)
	}
	if meta.MaxTokens != 123 {
		t.Fatalf("expected max output tokens to be preserved, got %d", meta.MaxTokens)
	}
	if meta.CombineText != "" {
		t.Fatalf("expected fast path to avoid CombineText allocation")
	}
	if !info.FastTokenEstimateEnabled || info.FastTokenEstimateTokens != expected || info.FastTokenEstimateRatio != fastTokenPreconsumeRatio {
		t.Fatalf("unexpected relay fast estimate metadata: %+v", info)
	}
}

func TestMaybeFastResponsesTokenMetaSkipsSmallRequestAndSensitiveCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := `{"model":"gpt-5.5","input":"hello"}`
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	req := &dto.OpenAIResponsesRequest{Model: "gpt-5.5", Input: []byte(`"hello"`)}
	info := &relaycommon.RelayInfo{
		RelayMode:   relayconstant.RelayModeResponses,
		RelayFormat: types.RelayFormatOpenAIResponses,
	}

	if _, ok := maybeFastResponsesTokenMeta(ctx, req, info, false); ok {
		t.Fatalf("expected small responses request to use exact token meta")
	}
	if _, ok := maybeFastResponsesTokenMeta(ctx, req, info, true); ok {
		t.Fatalf("expected sensitive check to force exact token meta")
	}
}

func TestFastResponsesTokenEstimateUsesSafetyRatioForPreConsume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousModelPrice := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByJSONString(previousModelPrice)
	})
	if err := ratio_setting.UpdateModelPriceByJSONString(`{}`); err != nil {
		t.Fatalf("failed to clear model price settings: %v", err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set(string(constant.ContextKeyUserGroup), "default")
	ctx.Set(string(constant.ContextKeyUsingGroup), "default")
	ctx.Set(string(constant.ContextKeyOriginalModel), "gpt-5.5")
	info := &relaycommon.RelayInfo{
		OriginModelName:          "gpt-5.5",
		UserGroup:                "default",
		UsingGroup:               "default",
		FastTokenEstimateEnabled: true,
		FastTokenEstimateTokens:  1000,
		FastTokenEstimateRatio:   fastTokenPreconsumeRatio,
		UserSetting:              dto.UserSetting{AcceptUnsetRatioModel: true},
	}

	priceData, err := helper.ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	if err != nil {
		t.Fatalf("ModelPriceHelper returned error: %v", err)
	}
	expected := int(float64(1000) * fastTokenPreconsumeRatio * priceData.ModelRatio * priceData.GroupRatioInfo.GroupRatio)
	if priceData.QuotaToPreConsume != expected {
		t.Fatalf("expected quota %d with safety ratio, got %d", expected, priceData.QuotaToPreConsume)
	}
}

func BenchmarkOpenAIResponsesTokenMetaLargeExactVsFast(b *testing.B) {
	gin.SetMode(gin.TestMode)
	largeInput := strings.Repeat("a", largeResponsesFastTokenBytes)
	body := fmt.Sprintf(`{"model":"gpt-5.5","input":"%s","max_output_tokens":1024}`, largeInput)
	req := &dto.OpenAIResponsesRequest{
		Model:           "gpt-5.5",
		Input:           []byte(fmt.Sprintf(`"%s"`, largeInput)),
		MaxOutputTokens: common.GetPointer[uint](1024),
	}

	b.Run("exact_meta", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			meta := req.GetTokenCountMeta()
			if meta.CombineText == "" {
				b.Fatal("expected exact meta to build combined text")
			}
		}
	})

	b.Run("fast_meta", func(b *testing.B) {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
		info := &relaycommon.RelayInfo{
			RelayMode:   relayconstant.RelayModeResponses,
			RelayFormat: types.RelayFormatOpenAIResponses,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			meta, ok := maybeFastResponsesTokenMeta(ctx, req, info, false)
			if !ok || meta.FastEstimatedTokens == 0 {
				b.Fatal("expected fast token estimate")
			}
		}
	})
}
