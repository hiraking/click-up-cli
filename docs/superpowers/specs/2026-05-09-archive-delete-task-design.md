# Archive and Delete Task — Design Spec

**Date:** 2026-05-09

## Problem

The CLI currently has no way to archive or delete tasks. These are common task lifecycle operations.

## Decisions

| Question | Decision |
|---|---|
| Archive interface | `--archive` / `--unarchive` flags added to `update-task` |
| Delete interface | New `delete-task` command |
| Delete confirmation | None — immediate deletion (agent/script-friendly) |
| Delete output | `{"deleted": true, "taskId": "<id>"}` |

## Design

### 1. `update-task` — New Flags

Add two mutually exclusive boolean flags:

- `--archive`: sends `archived: true` in the request body
- `--unarchive`: sends `archived: false` in the request body

These can be combined with other `update-task` flags (e.g., `--archive --status done`). Specifying both `--archive` and `--unarchive` in the same invocation is an error.

The output is the existing `TaskSummary` shape — no change.

```bash
clickup update-task 86exa7yq5 --archive
clickup update-task 86exa7yq5 --unarchive
clickup update-task 86exa7yq5 --archive --status done
```

### 2. `delete-task` — New Command

```
clickup delete-task <taskId>
```

- Calls `DELETE /v2/task/{task_id}`
- No confirmation prompt
- On success, outputs:

```json
{"deleted": true, "taskId": "86exa7yq5"}
```

- On error, writes to stderr and exits with code 1 (same behavior as all other commands)

```bash
clickup delete-task 86exa7yq5
```

## Implementation Plan

| File | Change |
|---|---|
| `internal/client/client.go` | Add `DeleteTask(taskId string) error` method; add `Archived *bool` to the update request struct |
| `cmd/clickup/update_task.go` | Add `--archive` / `--unarchive` flags; validate mutual exclusion; pass `Archived` to client |
| `cmd/clickup/delete_task.go` | New file — `delete-task` command |
| `README.md` | Document `--archive`/`--unarchive` in `update-task` section; add `delete-task` section |

## API Reference

- **Archive/Unarchive:** `PUT /v2/task/{task_id}` with `{ "archived": true|false }`
- **Delete:** `DELETE /v2/task/{task_id}` — returns empty body on success
