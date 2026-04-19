package service

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

func makeTestJWT(t *testing.T, payload string) string {
	t.Helper()
	encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return "header." + encoded + ".sig"
}

func TestResolveCodexOAuthKeyInputWithRefreshTokenString(t *testing.T) {
	token := makeTestJWT(t, `{"email":"tester@example.com","https://api.openai.com/auth":{"chatgpt_account_id":"acct_123"}}`)
	called := false

	key, err := resolveCodexOAuthKeyInput(context.Background(), "rt_test", "", func(ctx context.Context, refreshToken string, proxyURL string) (*CodexOAuthTokenResult, error) {
		called = true
		if refreshToken != "rt_test" {
			t.Fatalf("unexpected refresh token: %s", refreshToken)
		}
		return &CodexOAuthTokenResult{
			AccessToken:  token,
			RefreshToken: "rt_rotated",
			ExpiresAt:    time.Unix(1777777777, 0),
		}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected refresh flow to be used")
	}
	if key.AccessToken != token {
		t.Fatalf("unexpected access token: %s", key.AccessToken)
	}
	if key.RefreshToken != "rt_rotated" {
		t.Fatalf("unexpected refresh token: %s", key.RefreshToken)
	}
	if key.AccountID != "acct_123" {
		t.Fatalf("unexpected account_id: %s", key.AccountID)
	}
	if key.Email != "tester@example.com" {
		t.Fatalf("unexpected email: %s", key.Email)
	}
	if key.Type != "codex" {
		t.Fatalf("unexpected type: %s", key.Type)
	}
	if key.LastRefresh == "" || key.Expired == "" {
		t.Fatal("expected last_refresh and expired to be populated")
	}
}

func TestResolveCodexOAuthKeyInputWithFullJSONSkipsRefresh(t *testing.T) {
	token := makeTestJWT(t, `{"https://api.openai.com/auth":{"chatgpt_account_id":"acct_json"}}`)
	called := false

	key, err := resolveCodexOAuthKeyInput(context.Background(), `{"access_token":"`+token+`","account_id":"acct_json","refresh_token":"rt_keep"}`, "", func(ctx context.Context, refreshToken string, proxyURL string) (*CodexOAuthTokenResult, error) {
		called = true
		return nil, errors.New("should not be called")
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if called {
		t.Fatal("expected refresh flow to be skipped")
	}
	if key.AccountID != "acct_json" {
		t.Fatalf("unexpected account_id: %s", key.AccountID)
	}
	if key.RefreshToken != "rt_keep" {
		t.Fatalf("unexpected refresh token: %s", key.RefreshToken)
	}
}

func TestResolveCodexOAuthKeyInputWithRefreshOnlyJSON(t *testing.T) {
	token := makeTestJWT(t, `{"https://api.openai.com/auth":{"chatgpt_account_id":"acct_from_json"}}`)

	key, err := resolveCodexOAuthKeyInput(context.Background(), `{"refresh_token":"rt_json"}`, "", func(ctx context.Context, refreshToken string, proxyURL string) (*CodexOAuthTokenResult, error) {
		return &CodexOAuthTokenResult{
			AccessToken:  token,
			RefreshToken: "rt_json_rotated",
			ExpiresAt:    time.Unix(1888888888, 0),
		}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if key.AccountID != "acct_from_json" {
		t.Fatalf("unexpected account_id: %s", key.AccountID)
	}
	if key.RefreshToken != "rt_json_rotated" {
		t.Fatalf("unexpected refresh token: %s", key.RefreshToken)
	}
}

func TestResolveCodexOAuthKeyInputRejectsMissingFields(t *testing.T) {
	_, err := resolveCodexOAuthKeyInput(context.Background(), `{}`, "", func(ctx context.Context, refreshToken string, proxyURL string) (*CodexOAuthTokenResult, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error for missing access_token/account_id")
	}
}
