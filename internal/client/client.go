// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hiraking/click-up-cli/internal/models"
	"github.com/hiraking/click-up-cli/internal/tree"
)

const baseURL = "https://api.clickup.com/api/"
const maxPages = 10
const maxRetries = 3
const defaultRetryWait = 60 * time.Second

// ErrNotFound is returned when a resource is not found (HTTP 404).
var ErrNotFound = errors.New("not found")

// ClickUpClient は ClickUp API の HTTP クライアントインターフェース。
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
	UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error)
	GetTimeEntries(ctx context.Context, teamID string, opts GetTimeEntriesOptions) ([]models.TimeEntry, error)
}

// GetTasksOptions は GetTasks のフィルタオプション。
type GetTasksOptions struct {
	IncludeSubtasks bool
	ListIDs         []string
	Statuses        []string
	DueDateGt       *time.Time
	DueDateLt       *time.Time
	Query           string
}

// GetTimeEntriesOptions は GetTimeEntries のオプション。
type GetTimeEntriesOptions struct {
	Start time.Time
	End   time.Time
}

type httpClient struct {
	base   string
	apiKey string
	http   *http.Client
}

// New は apiKey を使う ClickUpClient を返す。
func New(apiKey string) ClickUpClient {
	return &httpClient{
		base:   baseURL,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *httpClient) GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error) {
	var allRaw []rawTask
	for page := 0; page < maxPages; page++ {
		rawURL := c.buildGetTasksURL(teamID, opts, page)

		var resp rawGetTasksResponse
		if err := c.doGet(ctx, rawURL, &resp); err != nil {
			return nil, err
		}
		allRaw = append(allRaw, resp.Tasks...)
		if resp.LastPage {
			break
		}
		if page == maxPages-1 {
			fmt.Fprintf(os.Stderr, "warning: reached max page limit (%d pages, %d tasks). There may be more tasks. Use filters to narrow down results.\n", maxPages, len(allRaw))
		}
	}

	summaries := make([]models.TaskSummary, len(allRaw))
	for i, t := range allRaw {
		summaries[i] = toSummary(t)
	}
	summaries = filterByQuery(summaries, opts.Query)
	return tree.Build(summaries), nil
}

func (c *httpClient) GetTask(ctx context.Context, taskID string) (models.TaskSummary, error) {
	rawURL := c.base + "v2/task/" + taskID

	var raw rawTask
	if err := c.doGet(ctx, rawURL, &raw); err != nil {
		return models.TaskSummary{}, err
	}
	return toSummary(raw), nil
}

func (c *httpClient) CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error) {
	rawURL := c.base + "v2/list/" + listID + "/task"
	body := mapToRawCreateBody(req)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		return httpReq, nil
	})
	if err != nil {
		return models.TaskSummary{}, err
	}
	if status >= 400 {
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", status, string(respBody))
	}

	var raw rawTask
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}

func (c *httpClient) UpdateTask(ctx context.Context, taskID string, req models.UpdateTaskRequest) (models.TaskSummary, error) {
	rawURL := c.base + "v2/task/" + taskID
	body := mapToRawUpdateBody(req)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBody, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, rawURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Authorization", c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		return httpReq, nil
	})
	if err != nil {
		return models.TaskSummary{}, err
	}
	if status >= 400 {
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", status, string(respBody))
	}

	var raw rawTask
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}

func (c *httpClient) GetTimeEntries(ctx context.Context, teamID string, opts GetTimeEntriesOptions) ([]models.TimeEntry, error) {
	const fetchBuffer = 3 * time.Hour
	fetchStart := opts.Start.Add(-fetchBuffer)
	fetchEnd := opts.End.Add(fetchBuffer)

	params := url.Values{}
	params.Set("start_date", strconv.FormatInt(fetchStart.UnixMilli(), 10))
	params.Set("end_date", strconv.FormatInt(fetchEnd.UnixMilli(), 10))
	params.Set("include_location_names", "true")

	rawURL := c.base + "v2/team/" + teamID + "/time_entries?" + params.Encode()

	var resp rawGetTimeEntriesResponse
	if err := c.doGet(ctx, rawURL, &resp); err != nil {
		return nil, err
	}

	entries := make([]models.TimeEntry, len(resp.Data))
	for i, raw := range resp.Data {
		entries[i] = toTimeEntry(raw)
	}
	return entries, nil
}

// doWithRetry は HTTP リクエストを実行し、429 の場合はリトライする。
// 成功時はレスポンスボディと HTTP ステータスコードを返す。
func (c *httpClient) doWithRetry(ctx context.Context, makeReq func() (*http.Request, error)) ([]byte, int, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := makeReq()
		if err != nil {
			return nil, 0, err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, 0, err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := calcRetryWait(resp)
			resp.Body.Close()
			if attempt == maxRetries {
				return nil, http.StatusTooManyRequests, fmt.Errorf("HTTP Error (429): rate limit exceeded after %d retries", maxRetries)
			}
			fmt.Fprintf(os.Stderr, "warning: rate limited, retrying in %s (attempt %d/%d)...\n", wait.Round(time.Second), attempt+1, maxRetries)
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(wait):
			}
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, resp.StatusCode, readErr
		}
		return body, resp.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("unexpected exit from retry loop")
}

// calcRetryWait は 429 レスポンスから待機時間を算出する。
// X-RateLimit-Reset ヘッダー（Unix 秒）があればリセット時刻まで待機。なければ固定 60 秒。
func calcRetryWait(resp *http.Response) time.Duration {
	resetStr := resp.Header.Get("X-RateLimit-Reset")
	if resetStr != "" {
		resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
		if err == nil {
			wait := time.Until(time.Unix(resetUnix, 0))
			if wait < time.Second {
				wait = time.Second
			}
			return wait
		}
	}
	return defaultRetryWait
}

func (c *httpClient) doGet(ctx context.Context, rawURL string, out any) error {
	body, status, err := c.doWithRetry(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", c.apiKey)
		return req, nil
	})
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status >= 400 {
		return fmt.Errorf("HTTP Error (%d): %s", status, string(body))
	}
	return json.Unmarshal(body, out)
}

func filterByQuery(tasks []models.TaskSummary, query string) []models.TaskSummary {
	if query == "" {
		return tasks
	}
	q := strings.ToLower(query)
	result := make([]models.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Name), q) {
			result = append(result, t)
			continue
		}
		if t.Description != nil && strings.Contains(strings.ToLower(*t.Description), q) {
			result = append(result, t)
		}
	}
	return result
}

func (c *httpClient) buildGetTasksURL(teamID string, opts GetTasksOptions, page int) string {
	params := url.Values{}
	if opts.IncludeSubtasks {
		params.Set("subtasks", "true")
	} else {
		params.Set("subtasks", "false")
	}
	params.Set("page", strconv.Itoa(page))
	for _, id := range opts.ListIDs {
		params.Add("list_ids[]", id)
	}
	for _, s := range opts.Statuses {
		params.Add("statuses[]", s)
	}
	if opts.DueDateGt != nil {
		params.Set("due_date_gt", strconv.FormatInt(opts.DueDateGt.UnixMilli(), 10))
	}
	if opts.DueDateLt != nil {
		params.Set("due_date_lt", strconv.FormatInt(opts.DueDateLt.UnixMilli(), 10))
	}
	return c.base + "v2/team/" + teamID + "/task?" + params.Encode()
}
