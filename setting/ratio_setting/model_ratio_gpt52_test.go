package ratio_setting

import "testing"

func TestGPT52DefaultModelRatios(t *testing.T) {
	InitRatioSettings()

	cases := map[string]float64{
		"gpt-5.2":                0.875,
		"gpt-5.2-2025-12-11":     0.875,
		"gpt-5.2-chat-latest":    0.875,
		"gpt-5.2-codex":          0.875,
		"gpt-5.2-pro":            10.5,
		"gpt-5.2-pro-2025-12-11": 10.5,
	}

	for modelName, expected := range cases {
		got, ok, _ := GetModelRatio(modelName)
		if !ok {
			t.Fatalf("expected model ratio for %s to exist", modelName)
		}
		if got != expected {
			t.Fatalf("unexpected model ratio for %s: got %v want %v", modelName, got, expected)
		}
	}
}

func TestGPT52DefaultCacheRatios(t *testing.T) {
	InitRatioSettings()

	cases := map[string]float64{
		"gpt-5.2":             0.1,
		"gpt-5.2-2025-12-11":  0.1,
		"gpt-5.2-chat-latest": 0.1,
		"gpt-5.2-codex":       0.1,
	}

	for modelName, expected := range cases {
		got, ok := GetCacheRatio(modelName)
		if !ok {
			t.Fatalf("expected cache ratio for %s to exist", modelName)
		}
		if got != expected {
			t.Fatalf("unexpected cache ratio for %s: got %v want %v", modelName, got, expected)
		}
	}
}

func TestGPT52CompletionRatios(t *testing.T) {
	InitRatioSettings()

	cases := map[string]float64{
		"gpt-5.2":             8,
		"gpt-5.2-codex":       8,
		"gpt-5.2-pro":         8,
		"gpt-5.2-chat-latest": 8,
	}

	for modelName, expected := range cases {
		got := GetCompletionRatio(modelName)
		if got != expected {
			t.Fatalf("unexpected completion ratio for %s: got %v want %v", modelName, got, expected)
		}
	}
}

func TestGPT54MiniDefaultPricing(t *testing.T) {
	InitRatioSettings()

	modelRatio, ok, _ := GetModelRatio("gpt-5.4-mini")
	if !ok {
		t.Fatal("expected model ratio for gpt-5.4-mini to exist")
	}
	if modelRatio != 0.375 {
		t.Fatalf("unexpected model ratio for gpt-5.4-mini: got %v want %v", modelRatio, 0.375)
	}

	cacheRatio, ok := GetCacheRatio("gpt-5.4-mini")
	if !ok {
		t.Fatal("expected cache ratio for gpt-5.4-mini to exist")
	}
	if cacheRatio != 0.1 {
		t.Fatalf("unexpected cache ratio for gpt-5.4-mini: got %v want %v", cacheRatio, 0.1)
	}

	completionRatio := GetCompletionRatio("gpt-5.4-mini")
	if completionRatio != 6 {
		t.Fatalf("unexpected completion ratio for gpt-5.4-mini: got %v want %v", completionRatio, 6.0)
	}
}
