// internal/models/task_summary.go
package models

import "time"

// TaskSummary はエージェント向けの整形済みタスク DTO。
// Subtasks に子タスクをネストして保持するツリーノードを兼ねる。
type TaskSummary struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Status      string        `json:"status"`
	Priority    *string       `json:"priority"`
	ParentID    *string       `json:"parentId"`
	URL         string        `json:"url"`
	StartDate   *time.Time    `json:"startDate,omitempty"`
	DueDate     *time.Time    `json:"dueDate"`
	Description *string       `json:"description"`
	ListID      string        `json:"listId"`
	ListName    string        `json:"listName"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Subtasks    []TaskSummary `json:"subtasks"`
}
