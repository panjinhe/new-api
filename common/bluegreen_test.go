package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldRunLeaderTasks_NoColorConfigured(t *testing.T) {
	IsMasterNode = true
	t.Setenv("BACKGROUND_TASKS_ENABLED", "true")
	t.Setenv("INSTANCE_COLOR", "")
	t.Setenv("ACTIVE_COLOR_FILE", "")

	assert.True(t, ShouldRunLeaderTasks())
}

func TestShouldRunLeaderTasks_MatchedColor(t *testing.T) {
	IsMasterNode = true
	dir := t.TempDir()
	colorFile := filepath.Join(dir, "active-color")
	err := os.WriteFile(colorFile, []byte("green\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("BACKGROUND_TASKS_ENABLED", "true")
	t.Setenv("INSTANCE_COLOR", "green")
	t.Setenv("ACTIVE_COLOR_FILE", colorFile)

	assert.True(t, ShouldRunLeaderTasks())
}

func TestShouldRunLeaderTasks_UnmatchedColor(t *testing.T) {
	IsMasterNode = true
	dir := t.TempDir()
	colorFile := filepath.Join(dir, "active-color")
	err := os.WriteFile(colorFile, []byte("legacy\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("BACKGROUND_TASKS_ENABLED", "true")
	t.Setenv("INSTANCE_COLOR", "blue")
	t.Setenv("ACTIVE_COLOR_FILE", colorFile)

	assert.False(t, ShouldRunLeaderTasks())
}

func TestShouldRunLeaderTasks_BackgroundTasksDisabled(t *testing.T) {
	IsMasterNode = true
	t.Setenv("BACKGROUND_TASKS_ENABLED", "false")
	t.Setenv("INSTANCE_COLOR", "")
	t.Setenv("ACTIVE_COLOR_FILE", "")

	assert.False(t, ShouldRunLeaderTasks())
}

func TestShouldRunLeaderTasks_NonMasterNode(t *testing.T) {
	IsMasterNode = false
	t.Cleanup(func() {
		IsMasterNode = true
	})
	t.Setenv("BACKGROUND_TASKS_ENABLED", "true")
	t.Setenv("INSTANCE_COLOR", "")
	t.Setenv("ACTIVE_COLOR_FILE", "")

	assert.False(t, ShouldRunLeaderTasks())
}

func TestShouldRunLeaderTasks_ConfiguredColorMissingActiveFile(t *testing.T) {
	IsMasterNode = true
	t.Setenv("BACKGROUND_TASKS_ENABLED", "true")
	t.Setenv("INSTANCE_COLOR", "blue")
	t.Setenv("ACTIVE_COLOR_FILE", filepath.Join(t.TempDir(), "missing-active-color"))

	assert.False(t, ShouldRunLeaderTasks())
}
