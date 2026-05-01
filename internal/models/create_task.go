// internal/models/create_task.go
package models

import "time"

// TaskPriority は ClickUp の優先度を表す。API には int として送信する。
type TaskPriority int

const (
	PriorityUrgent TaskPriority = 1
	PriorityHigh   TaskPriority = 2
	PriorityNormal TaskPriority = 3
	PriorityLow    TaskPriority = 4
)

// TaskType は ClickUp のカスタムタスクタイプを表す。API には custom_item_id として送信する。
type TaskType int

const (
	TaskTypeMilestone TaskType = 1
	TaskTypeProject   TaskType = 1001
	TaskTypeBook      TaskType = 1003
)

// CreateTaskRequest はタスク作成リクエストのパラメータ。
type CreateTaskRequest struct {
	Name         string
	ParentID     *string
	Description  *string
	Status       *string
	Priority     *TaskPriority
	DueDate      *time.Time
	StartDate    *time.Time
	TimeEstimate *time.Duration // 分単位で渡し、API には ms として送信
	CustomItemID *TaskType
}

// UpdateTaskRequest はタスク更新リクエストのパラメータ。
// nil フィールドは更新しない。ClearFields に含まれるフィールドは値をクリアする。
type UpdateTaskRequest struct {
	Name         *string
	Description  *string
	Status       *string
	Priority     *TaskPriority
	DueDate      *time.Time
	StartDate    *time.Time
	TimeEstimate *time.Duration
	Parent       *string
	ClearFields  []string
}
