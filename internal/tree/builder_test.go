// internal/tree/builder_test.go
package tree_test

import (
	"testing"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/hiraking/click-up-cli/internal/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTask(id string, parentID *string) models.TaskSummary {
	return models.TaskSummary{
		ID:       id,
		Name:     "Task " + id,
		ParentID: parentID,
		Subtasks: []models.TaskSummary{},
	}
}

func strPtr(s string) *string { return &s }

func TestBuild_EmptyInput(t *testing.T) {
	result := tree.Build(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestBuild_AllRoots(t *testing.T) {
	tasks := []models.TaskSummary{
		makeTask("a", nil),
		makeTask("b", nil),
	}
	result := tree.Build(tasks)
	assert.Len(t, result, 2)
}

func TestBuild_SingleLevel(t *testing.T) {
	parent := makeTask("parent", nil)
	child1 := makeTask("child1", strPtr("parent"))
	child2 := makeTask("child2", strPtr("parent"))

	result := tree.Build([]models.TaskSummary{parent, child1, child2})

	require.Len(t, result, 1)
	assert.Equal(t, "parent", result[0].ID)
	require.Len(t, result[0].Subtasks, 2)
}

func TestBuild_MultiLevel(t *testing.T) {
	grandparent := makeTask("gp", nil)
	parent := makeTask("p", strPtr("gp"))
	child := makeTask("c", strPtr("p"))

	result := tree.Build([]models.TaskSummary{grandparent, parent, child})

	require.Len(t, result, 1)
	gp := result[0]
	require.Len(t, gp.Subtasks, 1)
	p := gp.Subtasks[0]
	require.Len(t, p.Subtasks, 1)
	assert.Equal(t, "c", p.Subtasks[0].ID)
}

func TestBuild_OrphanedTaskBecomesRoot(t *testing.T) {
	// 親が存在しないタスクはルートとして扱う
	orphan := makeTask("orphan", strPtr("nonexistent"))

	result := tree.Build([]models.TaskSummary{orphan})
	assert.Len(t, result, 1)
	assert.Equal(t, "orphan", result[0].ID)
}
