package service

import "testing"

func TestNormalizeClaudeCodeCompatModel(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		want   string
		mapped bool
	}{
		{
			name:   "maps claude code compatibility alias to gpt-5.4",
			model:  "claude-opus-4-6",
			want:   "gpt-5.4",
			mapped: true,
		},
		{
			name:   "keeps normal model unchanged",
			model:  "gpt-5.4",
			want:   "gpt-5.4",
			mapped: false,
		},
		{
			name:   "does not map unrelated claude model",
			model:  "claude-opus-4-7",
			want:   "claude-opus-4-7",
			mapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, mapped := NormalizeClaudeCodeCompatModel(tt.model)
			if got != tt.want {
				t.Fatalf("expected model %q, got %q", tt.want, got)
			}
			if mapped != tt.mapped {
				t.Fatalf("expected mapped=%t, got %t", tt.mapped, mapped)
			}
		})
	}
}
