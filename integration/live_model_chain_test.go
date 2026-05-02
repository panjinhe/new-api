package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	liveBaseURLEnv             = "NEWAPI_LIVE_BASE_URL"
	liveAPIKeyEnv              = "NEWAPI_LIVE_API_KEY"
	liveOpenAIModelsEnv        = "NEWAPI_LIVE_OPENAI_MODELS"
	liveClaudeModelEnv         = "NEWAPI_LIVE_CLAUDE_MODEL"
	liveClaudeExpectedModelEnv = "NEWAPI_LIVE_CLAUDE_EXPECTED_MODEL"
	defaultClaudeAliasModel    = "claude-opus-4-6"
	defaultClaudeExpectedModel = "gpt-5.4"
	liveRequestPrompt          = "Reply exactly with OK"
)

var defaultOpenAIModels = []string{
	"gpt-5.3-codex",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.5",
}

type liveTestConfig struct {
	baseURL             string
	apiKey              string
	openAIModels        []string
	claudeModel         string
	claudeExpectedModel string
	client              *http.Client
}

type modelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type claudeMessageResponse struct {
	Type       string `json:"type"`
	Role       string `json:"role"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage *struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type relayErrorResponse struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func TestLiveModelChains(t *testing.T) {
	cfg := loadLiveTestConfig(t)

	availableModels := fetchAvailableModels(t, cfg)
	availableSet := make(map[string]struct{}, len(availableModels))
	for _, model := range availableModels {
		availableSet[model] = struct{}{}
	}

	t.Run("openai_model_catalog", func(t *testing.T) {
		for _, model := range cfg.openAIModels {
			if _, ok := availableSet[model]; !ok {
				t.Fatalf("model %q not found in /v1/models, available=%v", model, availableModels)
			}
		}
	})

	t.Run("openai_chat_completions", func(t *testing.T) {
		for _, model := range cfg.openAIModels {
			model := model
			t.Run(model, func(t *testing.T) {
				body := map[string]any{
					"model": model,
					"messages": []map[string]any{
						{
							"role":    "user",
							"content": liveRequestPrompt,
						},
					},
					"max_tokens": 16,
				}

				statusCode, responseBody := doJSONRequest(
					t,
					cfg,
					http.MethodPost,
					"/v1/chat/completions",
					map[string]string{
						"Authorization": "Bearer " + cfg.apiKey,
						"Content-Type":  "application/json",
					},
					body,
				)
				if statusCode != http.StatusOK {
					t.Fatalf("unexpected status=%d body=%s", statusCode, formatErrorBody(responseBody))
				}

				var response openAIChatResponse
				if err := common.Unmarshal(responseBody, &response); err != nil {
					t.Fatalf("unmarshal openai response: %v", err)
				}
				if len(response.Choices) == 0 {
					t.Fatalf("empty choices for model %q", model)
				}
				if got := strings.TrimSpace(response.Choices[0].Message.Content); got != "OK" {
					t.Fatalf("unexpected reply for model %q: %q", model, got)
				}
				if response.Choices[0].FinishReason == "" {
					t.Fatalf("finish_reason missing for model %q", model)
				}
			})
		}
	})

	t.Run("claude_messages_non_stream", func(t *testing.T) {
		body := map[string]any{
			"model":      cfg.claudeModel,
			"max_tokens": 16,
			"messages": []map[string]any{
				{
					"role":    "user",
					"content": liveRequestPrompt,
				},
			},
		}

		statusCode, responseBody := doJSONRequest(
			t,
			cfg,
			http.MethodPost,
			"/v1/messages",
			map[string]string{
				"x-api-key":         cfg.apiKey,
				"anthropic-version": "2023-06-01",
				"content-type":      "application/json",
			},
			body,
		)
		if statusCode != http.StatusOK {
			t.Fatalf("unexpected status=%d body=%s", statusCode, formatErrorBody(responseBody))
		}

		var response claudeMessageResponse
		if err := common.Unmarshal(responseBody, &response); err != nil {
			t.Fatalf("unmarshal claude response: %v", err)
		}
		if response.Type != "message" {
			t.Fatalf("unexpected claude response type: %q", response.Type)
		}
		if response.Role != "assistant" {
			t.Fatalf("unexpected claude response role: %q", response.Role)
		}
		if response.Model != cfg.claudeExpectedModel {
			t.Fatalf("unexpected mapped model: got=%q want=%q", response.Model, cfg.claudeExpectedModel)
		}
		if response.StopReason != "end_turn" {
			t.Fatalf("unexpected stop_reason: %q", response.StopReason)
		}
		if len(response.Content) == 0 {
			t.Fatal("claude response content is empty")
		}
		if got := strings.TrimSpace(response.Content[0].Text); got != "OK" {
			t.Fatalf("unexpected claude reply: %q", got)
		}
		if response.Usage == nil {
			t.Fatal("claude usage is missing")
		}
	})

	t.Run("claude_messages_stream", func(t *testing.T) {
		body := map[string]any{
			"model":      cfg.claudeModel,
			"max_tokens": 16,
			"stream":     true,
			"messages": []map[string]any{
				{
					"role":    "user",
					"content": liveRequestPrompt,
				},
			},
		}

		statusCode, responseBody := doJSONRequest(
			t,
			cfg,
			http.MethodPost,
			"/v1/messages",
			map[string]string{
				"x-api-key":         cfg.apiKey,
				"anthropic-version": "2023-06-01",
				"content-type":      "application/json",
			},
			body,
		)
		if statusCode != http.StatusOK {
			t.Fatalf("unexpected status=%d body=%s", statusCode, formatErrorBody(responseBody))
		}

		streamBody := string(responseBody)
		expectedEvents := []string{
			"event: message_start",
			"event: content_block_start",
			"event: content_block_delta",
			"event: content_block_stop",
			"event: message_delta",
			"event: message_stop",
		}
		lastIndex := -1
		for _, event := range expectedEvents {
			idx := strings.Index(streamBody, event)
			if idx == -1 {
				t.Fatalf("missing SSE event %q in body=%s", event, streamBody)
			}
			if idx <= lastIndex {
				t.Fatalf("SSE event %q is out of order in body=%s", event, streamBody)
			}
			lastIndex = idx
		}
		if !strings.Contains(streamBody, `"text":"OK"`) {
			t.Fatalf("stream body missing OK delta: %s", streamBody)
		}
		if !strings.Contains(streamBody, fmt.Sprintf(`"model":"%s"`, cfg.claudeExpectedModel)) {
			t.Fatalf("stream body missing mapped model %q: %s", cfg.claudeExpectedModel, streamBody)
		}
	})
}

func loadLiveTestConfig(t *testing.T) liveTestConfig {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping live model chain tests in short mode")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(liveBaseURLEnv)), "/")
	apiKey := strings.TrimSpace(os.Getenv(liveAPIKeyEnv))
	if baseURL == "" || apiKey == "" {
		t.Skipf("set %s and %s to run live model chain tests", liveBaseURLEnv, liveAPIKeyEnv)
	}

	openAIModels := splitCSVEnv(os.Getenv(liveOpenAIModelsEnv))
	if len(openAIModels) == 0 {
		openAIModels = append([]string(nil), defaultOpenAIModels...)
	}

	claudeModel := strings.TrimSpace(os.Getenv(liveClaudeModelEnv))
	if claudeModel == "" {
		claudeModel = defaultClaudeAliasModel
	}

	claudeExpectedModel := strings.TrimSpace(os.Getenv(liveClaudeExpectedModelEnv))
	if claudeExpectedModel == "" {
		claudeExpectedModel = defaultClaudeExpectedModel
	}

	return liveTestConfig{
		baseURL:             baseURL,
		apiKey:              apiKey,
		openAIModels:        openAIModels,
		claudeModel:         claudeModel,
		claudeExpectedModel: claudeExpectedModel,
		client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func fetchAvailableModels(t *testing.T, cfg liveTestConfig) []string {
	t.Helper()

	statusCode, body := doRequest(
		t,
		cfg,
		http.MethodGet,
		"/v1/models",
		map[string]string{
			"Authorization": "Bearer " + cfg.apiKey,
		},
		nil,
	)
	if statusCode != http.StatusOK {
		t.Fatalf("fetch /v1/models failed: status=%d body=%s", statusCode, formatErrorBody(body))
	}

	var response modelListResponse
	if err := common.Unmarshal(body, &response); err != nil {
		t.Fatalf("unmarshal /v1/models response: %v", err)
	}

	models := make([]string, 0, len(response.Data))
	for _, item := range response.Data {
		models = append(models, item.ID)
	}
	sort.Strings(models)
	return models
}

func doJSONRequest(
	t *testing.T,
	cfg liveTestConfig,
	method string,
	path string,
	headers map[string]string,
	body any,
) (int, []byte) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		payload, err := common.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	return doRequest(t, cfg, method, path, headers, bodyReader)
}

func doRequest(
	t *testing.T,
	cfg liveTestConfig,
	method string,
	path string,
	headers map[string]string,
	body io.Reader,
) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, cfg.baseURL+path, body)
	if err != nil {
		t.Fatalf("build request %s %s: %v", method, path, err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := cfg.client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body for %s %s: %v", method, path, err)
	}
	return resp.StatusCode, responseBody
}

func formatErrorBody(body []byte) string {
	var response relayErrorResponse
	if err := common.Unmarshal(body, &response); err == nil {
		if response.Error != nil && response.Error.Message != "" {
			return response.Error.Message
		}
		if response.Message != "" {
			return response.Message
		}
	}
	return strings.TrimSpace(string(body))
}

func splitCSVEnv(raw string) []string {
	items := strings.Split(raw, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
