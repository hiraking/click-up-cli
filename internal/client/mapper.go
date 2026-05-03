// internal/client/mapper.go
package client

import (
	"strconv"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
)

// toSummary は rawTask を models.TaskSummary に変換する。
// Subtasks は空スライス（nil でなく []）で初期化する。
func toSummary(raw rawTask) models.TaskSummary {
	return models.TaskSummary{
		ID:          raw.ID,
		Name:        raw.Name,
		Status:      raw.Status.Status,
		Priority:    priorityStr(raw.Priority),
		ParentID:    raw.Parent,
		URL:         raw.URL,
		DueDate:     parseUnixMsPtr(raw.DueDate),
		StartDate:   parseUnixMsPtr(raw.StartDate),
		Description: raw.Description,
		ListID:      raw.List.ID,
		ListName:    raw.List.Name,
		CreatedAt:   parseUnixMs(raw.DateCreated),
		UpdatedAt:   parseUnixMs(raw.DateUpdated),
		Subtasks:    []models.TaskSummary{},
	}
}

// mapToRawCreateBody は models.CreateTaskRequest を rawCreateTaskBody に変換する。
func mapToRawCreateBody(req models.CreateTaskRequest) rawCreateTaskBody {
	body := rawCreateTaskBody{
		Name:        req.Name,
		Parent:      req.ParentID,
		Description: req.Description,
		Status:      req.Status,
	}

	if req.Priority != nil {
		p := int(*req.Priority)
		body.Priority = &p
	}

	if req.DueDate != nil {
		ms := req.DueDate.UnixMilli()
		body.DueDate = &ms
		hasTime := hasTimeComponent(*req.DueDate)
		body.DueDateTime = &hasTime
	}

	if req.StartDate != nil {
		ms := req.StartDate.UnixMilli()
		body.StartDate = &ms
		hasTime := hasTimeComponent(*req.StartDate)
		body.StartDateTime = &hasTime
	}

	if req.TimeEstimate != nil {
		ms := int(req.TimeEstimate.Milliseconds())
		body.TimeEstimate = &ms
	}

	if req.CustomItemID != nil {
		id := int(*req.CustomItemID)
		body.CustomItemID = &id
	}

	return body
}

func priorityStr(p *rawPriority) *string {
	if p == nil {
		return nil
	}
	s := p.Priority
	return &s
}

func parseUnixMsPtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	ms, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil
	}
	t := time.UnixMilli(ms).UTC()
	return &t
}

func parseUnixMs(s string) time.Time {
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// hasTimeComponent は time.Time の時刻部分（時・分・秒・ナノ秒）が
// すべてゼロかどうかを判定する。UTC に正規化してから判定する。
func hasTimeComponent(t time.Time) bool {
	u := t.UTC()
	return u.Hour() != 0 || u.Minute() != 0 || u.Second() != 0 || u.Nanosecond() != 0
}

// mapToRawUpdateBody は models.UpdateTaskRequest を PUT /v2/task/{taskId} ボディに変換する。
// map[string]interface{} を使うことでクリアフィールドへの明示的 null 送信を実現する。
func mapToRawUpdateBody(req models.UpdateTaskRequest) map[string]interface{} {
	body := make(map[string]interface{})

	if req.Name != nil {
		body["name"] = *req.Name
	}
	if req.Description != nil {
		body["description"] = *req.Description
	}
	if req.Status != nil {
		body["status"] = *req.Status
	}
	if req.Priority != nil {
		body["priority"] = int(*req.Priority)
	}
	if req.DueDate != nil {
		body["due_date"] = req.DueDate.UnixMilli()
		body["due_date_time"] = hasTimeComponent(*req.DueDate)
	}
	if req.StartDate != nil {
		body["start_date"] = req.StartDate.UnixMilli()
		body["start_date_time"] = hasTimeComponent(*req.StartDate)
	}
	if req.TimeEstimate != nil {
		body["time_estimate"] = int(req.TimeEstimate.Milliseconds())
	}
	if req.Parent != nil {
		body["parent"] = *req.Parent
	}

	for _, field := range req.ClearFields {
		switch field {
		case "description":
			body["description"] = " "
		case "status":
			body["status"] = nil
		case "priority":
			body["priority"] = nil
		case "due-date":
			body["due_date"] = nil
			delete(body, "due_date_time")
		case "start-date":
			body["start_date"] = nil
			delete(body, "start_date_time")
		case "time-estimate":
			body["time_estimate"] = nil
		}
	}

	return body
}
