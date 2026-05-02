package ratio_setting

import "testing"

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

func TestGPT55DefaultPricing(t *testing.T) {
	InitRatioSettings()

	modelRatio, ok, _ := GetModelRatio("gpt-5.5")
	if !ok {
		t.Fatal("expected model ratio for gpt-5.5 to exist")
	}
	if modelRatio != 2.5 {
		t.Fatalf("unexpected model ratio for gpt-5.5: got %v want %v", modelRatio, 2.5)
	}

	cacheRatio, ok := GetCacheRatio("gpt-5.5")
	if !ok {
		t.Fatal("expected cache ratio for gpt-5.5 to exist")
	}
	if cacheRatio != 0.1 {
		t.Fatalf("unexpected cache ratio for gpt-5.5: got %v want %v", cacheRatio, 0.1)
	}

	createCacheRatio, ok := GetCreateCacheRatio("gpt-5.5")
	if !ok {
		t.Fatal("expected create cache ratio for gpt-5.5 to exist")
	}
	if createCacheRatio != 1 {
		t.Fatalf("unexpected create cache ratio for gpt-5.5: got %v want %v", createCacheRatio, 1.0)
	}

	completionRatio := GetCompletionRatio("gpt-5.5")
	if completionRatio != 6 {
		t.Fatalf("unexpected completion ratio for gpt-5.5: got %v want %v", completionRatio, 6.0)
	}
}
