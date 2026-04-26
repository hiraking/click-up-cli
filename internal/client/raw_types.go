// internal/client/raw_types.go
package client

// rawTask は GET /v2/team/{teamId}/task および GET /v2/task/{taskId} のレスポンス要素。
type rawTask struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description *string       `json:"description"`
	Status      rawTaskStatus `json:"status"`
	Parent      *string       `json:"parent"`
	Priority    *rawPriority  `json:"priority"`
	DueDate     *string       `json:"due_date"`
	StartDate   *string       `json:"start_date"`
	DateCreated string        `json:"date_created"`
	DateUpdated string        `json:"date_updated"`
	URL         string        `json:"url"`
	List        rawListRef    `json:"list"`
	TeamID      string        `json:"team_id"`
}

type rawTaskStatus struct {
	Status string `json:"status"`
}

type rawPriority struct {
	Priority string `json:"priority"`
}

type rawListRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type rawGetTasksResponse struct {
	Tasks []rawTask `json:"tasks"`
}
