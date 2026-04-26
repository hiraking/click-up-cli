// internal/client/mapper_test.go
package client

import (
	"testing"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSummary_FullFields(t *testing.T) {
	dueDate := "1567780450202"
	parentID := "parent123"
	desc := "task description"
	raw := rawTask{
		ID:          "task123",
		Name:        "Test Task",
		Description: &desc,
		Status:      rawTaskStatus{Status: "in progress"},
		Parent:      &parentID,
		Priority:    &rawPriority{Priority: "high"},
		DueDate:     &dueDate,
		DateCreated: "1567780450000",
		DateUpdated: "1567780451000",
		URL:         "https://app.clickup.com/t/task123",
		List:        rawListRef{ID: "list1", Name: "My List"},
	}

	s := toSummary(raw)

	assert.Equal(t, "task123", s.ID)
	assert.Equal(t, "Test Task", s.Name)
	assert.Equal(t, "in progress", s.Status)
	require.NotNil(t, s.Priority)
	assert.Equal(t, "high", *s.Priority)
	require.NotNil(t, s.ParentID)
	assert.Equal(t, "parent123", *s.ParentID)
	assert.Equal(t, "https://app.clickup.com/t/task123", s.URL)
	require.NotNil(t, s.DueDate)
	assert.Equal(t, int64(1567780450202), s.DueDate.UnixMilli())
	require.NotNil(t, s.Description)
	assert.Equal(t, "task description", *s.Description)
	assert.Equal(t, "list1", s.ListID)
	assert.Equal(t, "My List", s.ListName)
	assert.Equal(t, int64(1567780450000), s.CreatedAt.UnixMilli())
	assert.Equal(t, int64(1567780451000), s.UpdatedAt.UnixMilli())
	assert.NotNil(t, s.Subtasks)
	assert.Empty(t, s.Subtasks)
}

func TestToSummary_NullableFields(t *testing.T) {
	raw := rawTask{
		ID:          "task456",
		Name:        "Minimal Task",
		Status:      rawTaskStatus{Status: "to do"},
		DateCreated: "1567780450000",
		DateUpdated: "1567780450000",
		URL:         "https://app.clickup.com/t/task456",
		List:        rawListRef{ID: "list2", Name: "Work"},
	}

	s := toSummary(raw)

	assert.Nil(t, s.Priority)
	assert.Nil(t, s.ParentID)
	assert.Nil(t, s.DueDate)
	assert.Nil(t, s.Description)
}

func TestMapToRawCreateBody_DueDateTimeFlag(t *testing.T) {
	// 時刻あり → due_date_time = true
	withTime := time.Date(2026, 5, 1, 18, 0, 0, 0, time.UTC)
	dur := 2 * time.Hour
	pri := models.PriorityHigh
	req := models.CreateTaskRequest{
		Name:         "Test",
		DueDate:      &withTime,
		TimeEstimate: &dur,
		Priority:     &pri,
	}

	body := mapToRawCreateBody(req)

	assert.Equal(t, withTime.UnixMilli(), *body.DueDate)
	assert.True(t, *body.DueDateTime)
	assert.Equal(t, int(dur.Milliseconds()), *body.TimeEstimate)
	assert.Equal(t, 2, *body.Priority) // PriorityHigh = 2
}

func TestMapToRawCreateBody_MidnightHasNoTime(t *testing.T) {
	// 00:00:00 → due_date_time = false
	midnight := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	req := models.CreateTaskRequest{
		Name:    "Test",
		DueDate: &midnight,
	}

	body := mapToRawCreateBody(req)

	assert.False(t, *body.DueDateTime)
}

func TestMapToRawCreateBody_NilFields(t *testing.T) {
	req := models.CreateTaskRequest{Name: "Minimal"}

	body := mapToRawCreateBody(req)

	assert.Equal(t, "Minimal", body.Name)
	assert.Nil(t, body.Parent)
	assert.Nil(t, body.Description)
	assert.Nil(t, body.Status)
	assert.Nil(t, body.Priority)
	assert.Nil(t, body.DueDate)
	assert.Nil(t, body.StartDate)
	assert.Nil(t, body.TimeEstimate)
}
