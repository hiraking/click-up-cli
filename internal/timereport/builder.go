// internal/timereport/builder.go
package timereport

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hiraking/click-up-cli/internal/client"
	"github.com/hiraking/click-up-cli/internal/models"
)

// TaskFetcher はタスクIDからタスクメタデータを取得する関数型。
// client.ClickUpClient.GetTask をそのまま渡せるシグネチャ。
type TaskFetcher func(ctx context.Context, taskID string) (models.TaskSummary, error)

// Build は time entries を集計して TimeReport を返す。
// start は集計範囲の開始（含む）、end は終了（含まない）の半開区間。
func Build(
	ctx context.Context,
	entries []models.TimeEntry,
	start, end time.Time,
	fetch TaskFetcher,
) (models.TimeReport, error) {
	// 1. Dedup by entry ID
	seen := make(map[string]bool)
	var unique []models.TimeEntry
	for _, e := range entries {
		if !seen[e.ID] {
			seen[e.ID] = true
			unique = append(unique, e)
		}
	}

	// 2. Filter running timers (DurationMs < 0)
	var valid []models.TimeEntry
	for _, e := range unique {
		if e.DurationMs >= 0 {
			valid = append(valid, e)
		}
	}

	// 3. Clip each entry to [start, end) and discard zero-duration results
	type clippedEntry struct {
		entry  models.TimeEntry
		cStart time.Time
		cEnd   time.Time
		cDurMs int64
	}
	var clipped []clippedEntry
	for _, e := range valid {
		cs := maxTime(e.Start, start)
		ce := minTime(e.End, end)
		dur := ce.Sub(cs).Milliseconds()
		if dur <= 0 {
			continue
		}
		clipped = append(clipped, clippedEntry{e, cs, ce, dur})
	}

	// Task metadata in-memory cache
	taskCache := make(map[string]models.TaskSummary)

	// fetchCached fetches a task by ID, using the cache.
	fetchCached := func(taskID string) (models.TaskSummary, error) {
		if t, ok := taskCache[taskID]; ok {
			return t, nil
		}
		t, err := fetch(ctx, taskID)
		if err != nil {
			return models.TaskSummary{}, err
		}
		taskCache[taskID] = t
		return t, nil
	}

	// resolveTopLevel walks the parent chain to find the root task.
	var resolveTopLevel func(taskID string) (models.TaskSummary, error)
	resolveTopLevel = func(taskID string) (models.TaskSummary, error) {
		t, err := fetchCached(taskID)
		if err != nil {
			return models.TaskSummary{}, err
		}
		if t.ParentID == nil {
			return t, nil
		}
		parent, err := resolveTopLevel(*t.ParentID)
		if err != nil {
			if errors.Is(err, client.ErrNotFound) {
				// Parent is gone; treat current task as root
				return t, nil
			}
			return models.TaskSummary{}, err
		}
		return parent, nil
	}

	// Hierarchy accumulation maps (insertion-order tracking via slices)
	listOrder := []string{}
	listNames := make(map[string]string)
	listDur := make(map[string]int64)

	type listTask struct{ listID, taskID string }

	taskOrder := make(map[string][]string) // listID -> []topTaskIDs
	taskNames := make(map[string]string)
	taskDur := make(map[listTask]int64) // (listID, topTaskID) -> duration
	taskSeen := make(map[listTask]bool) // (listID, topTaskID) -> seen

	type bk struct{ top, rec string }
	bdOrder := make(map[string][]string) // topTaskID -> []recTaskIDs (in order)
	bdNames := make(map[string]string)   // recTaskID -> name
	bdDur := make(map[bk]int64)
	bdSeen := make(map[string]bool) // "topTaskID|recTaskID"

	var rows []models.TimeReportRow
	var totalDurMs int64

	for _, c := range clipped {
		e := c.entry
		totalDurMs += c.cDurMs

		// Resolve recorded task
		var recTask models.TaskSummary
		if e.TaskID == "" {
			recTask = models.TaskSummary{ID: e.ID, Name: e.TaskName, ListID: e.ListID, ListName: e.ListName}
		} else {
			t, err := fetchCached(e.TaskID)
			if err != nil {
				if errors.Is(err, client.ErrNotFound) {
					t = models.TaskSummary{ID: e.TaskID, Name: e.TaskName, ListID: e.ListID, ListName: e.ListName}
					taskCache[e.TaskID] = t
				} else {
					return models.TimeReport{}, fmt.Errorf("fetching task %s: %w", e.TaskID, err)
				}
			}
			recTask = t
		}

		// Resolve top-level task
		var topTask models.TaskSummary
		if e.TaskID == "" {
			topTask = recTask
		} else {
			var err error
			topTask, err = resolveTopLevel(e.TaskID)
			if err != nil {
				return models.TimeReport{}, fmt.Errorf("resolving top-level task for %s: %w", e.TaskID, err)
			}
		}

		// List resolution: top-level task > entry > "unknown"
		listID := topTask.ListID
		listName := topTask.ListName
		if listID == "" {
			listID = e.ListID
			listName = e.ListName
		}
		if listID == "" {
			listID = "unknown"
			listName = "unknown"
		}

		topTaskID := topTask.ID
		topTaskName := topTask.Name
		recTaskID := recTask.ID
		recTaskName := recTask.Name

		// Update list maps
		if _, ok := listNames[listID]; !ok {
			listOrder = append(listOrder, listID)
			listNames[listID] = listName
		}
		listDur[listID] += c.cDurMs

		// Update task maps
		lt := listTask{listID, topTaskID}
		if !taskSeen[lt] {
			taskSeen[lt] = true
			taskOrder[listID] = append(taskOrder[listID], topTaskID)
			taskNames[topTaskID] = topTaskName
		}
		taskDur[lt] += c.cDurMs

		// Update breakdown maps
		bkStr := topTaskID + "|" + recTaskID
		if !bdSeen[bkStr] {
			bdSeen[bkStr] = true
			bdOrder[topTaskID] = append(bdOrder[topTaskID], recTaskID)
			bdNames[recTaskID] = recTaskName
		}
		bdDur[bk{topTaskID, recTaskID}] += c.cDurMs

		rows = append(rows, models.TimeReportRow{
			TimeEntryID:        e.ID,
			ListID:             listID,
			ListName:           listName,
			TopLevelTaskID:     topTaskID,
			TopLevelTaskName:   topTaskName,
			RecordedTaskID:     recTaskID,
			RecordedTaskName:   recTaskName,
			OriginalStart:      e.Start,
			OriginalEnd:        e.End,
			OriginalDurationMs: e.DurationMs,
			ClippedStart:       c.cStart,
			ClippedEnd:         c.cEnd,
			ClippedDurationMs:  c.cDurMs,
		})
	}

	// Build hierarchy slices from maps
	var lists []models.TimeReportList
	var topLevelCount, breakdownCount int

	for _, lid := range listOrder {
		var tasks []models.TimeReportTask
		for _, tid := range taskOrder[lid] {
			// Only populate breakdown when at least one recorded task differs from the top-level task.
			hasSubtask := false
			for _, rid := range bdOrder[tid] {
				if rid != tid {
					hasSubtask = true
					break
				}
			}
			var breakdown []models.TimeReportBreakdown
			if hasSubtask {
				for _, rid := range bdOrder[tid] {
					breakdown = append(breakdown, models.TimeReportBreakdown{
						TaskID:      rid,
						TaskName:    bdNames[rid],
						DurationMin: bdDur[bk{tid, rid}] / 60000,
					})
					breakdownCount++
				}
			}
			tasks = append(tasks, models.TimeReportTask{
				TaskID:      tid,
				TaskName:    taskNames[tid],
				DurationMin: taskDur[listTask{lid, tid}] / 60000,
				Breakdown:   breakdown,
			})
			topLevelCount++
		}
		lists = append(lists, models.TimeReportList{
			ListID:      lid,
			ListName:    listNames[lid],
			DurationMin: listDur[lid] / 60000,
			Tasks:       tasks,
		})
	}

	if lists == nil {
		lists = []models.TimeReportList{}
	}

	return models.TimeReport{
		SchemaVersion: 1,
		GeneratedAt:   time.Now(),
		Period: models.TimePeriod{
			Start:    start,
			End:      end,
			Timezone: start.Location().String(),
		},
		Summary: models.TimeReportSummary{
			TotalDurationMin:   totalDurMs / 60000,
			ListCount:          len(lists),
			TopLevelTaskCount:  topLevelCount,
			BreakdownTaskCount: breakdownCount,
		},
		Hierarchy: lists,
		Rows:      rows,
	}, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
