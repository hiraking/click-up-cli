// internal/models/time_report.go
package models

import "time"

// TimeEntry は ClickUp time entry の処理済み DTO。
type TimeEntry struct {
	ID         string
	TaskID     string
	TaskName   string
	UserID     string
	UserName   string
	Start      time.Time
	End        time.Time
	DurationMs int64 // 元の duration（ms）。負値 = running timer
	// task_location から取得したリスト情報（フォールバック用）
	ListID   string
	ListName string
}

// TimeReport は time-report コマンドの出力 DTO。
type TimeReport struct {
	SchemaVersion int               `json:"schemaVersion"`
	GeneratedAt   time.Time         `json:"generatedAt"`
	Period        TimePeriod        `json:"period"`
	Summary       TimeReportSummary `json:"summary"`
	Hierarchy     []TimeReportList  `json:"hierarchy"`
	Rows          []TimeReportRow   `json:"rows,omitempty"`
}

// TimePeriod は集計期間のメタデータ。
type TimePeriod struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Timezone string    `json:"timezone"`
}

// TimeReportSummary は集計サマリー。
type TimeReportSummary struct {
	TotalDurationMin   int64 `json:"totalDurationMin"` // 分単位（切り捨て）
	ListCount          int   `json:"listCount"`
	TopLevelTaskCount  int   `json:"topLevelTaskCount"`
	BreakdownTaskCount int   `json:"breakdownTaskCount"`
}

// TimeReportList は List 単位の集計。
type TimeReportList struct {
	ListID      string           `json:"listId"`
	ListName    string           `json:"listName"`
	DurationMin int64            `json:"durationMin"` // 分単位（切り捨て）
	Tasks       []TimeReportTask `json:"tasks"`
}

// TimeReportTask は top-level task 単位の集計。
type TimeReportTask struct {
	TaskID      string                `json:"taskId"`
	TaskName    string                `json:"taskName"`
	DurationMin int64                 `json:"durationMin"` // 分単位（切り捨て）
	Breakdown   []TimeReportBreakdown `json:"breakdown,omitempty"`
}

// TimeReportBreakdown は recorded task 単位の内訳。
// Breakdown は常にフラット（1段）: recorded task = 実際に time entry が記録されたタスク。
// タスク階層が何段あっても top-level task の直下に recorded task が並ぶ。
type TimeReportBreakdown struct {
	TaskID      string `json:"taskId"`
	TaskName    string `json:"taskName"`
	DurationMin int64  `json:"durationMin"` // 分単位（切り捨て）
}

// TimeReportRow は後続分析用の正規化済み明細行。
type TimeReportRow struct {
	TimeEntryID        string    `json:"timeEntryId"`
	ListID             string    `json:"listId"`
	ListName           string    `json:"listName"`
	TopLevelTaskID     string    `json:"topLevelTaskId"`
	TopLevelTaskName   string    `json:"topLevelTaskName"`
	RecordedTaskID     string    `json:"recordedTaskId"`
	RecordedTaskName   string    `json:"recordedTaskName"`
	OriginalStart      time.Time `json:"originalStart"`
	OriginalEnd        time.Time `json:"originalEnd"`
	OriginalDurationMs int64     `json:"originalDurationMs"`
	ClippedStart       time.Time `json:"clippedStart"`
	ClippedEnd         time.Time `json:"clippedEnd"`
	ClippedDurationMs  int64     `json:"clippedDurationMs"`
}
