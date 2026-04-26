// internal/client/raw_create.go
package client

// rawCreateTaskBody は POST /v2/list/{listId}/task のリクエストボディ。
// omitempty で nil フィールドを省略する。
type rawCreateTaskBody struct {
	Name          string  `json:"name"`
	Parent        *string `json:"parent,omitempty"`
	Description   *string `json:"description,omitempty"`
	Status        *string `json:"status,omitempty"`
	Priority      *int    `json:"priority,omitempty"`
	DueDate       *int64  `json:"due_date,omitempty"`
	DueDateTime   *bool   `json:"due_date_time,omitempty"`
	StartDate     *int64  `json:"start_date,omitempty"`
	StartDateTime *bool   `json:"start_date_time,omitempty"`
	TimeEstimate  *int    `json:"time_estimate,omitempty"`
}
