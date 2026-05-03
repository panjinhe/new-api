package relay

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	appconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

func ResponsesHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	stageStart := time.Now()
	info.InitChannelMeta(c)
	info.RecordGatewayStage("init_channel_meta", stageStart)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		switch info.ApiType {
		case appconstant.APITypeOpenAI, appconstant.APITypeCodex:
		default:
			return types.NewErrorWithStatusCode(
				fmt.Errorf("unsupported endpoint %q for api type %d", "/v1/responses/compact", info.ApiType),
				types.ErrorCodeInvalidRequest,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			)
		}
	}

	var responsesReq *dto.OpenAIResponsesRequest
	switch req := info.Request.(type) {
	case *dto.OpenAIResponsesRequest:
		responsesReq = req
	case *dto.OpenAIResponsesCompactionRequest:
		responsesReq = &dto.OpenAIResponsesRequest{
			Model:                req.Model,
			Input:                req.Input,
			Instructions:         req.Instructions,
			PreviousResponseID:   req.PreviousResponseID,
			Store:                req.Store,
			PromptCacheKey:       req.PromptCacheKey,
			PromptCacheRetention: req.PromptCacheRetention,
		}
	default:
		return types.NewErrorWithStatusCode(
			fmt.Errorf("invalid request type, expected dto.OpenAIResponsesRequest or dto.OpenAIResponsesCompactionRequest, got %T", info.Request),
			types.ErrorCodeInvalidRequest,
			http.StatusBadRequest,
			types.ErrOptionWithSkipRetry(),
		)
	}

	stageStart = time.Now()
	err := helper.ModelMappedHelper(c, info, responsesReq)
	info.RecordGatewayStage("model_map", stageStart)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)
	var requestBody io.Reader
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		stageStart = time.Now()
		storage, err := common.GetBodyStorage(c)
		info.RecordGatewayStage("pass_through_body", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}
		requestBody = common.ReaderOnly(storage)
	} else if shouldUseRawOpenAIResponsesBody(info) {
		stageStart = time.Now()
		jsonData, err := buildRawOpenAIResponsesBody(c, info)
		info.RecordGatewayStage("responses_raw_body", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeReadRequestBodyFailed, types.ErrOptionWithSkipRetry())
		}

		stageStart = time.Now()
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
		info.RecordGatewayStage("remove_disabled_fields", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		if len(info.ParamOverride) > 0 {
			stageStart = time.Now()
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			info.RecordGatewayStage("param_override", stageStart)
			if err != nil {
				return newAPIErrorFromParamOverride(err)
			}
		}

		if common.DebugEnabled {
			println("requestBody: ", string(jsonData))
		}
		requestBody = bytes.NewBuffer(jsonData)
	} else {
		stageStart = time.Now()
		convertedRequest, err := adaptor.ConvertOpenAIResponsesRequest(c, info, *responsesReq)
		info.RecordGatewayStage("responses_convert", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
		stageStart = time.Now()
		jsonData, err := common.Marshal(convertedRequest)
		info.RecordGatewayStage("marshal_request", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// remove disabled fields for OpenAI Responses API
		stageStart = time.Now()
		jsonData, err = relaycommon.RemoveDisabledFields(jsonData, info.ChannelOtherSettings, info.ChannelSetting.PassThroughBodyEnabled)
		info.RecordGatewayStage("remove_disabled_fields", stageStart)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		// apply param override
		if len(info.ParamOverride) > 0 {
			stageStart = time.Now()
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			info.RecordGatewayStage("param_override", stageStart)
			if err != nil {
				return newAPIErrorFromParamOverride(err)
			}
		}

		if common.DebugEnabled {
			println("requestBody: ", string(jsonData))
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	var httpResp *http.Response
	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")

	if resp != nil {
		httpResp = resp.(*http.Response)

		if httpResp.StatusCode != http.StatusOK {
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			// reset status code 重置状态码
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}

	usageDto := usage.(*dto.Usage)
	if info.RelayMode == relayconstant.RelayModeResponsesCompact {
		originModelName := info.OriginModelName
		originPriceData := info.PriceData

		_, err := helper.ModelPriceHelper(c, info, info.GetEstimatePromptTokens(), &types.TokenCountMeta{})
		if err != nil {
			info.OriginModelName = originModelName
			info.PriceData = originPriceData
			return types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry(), types.ErrOptionWithStatusCode(http.StatusBadRequest))
		}
		service.PostTextConsumeQuota(c, info, usageDto, nil)

		info.OriginModelName = originModelName
		info.PriceData = originPriceData
		return nil
	}

	if strings.HasPrefix(info.OriginModelName, "gpt-4o-audio") {
		service.PostAudioConsumeQuota(c, info, usageDto, "")
	} else {
		service.PostTextConsumeQuota(c, info, usageDto, nil)
	}
	return nil
}

func shouldUseRawOpenAIResponsesBody(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}
	if info.RelayMode != relayconstant.RelayModeResponses || info.RelayFormat != types.RelayFormatOpenAIResponses {
		return false
	}
	if info.ApiType != appconstant.APITypeOpenAI || info.IsModelMapped {
		return false
	}
	switch info.ChannelType {
	case appconstant.ChannelTypeOpenAI, appconstant.ChannelTypeCustom:
		return true
	default:
		return false
	}
}

func buildRawOpenAIResponsesBody(c *gin.Context, info *relaycommon.RelayInfo) ([]byte, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	jsonData, err := storage.Bytes()
	if err != nil {
		return nil, err
	}
	return patchRawOpenAIResponsesReasoning(jsonData, info)
}

func patchRawOpenAIResponsesReasoning(jsonData []byte, info *relaycommon.RelayInfo) ([]byte, error) {
	req, _ := info.Request.(*dto.OpenAIResponsesRequest)
	model := ""
	if req != nil {
		model = req.Model
	}
	effort, originModel := parseResponsesReasoningEffortFromModelSuffix(model)
	if effort == "" {
		if req != nil && req.Reasoning != nil && req.Reasoning.Effort != "" {
			info.ReasoningEffort = req.Reasoning.Effort
		}
		return jsonData, nil
	}

	patched, err := sjson.SetBytes(jsonData, "model", originModel)
	if err != nil {
		return nil, err
	}
	patched, err = sjson.SetBytes(patched, "reasoning.effort", effort)
	if err != nil {
		return nil, err
	}
	info.ReasoningEffort = effort
	return patched, nil
}

func parseResponsesReasoningEffortFromModelSuffix(model string) (string, string) {
	effortSuffixes := []string{"-high", "-minimal", "-low", "-medium", "-none", "-xhigh"}
	for _, suffix := range effortSuffixes {
		if strings.HasSuffix(model, suffix) {
			return strings.TrimPrefix(suffix, "-"), strings.TrimSuffix(model, suffix)
		}
	}
	return "", model
}
