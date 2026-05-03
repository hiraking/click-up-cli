// cmd/clickup/create_task_test.go
package main

import (
	"testing"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskType_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected models.TaskType
	}{
		{"milestone", models.TaskTypeMilestone},
		{"project", models.TaskTypeProject},
		{"book", models.TaskTypeBook},
		{"MILESTONE", models.TaskTypeMilestone},
		{"Project", models.TaskTypeProject},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseTaskType(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseTaskType_InvalidValue(t *testing.T) {
	_, err := parseTaskType("foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid task type 'foo'")
	assert.Contains(t, err.Error(), "milestone, project, or book")
}
