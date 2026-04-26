// internal/client/mapper.go
package client

import (
	"strconv"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
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
