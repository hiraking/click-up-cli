// internal/client/client.go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/hiraking/click-up-client/internal/models"
	"github.com/hiraking/click-up-client/internal/tree"
)

const baseURL = "https://api.clickup.com/api/"
const maxPages = 10

// ClickUpClient は ClickUp API の HTTP クライアントインターフェース。
type ClickUpClient interface {
	GetTasks(ctx context.Context, teamID string, opts GetTasksOptions) ([]models.TaskSummary, error)
	GetTask(ctx context.Context, taskID string) (models.TaskSummary, error)
	CreateTask(ctx context.Context, listID string, req models.CreateTaskRequest) (models.TaskSummary, error)
}

// GetTasksOptions は GetTasks のフィルタオプション。
type GetTasksOptions struct {
	IncludeSubtasks bool
	ListIDs         []string
	Statuses        []string
	DueDateGt       *time.Time
	DueDateLt       *time.Time
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return models.TaskSummary{}, err
	}
	httpReq.Header.Set("Authorization", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return models.TaskSummary{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		b, _ := io.ReadAll(httpResp.Body)
		return models.TaskSummary{}, fmt.Errorf("HTTP Error (%d): %s", httpResp.StatusCode, string(b))
	}

	var raw rawTask
	if err := json.NewDecoder(httpResp.Body).Decode(&raw); err != nil {
		return models.TaskSummary{}, fmt.Errorf("failed to decode response: %w", err)
	}
	return toSummary(raw), nil
}

func (c *httpClient) doGet(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP Error (%d): %s", resp.StatusCode, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(out)
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
