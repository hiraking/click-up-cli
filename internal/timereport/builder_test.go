// internal/timereport/builder_test.go
package timereport_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/hiraking/click-up-cli/internal/timereport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var jst = time.FixedZone("JST", 9*60*60)

var (
	reportStart = time.Date(2026, 4, 27, 0, 0, 0, 0, jst)
	reportEnd   = time.Date(2026, 5, 4, 0, 0, 0, 0, jst)
)

func strPtr(s string) *string { return &s }

func makeEntry(id, taskID string, start, end time.Time, durMs int64) models.TimeEntry {
	return models.TimeEntry{
		ID:         id,
		TaskID:     taskID,
		TaskName:   "Task " + taskID,
		UserID:     "user1",
		UserName:   "Test User",
		Start:      start,
		End:        end,
		DurationMs: durMs,
	}
}

func makeTask(id string, parentID *string, listID, listName string) models.TaskSummary {
	return models.TaskSummary{
		ID:       id,
		Name:     "Task " + id,
		ParentID: parentID,
		ListID:   listID,
		ListName: listName,
		Subtasks: []models.TaskSummary{},
	}
}

func noFetch(_ context.Context, id string) (models.TaskSummary, error) {
	return models.TaskSummary{}, errors.New("unexpected fetch for " + id)
}

func mapFetch(tasks map[string]models.TaskSummary) timereport.TaskFetcher {
	return func(_ context.Context, id string) (models.TaskSummary, error) {
		t, ok := tasks[id]
		if !ok {
			return models.TaskSummary{}, errors.New("not found: " + id)
		}
		return t, nil
	}
}

func TestBuild_EmptyEntries(t *testing.T) {
	report, err := timereport.Build(context.Background(), nil, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Equal(t, 1, report.SchemaVersion)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
	assert.Equal(t, 0, report.Summary.ListCount)
}

func TestBuild_RunningTimerExcluded(t *testing.T) {
	running := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		-1, // negative = running timer
	)
	report, err := timereport.Build(context.Background(), []models.TimeEntry{running}, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
}

func TestBuild_EntryFullyOutsideRange(t *testing.T) {
	outside := makeEntry("e1", "t1",
		time.Date(2026, 5, 5, 9, 0, 0, 0, jst), // reportEnd より後
		time.Date(2026, 5, 5, 10, 0, 0, 0, jst),
		3600000,
	)
	report, err := timereport.Build(context.Background(), []models.TimeEntry{outside}, reportStart, reportEnd, noFetch)
	require.NoError(t, err)
	assert.Empty(t, report.Hierarchy)
	assert.Equal(t, int64(0), report.Summary.TotalDurationMin)
}

func TestBuild_EntryClippedAtStart(t *testing.T) {
	// reportStart の 2h 前から始まり、1h 後に終わる → clipped = 1h = 3600000ms
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 26, 22, 0, 0, 0, jst),
		time.Date(2026, 4, 27, 1, 0, 0, 0, jst),
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	assert.Equal(t, int64(60), report.Hierarchy[0].DurationMin)

	row := report.Rows[0]
	assert.Equal(t, int64(10800000), row.OriginalDurationMs)
	assert.Equal(t, int64(3600000), row.ClippedDurationMs)
	assert.Equal(t, reportStart, row.ClippedStart)
}

func TestBuild_EntryClippedAtEnd(t *testing.T) {
	// reportEnd の 1h 前から始まり、2h 後に終わる → clipped = 1h = 3600000ms
	entry := makeEntry("e1", "t1",
		time.Date(2026, 5, 3, 23, 0, 0, 0, jst),
		time.Date(2026, 5, 4, 2, 0, 0, 0, jst),
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	row := report.Rows[0]
	assert.Equal(t, reportEnd, row.ClippedEnd)
}

func TestBuild_DuplicateEntriesDeduped(t *testing.T) {
	e1 := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	// 同一エントリを2回渡す → 1回分だけ集計
	report, err := timereport.Build(context.Background(), []models.TimeEntry{e1, e1}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)
	assert.Equal(t, int64(60), report.Summary.TotalDurationMin)
	assert.Len(t, report.Rows, 1)
}

func TestBuild_TopLevelTask_BreakdownIsSelf(t *testing.T) {
	// top-level task に直接 time entry が記録された場合、breakdown はその task 自身
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	list := report.Hierarchy[0]
	assert.Equal(t, "list1", list.ListID)
	assert.Equal(t, "My List", list.ListName)
	assert.Equal(t, int64(60), list.DurationMin)

	require.Len(t, list.Tasks, 1)
	task := list.Tasks[0]
	assert.Equal(t, "t1", task.TaskID)
	assert.Equal(t, int64(60), task.DurationMin)

	// top-level task に直接記録 → breakdown は自分自身
	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "t1", task.Breakdown[0].TaskID)
	assert.Equal(t, int64(60), task.Breakdown[0].DurationMin)
}

func TestBuild_SubtaskResolvesToTopLevel(t *testing.T) {
	// sub1 (parent=top1) に記録 → top1 に集約、breakdown は sub1
	entry := makeEntry("e1", "sub1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 11, 0, 0, 0, jst),
		7200000,
	)
	tasks := map[string]models.TaskSummary{
		"sub1": makeTask("sub1", strPtr("top1"), "", ""),
		"top1": makeTask("top1", nil, "list1", "Work"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, "list1", report.Hierarchy[0].ListID)

	require.Len(t, report.Hierarchy[0].Tasks, 1)
	task := report.Hierarchy[0].Tasks[0]
	assert.Equal(t, "top1", task.TaskID)
	assert.Equal(t, int64(120), task.DurationMin)

	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "sub1", task.Breakdown[0].TaskID)
}

