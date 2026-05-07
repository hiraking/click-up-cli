# Design: task-type Config Mapping

**Date:** 2026-05-08  
**Topic:** Loosen `--task-type` validation by moving from hardcoded constants to `config.json` mapping

## Problem

`--task-type` currently validates against three hardcoded aliases (`milestone`, `project`, `book`) mapped to workspace-specific integer constants (`1`, `1001`, `1003`). This is not portable: every ClickUp workspace has different `custom_item_id` values for custom task types. Now that the repository is public, other users cannot use this flag without modifying source code.

## Approach

Add an optional `taskTypes` field to `config.json` that maps user-defined string aliases to `custom_item_id` integers. The CLI resolves `--task-type <name>` against this map at runtime.

## Design

### Config Schema

`taskTypes` is an optional `map[string]int` field in `config.json`:

```json
{
  "apiKey": "pk_...",
  "teamId": "...",
  "lists": { "work": "900523741862" },
  "timezone": "Asia/Tokyo",
  "taskTypes": {
    "milestone": 1,
    "project": 1001,
    "book": 1003
  }
}
```

- Keys: arbitrary strings (user-defined alias)
- Values: `custom_item_id` integer for the ClickUp workspace
- Entirely optional; the field may be absent or empty

### CLI Behavior

```bash
clickup create-task "Q2 Plan" --list work --task-type milestone
```

1. Load config
2. Look up `"milestone"` in `cfg.TaskTypes`
3. If found: use the integer value as `custom_item_id` in the API request
4. If not found: return error listing available keys
5. If `taskTypes` is absent/empty and `--task-type` is used: return error directing user to add a mapping

### Error Messages

| Situation | Error |
|---|---|
| `taskTypes` not in config | `Error: No task types configured. Add a "taskTypes" mapping to config.json.` |
| Key not found | `Error: Unknown task type 'foo'. Available: milestone, project, book` |

### Code Changes

#### `internal/config/config.go`
Add `TaskTypes map[string]int` field to `AppConfig`:
```go
TaskTypes map[string]int `mapstructure:"taskTypes"`
```
No validation needed (field is optional).

#### `internal/models/create_task.go`
- Remove `TaskType` type and named constants (`TaskTypeMilestone`, `TaskTypeProject`, `TaskTypeBook`)
- Change `CustomItemID *TaskType` → `CustomItemID *int`

#### `internal/client/mapper.go`
Update the `CustomItemID` mapping to use `*int` directly (previously cast from `*TaskType`):
```go
if req.CustomItemID != nil {
    body.CustomItemID = req.CustomItemID
}
```

#### `cmd/clickup/create_task.go`
- Remove `parseTaskType()` function
- Replace with config-based lookup in `RunE`:
```go
if cmd.Flags().Changed("task-type") {
    id, ok := cfg.TaskTypes[taskTypeStr]
    if !ok {
        if len(cfg.TaskTypes) == 0 {
            return fmt.Errorf("Error: No task types configured. Add a \"taskTypes\" mapping to config.json.")
        }
        keys := sortedKeys(cfg.TaskTypes)
        return fmt.Errorf("Error: Unknown task type '%s'. Available: %s", taskTypeStr, strings.Join(keys, ", "))
    }
    req.CustomItemID = &id
}
```
- Update flag description: `"Task type name as defined in the taskTypes config."`

#### `config.sample.json`
Add `taskTypes` example:
```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "my-list": "LIST_ID_HERE"
  },
  "timezone": "UTC",
  "taskTypes": {
    "milestone": 1
  }
}
```

#### `README.md`
- Update `--task-type` row in create-task options table
- Add `taskTypes` field to the config table in Setup section
- Update example error message in Error Handling section

### Testing

- **Remove:** `TestParseTaskType_ValidValues`, `TestParseTaskType_InvalidValue`
- **Update:** `TestMapToRawCreateBody_CustomItemID` (model type changes from `TaskType` to `int`)
- **Add:** Tests for config-based lookup in `create_task_test.go`:
  - Valid key → correct ID used
  - Unknown key → correct error with available keys listed
  - No `taskTypes` configured → correct error

## Out of Scope

- Adding a `get-task-types` command to query the ClickUp API (separate feature)
- Supporting `custom_item_id = 0` (reset to default "Task" type)
