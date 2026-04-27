// internal/client/client_test.go
package client

import (
	"testing"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/stretchr/testify/assert"
)

func ptr(s string) *string { return &s }

func TestFilterByQuery_EmptyQuery(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug"},
		{Name: "Write tests"},
	}
	result := filterByQuery(tasks, "")
	assert.Equal(t, tasks, result)
}

func TestFilterByQuery_MatchesName(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug in login"},
		{Name: "Write documentation"},
	}
	result := filterByQuery(tasks, "bug")
	assert.Len(t, result, 1)
	assert.Equal(t, "Fix bug in login", result[0].Name)
}

func TestFilterByQuery_MatchesDescription(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Task A", Description: ptr("バグ修正が必要")},
		{Name: "Task B", Description: ptr("ドキュメント更新")},
	}
	result := filterByQuery(tasks, "バグ")
	assert.Len(t, result, 1)
	assert.Equal(t, "Task A", result[0].Name)
}

func TestFilterByQuery_CaseInsensitive(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix Bug"},
		{Name: "write tests"},
	}
	result := filterByQuery(tasks, "BUG")
	assert.Len(t, result, 1)
	assert.Equal(t, "Fix Bug", result[0].Name)
}

func TestFilterByQuery_NilDescription(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Task A", Description: nil},
	}
	// Description が nil でもパニックしないこと、かつ名前でマッチすること
	result := filterByQuery(tasks, "Task A")
	assert.Len(t, result, 1)

	// Description にしか存在しないクエリ → nil Description でもパニックせずマッチしないこと
	result2 := filterByQuery(tasks, "description content")
	assert.Empty(t, result2)
}

func TestFilterByQuery_NoMatch(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug"},
		{Name: "Write tests"},
	}
	result := filterByQuery(tasks, "deploy")
	assert.Empty(t, result)
}

func TestFilterByQuery_NoFalsePositiveAcrossFields(t *testing.T) {
	tasks := []models.TaskSummary{
		{Name: "Fix bug", Description: ptr("login issue")},
	}
	// "bug l" は名前末尾と説明先頭をまたぐクエリ → マッチしないこと
	result := filterByQuery(tasks, "bug l")
	assert.Empty(t, result)
}
