package service

import "strings"

const (
	ClaudeCodeCompatModelAlias    = "claude-opus-4-6"
	ClaudeCodeCompatUpstreamModel = "gpt-5.4"
)

func NormalizeClaudeCodeCompatModel(model string) (string, bool) {
	if strings.TrimSpace(model) == ClaudeCodeCompatModelAlias {
		return ClaudeCodeCompatUpstreamModel, true
	}
	return model, false
}
