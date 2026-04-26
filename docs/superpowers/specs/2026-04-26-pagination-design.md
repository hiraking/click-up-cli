# Pagination Support for `get-tasks`

## Problem

`GetTasks` fetches only page=0 (up to 100 tasks). When a workspace has more than 100 tasks matching the filter, the remainder is silently dropped.

## Approach

Auto-fetch all pages inside `client.GetTasks`. Callers (CLI) are unchanged. An internal safety cap of 10 pages (1,000 tasks) prevents runaway requests. If the cap is reached and the API indicates more pages remain, a warning is printed to stderr.

## Changes

### `internal/client/raw_types.go`

Add `LastPage bool` to `rawGetTasksResponse`:

```go
type rawGetTasksResponse struct {
    Tasks    []rawTask `json:"tasks"`
    LastPage bool      `json:"last_page"`
}
```

### `internal/client/client.go`

- Add internal constant: `const maxPages = 10`
- Remove `Page` field from `GetTasksOptions` (no longer meaningful to callers)
- Rewrite `GetTasks` to loop pages:

```
allRaw := []rawTask{}
for page := 0; page < maxPages; page++ {
    resp = fetch(page)
    allRaw = append(allRaw, resp.Tasks...)
    if resp.LastPage {
        break
    }
    if page == maxPages-1 {
        fmt.Fprintf(os.Stderr, "warning: reached max page limit (%d pages, %d tasks). There may be more tasks. Use filters to narrow down results.\n", maxPages, len(allRaw))
    }
}
summaries = toSummary(allRaw)
return tree.Build(summaries)
```

**Termination condition:** `last_page == true` from the API response (authoritative). Does not rely on guessing from task count.

### `cmd/clickup/get_tasks.go`

No changes needed. `GetTasksOptions.Page` removal is the only interface change, and this field was never set by the CLI.

## Behavior

| Scenario | Result |
|---|---|
| Tasks ≤ 100 | Single request, `last_page=true`, returns all |
| 100 < Tasks ≤ 1000 | Multiple requests, `last_page=true` on last page, returns all |
| Tasks > 1000 | Fetches 1000 tasks, prints warning to stderr, returns what was fetched |

## Out of Scope

- `--max-pages` flag (internal limit only)
- Streaming / incremental output
- Caching between pages
