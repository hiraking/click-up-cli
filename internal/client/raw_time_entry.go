// internal/client/raw_time_entry.go
package client

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
)

// rawTimeEntry は GET /v2/team/{teamId}/time_entries のレスポンス要素。
type rawTimeEntry struct {
	ID           string               `json:"id"`
	Task         *rawEntryTask        `json:"task"`
	User         rawEntryUser         `json:"user"`
	Start        json.Number          `json:"start"`    // Unix ms（API は integer または string を返す）
	End          json.Number          `json:"end"`      // Unix ms
	Duration     string               `json:"duration"` // ms 文字列。負値 = running timer
	TaskLocation rawTimeEntryLocation `json:"task_location"`
}

type rawEntryTask struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawEntryUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type rawTimeEntryLocation struct {
	ListID   json.Number `json:"list_id"` // API が integer または string を返す
	ListName string      `json:"list_name"`
}

type rawGetTimeEntriesResponse struct {
	Data []rawTimeEntry `json:"data"`
}

// toTimeEntry は rawTimeEntry を models.TimeEntry に変換する。
func toTimeEntry(raw rawTimeEntry) models.TimeEntry {
	startMs, _ := strconv.ParseInt(raw.Start.String(), 10, 64)
	endMs, _ := strconv.ParseInt(raw.End.String(), 10, 64)
	durMs, _ := strconv.ParseInt(raw.Duration, 10, 64)

	taskID := ""
	taskName := ""
	if raw.Task != nil {
		taskID = raw.Task.ID
		taskName = raw.Task.Name
	}

	return models.TimeEntry{
		ID:         raw.ID,
		TaskID:     taskID,
		TaskName:   taskName,
		UserID:     strconv.Itoa(raw.User.ID),
		UserName:   raw.User.Username,
		Start:      time.UnixMilli(startMs).UTC(),
		End:        time.UnixMilli(endMs).UTC(),
		DurationMs: durMs,
		ListID:     raw.TaskLocation.ListID.String(),
		ListName:   raw.TaskLocation.ListName,
	}
}