func TestBuild_MultiLevelSubtask_CollapsesToTopLevel(t *testing.T) {
	// A -> B -> C -> D（4段）、D に記録 → A が top-level、breakdown は D（B・C は出ない）
	entry := makeEntry("e1", "D",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	tasks := map[string]models.TaskSummary{
		"D": makeTask("D", strPtr("C"), "", ""),
		"C": makeTask("C", strPtr("B"), "", ""),
		"B": makeTask("B", strPtr("A"), "", ""),
		"A": makeTask("A", nil, "list1", "Project"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	require.Len(t, report.Hierarchy[0].Tasks, 1)
	task := report.Hierarchy[0].Tasks[0]
	assert.Equal(t, "A", task.TaskID)

	require.Len(t, task.Breakdown, 1)
	assert.Equal(t, "D", task.Breakdown[0].TaskID)

	assert.Equal(t, 1, report.Summary.TopLevelTaskCount)
	assert.Equal(t, 1, report.Summary.BreakdownTaskCount)
}

func TestBuild_MultipleEntries_Summary(t *testing.T) {
	entries := []models.TimeEntry{
		makeEntry("e1", "t1", time.Date(2026, 4, 28, 9, 0, 0, 0, jst), time.Date(2026, 4, 28, 10, 0, 0, 0, jst), 3600000),
		makeEntry("e2", "t2", time.Date(2026, 4, 29, 9, 0, 0, 0, jst), time.Date(2026, 4, 29, 11, 0, 0, 0, jst), 7200000),
	}
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "List 1"),
		"t2": makeTask("t2", nil, "list2", "List 2"),
	}
	report, err := timereport.Build(context.Background(), entries, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	assert.Equal(t, int64(180), report.Summary.TotalDurationMin)
	assert.Equal(t, 2, report.Summary.ListCount)
	assert.Equal(t, 2, report.Summary.TopLevelTaskCount)
	assert.Equal(t, 2, report.Summary.BreakdownTaskCount)
}

func TestBuild_TaskCachePreventsDoubleFetch(t *testing.T) {
	// 同じ subtask を持つ 2 つの time entry → sub1 と top1 はそれぞれ 1 回だけ fetch
	entries := []models.TimeEntry{
		makeEntry("e1", "sub1", time.Date(2026, 4, 28, 9, 0, 0, 0, jst), time.Date(2026, 4, 28, 10, 0, 0, 0, jst), 3600000),
		makeEntry("e2", "sub1", time.Date(2026, 4, 28, 11, 0, 0, 0, jst), time.Date(2026, 4, 28, 12, 0, 0, 0, jst), 3600000),
	}
	tasks := map[string]models.TaskSummary{
		"sub1": makeTask("sub1", strPtr("top1"), "", ""),
		"top1": makeTask("top1", nil, "list1", "Work"),
	}
	fetchCount := 0
	fetch := func(_ context.Context, id string) (models.TaskSummary, error) {
		fetchCount++
		t, ok := tasks[id]
		if !ok {
			return models.TaskSummary{}, errors.New("not found: " + id)
		}
		return t, nil
	}
	report, err := timereport.Build(context.Background(), entries, reportStart, reportEnd, fetch)
	require.NoError(t, err)

	// sub1 と top1 をそれぞれ 1 回ずつ = 計 2 回
	assert.Equal(t, 2, fetchCount)
	assert.Equal(t, int64(120), report.Summary.TotalDurationMin)
}

func TestBuild_ListFallback_UsesEntryList(t *testing.T) {
	// top-level task に ListID がない場合は entry.ListID にフォールバック
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 28, 9, 0, 0, 0, jst),
		time.Date(2026, 4, 28, 10, 0, 0, 0, jst),
		3600000,
	)
	entry.ListID = "fallback_list"
	entry.ListName = "Fallback"
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "", ""), // ListID が空
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Hierarchy, 1)
	assert.Equal(t, "fallback_list", report.Hierarchy[0].ListID)
	assert.Equal(t, "Fallback", report.Hierarchy[0].ListName)
}

func TestBuild_Rows_ContainClippedData(t *testing.T) {
	entry := makeEntry("e1", "t1",
		time.Date(2026, 4, 26, 22, 0, 0, 0, jst), // 2h before reportStart
		time.Date(2026, 4, 27, 1, 0, 0, 0, jst),  // 1h after reportStart
		10800000,
	)
	tasks := map[string]models.TaskSummary{
		"t1": makeTask("t1", nil, "list1", "My List"),
	}
	report, err := timereport.Build(context.Background(), []models.TimeEntry{entry}, reportStart, reportEnd, mapFetch(tasks))
	require.NoError(t, err)

	require.Len(t, report.Rows, 1)
	row := report.Rows[0]
	assert.Equal(t, "e1", row.TimeEntryID)
	assert.Equal(t, "user1", row.UserID)
	assert.Equal(t, "t1", row.TopLevelTaskID)
	assert.Equal(t, "t1", row.RecordedTaskID)
	assert.Equal(t, int64(10800000), row.OriginalDurationMs)
	assert.Equal(t, int64(3600000), row.ClippedDurationMs)
	assert.Equal(t, reportStart, row.ClippedStart)
	assert.Equal(t, entry.End, row.ClippedEnd)
}

func TestBuild_PeriodAndSchemaVersion(t *testing.T) {
	report, err := timereport.Build(context.Background(), nil, reportStart, reportEnd, noFetch)
	require.NoError(t, err)

	assert.Equal(t, 1, report.SchemaVersion)
	assert.Equal(t, reportStart, report.Period.Start)
	assert.Equal(t, reportEnd, report.Period.End)
	assert.Equal(t, "JST", report.Period.Timezone)
}
