// cmd/clickup/task/update_test.go
package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCmd_ArchiveAndUnarchiveTogether(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--archive", "--unarchive"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestUpdateCmd_NoFlagsProvided(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified")
}

func isolatedHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("CLICKUP_CONFIG", "")
}

func TestUpdateCmd_ArchiveFlag_ValidationPasses(t *testing.T) {
	isolatedHome(t)
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--archive"})

	err := cmd.Execute()
	// Validation passes; config load fails because no config file exists.
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "no fields specified")
	assert.NotContains(t, err.Error(), "cannot be used together")
}

func TestUpdateCmd_UnarchiveFlag_ValidationPasses(t *testing.T) {
	isolatedHome(t)
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--unarchive"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "no fields specified")
	assert.NotContains(t, err.Error(), "cannot be used together")
}

func TestUpdateCmd_WithNameFlag_ValidationPasses(t *testing.T) {
	isolatedHome(t)
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--name", "New Task Name"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "no fields specified")
	assert.NotContains(t, err.Error(), "cannot be used together")
}

func TestUpdateCmd_WithClearFlag_ValidationPasses(t *testing.T) {
	isolatedHome(t)
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--clear", "description"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "no fields specified")
	assert.NotContains(t, err.Error(), "cannot be used together")
}
