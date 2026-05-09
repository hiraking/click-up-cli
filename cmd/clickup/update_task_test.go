// cmd/clickup/update_task_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateTaskCmd_ArchiveAndUnarchiveTogether(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123", "--archive", "--unarchive"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestUpdateTaskCmd_NoFlagsProvided(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no fields specified")
}

func TestUpdateTaskCmd_ArchiveFlag_ValidationPasses(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123", "--archive"})

	err := cmd.Execute()
	// Validation should pass (no "no fields specified" or "cannot be used together" errors).
	// The error at this point will be from loadConfig or API call, which is expected.
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateTaskCmd_UnarchiveFlag_ValidationPasses(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123", "--unarchive"})

	err := cmd.Execute()
	// Validation should pass (no "no fields specified" or "cannot be used together" errors).
	// The error at this point will be from loadConfig or API call, which is expected.
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateTaskCmd_WithNameFlag_ValidationPasses(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123", "--name", "New Task Name"})

	err := cmd.Execute()
	// Validation should pass (no "no fields specified" error).
	// The error at this point will be from loadConfig or API call, which is expected.
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}

func TestUpdateTaskCmd_WithClearFlag_ValidationPasses(t *testing.T) {
	cmd := newUpdateTaskCmd()
	cmd.SetArgs([]string{"task123", "--clear", "description"})

	err := cmd.Execute()
	// Validation should pass (no "no fields specified" error).
	// The error at this point will be from loadConfig or API call, which is expected.
	if err != nil {
		assert.NotContains(t, err.Error(), "no fields specified")
		assert.NotContains(t, err.Error(), "cannot be used together")
	}
}
