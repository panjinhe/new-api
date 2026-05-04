package common

import (
	"os"
	"strings"
)

// ShouldRunLeaderTasks returns true when this instance should execute
// cluster-wide maintenance tasks.
//
// The current instance is considered active when either:
// 1. no INSTANCE_COLOR is configured, preserving legacy single-instance behavior; or
// 2. ACTIVE_COLOR_FILE is missing/empty, preserving bootstrapping behavior; or
// 3. the trimmed contents of ACTIVE_COLOR_FILE match INSTANCE_COLOR.
func ShouldRunLeaderTasks() bool {
	if !GetEnvOrDefaultBool("BACKGROUND_TASKS_ENABLED", true) {
		return false
	}
	if !IsMasterNode {
		return false
	}

	instanceColor := strings.TrimSpace(os.Getenv("INSTANCE_COLOR"))
	if instanceColor == "" {
		return true
	}

	activeColorFile := strings.TrimSpace(os.Getenv("ACTIVE_COLOR_FILE"))
	if activeColorFile == "" {
		return true
	}

	data, err := os.ReadFile(activeColorFile)
	if err != nil {
		return false
	}

	activeColor := strings.ToLower(strings.TrimSpace(string(data)))
	if activeColor == "" {
		return false
	}

	return strings.EqualFold(activeColor, instanceColor)
}
