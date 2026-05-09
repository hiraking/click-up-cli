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

func TestUpdateCmd_ArchiveFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--archive"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_UnarchiveFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--unarchive"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_WithNameFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--name", "New Task Name"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateCmd_WithClearFlag_ValidationPasses(t *testing.T) {
	configPath := ""
	cmd := newUpdateCmd(&configPath)
	cmd.SetArgs([]string{"task123", "--clear", "description"})

	err := cmd.Execute()
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}
