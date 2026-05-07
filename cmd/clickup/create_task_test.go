// cmd/clickup/create_task_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupTaskType_ValidKey(t *testing.T) {
	taskTypes := map[string]int{"milestone": 1, "project": 1001}

	id, err := lookupTaskType(taskTypes, "milestone")
	require.NoError(t, err)
	assert.Equal(t, 1, id)
}

func TestLookupTaskType_UnknownKey(t *testing.T) {
	taskTypes := map[string]int{"milestone": 1, "project": 1001}

	_, err := lookupTaskType(taskTypes, "foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unknown task type 'foo'")
	assert.Contains(t, err.Error(), "milestone")
	assert.Contains(t, err.Error(), "project")
}

func TestLookupTaskType_EmptyConfig(t *testing.T) {
	_, err := lookupTaskType(nil, "milestone")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No task types configured")
}

func TestLookupTaskType_KeysSorted(t *testing.T) {
	taskTypes := map[string]int{"zebra": 3, "alpha": 1, "milestone": 2}

	_, err := lookupTaskType(taskTypes, "unknown")
	require.Error(t, err)
	// Available list should be alphabetically sorted
	assert.Contains(t, err.Error(), "alpha, milestone, zebra")
}
