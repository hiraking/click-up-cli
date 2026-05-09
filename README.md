# ClickUp CLI

A lightweight CLI wrapper for the ClickUp REST API v2, designed for use with AI agents and scripts. Fetches and creates tasks as structured JSON.

## Notes

- **Personal use only.** This tool was built for my own use and only implements the features I personally need. It is not a full-featured ClickUp client.
- **Personal Workspace assumed.** The tool is designed for ClickUp used with [Personal Workspace Layout](https://help.clickup.com/hc/en-us/articles/9867165253271-Activate-the-Personal-Workspace-layout) enabled. Features that are not relevant to solo use (e.g., assignees) are intentionally omitted.
- **JSON-only output.** All commands output JSON. This is intentional — the primary use case is consumption by AI agents and scripts, not human-readable display.

## Setup

### 1. Install

```bash
go install github.com/hiraking/click-up-cli/cmd/clickup@latest
```

The `clickup` binary is placed in `$GOPATH/bin` (typically `~/go/bin`).

### 2. Create a config file

Create `~/.clickup/config.json` (copy `config.sample.json` and fill in your values):

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "work":  "LIST_ID_1",
    "study": "LIST_ID_2"
  },
  "timezone": "UTC",
  "taskTypes": {
    "milestone": 1
  }
}
```

| Field | Description |
|---|---|
| `apiKey` | Personal API Token (Settings → Apps → API Token) |
| `teamId` | Workspace ID (found in the URL: `/w/{teamId}/`) |
| `lists` | Name-to-ID mapping used by the `--list` flag |
| `timezone` | IANA timezone name for offset-less datetime strings (e.g. `"Asia/Tokyo"`, `"UTC"`). Defaults to `"UTC"` if omitted. |
| `taskTypes` | Optional. Name-to-`custom_item_id` mapping used by `--task-type`. Keys are case-sensitive. To find your workspace's IDs, use the [Get Custom Task Types](https://clickup.com/api/clickupreference/operation/GetCustomItems/) endpoint. |

### 3. Override config

Use environment variables or the `--config` flag to override the config file.

**Environment variables**

| Variable | Field | Notes |
|---|---|---|
| `CLICKUP_API_KEY` | `apiKey` | Takes precedence over the config file |
| `CLICKUP_TEAM_ID` | `teamId` | Takes precedence over the config file |
| `CLICKUP_CONFIG` | config file path | Lower priority than `--config` |

> If both `CLICKUP_API_KEY` and `CLICKUP_TEAM_ID` are set, no config file is needed.

**`--config` flag** (available on all subcommands)

```bash
clickup --config /path/to/config.json get-tasks
```

Priority: `--config` flag > `CLICKUP_CONFIG` env var > `~/.clickup/config.json`

---

## Commands

### `get-tasks`

Fetches tasks as a tree.

```
clickup get-tasks [options]
```

| Option | Type | Description |
|---|---|---|
| `--list <name>` | string | List name from `config.json`. Repeatable. Defaults to all lists. |
| `--status <name>` | string | Filter by status. Repeatable. |
| `--due-after <ISO8601>` | string | Only tasks with a due date after this datetime. Timezone-less values use the `timezone` from config (default UTC). |
| `--due-before <ISO8601>` | string | Only tasks with a due date before this datetime. Timezone-less values use the `timezone` from config (default UTC). |
| `--no-subtasks` | flag | Exclude subtasks (default: included) |
| `--archived` | flag | Return archived tasks only (non-archived tasks are excluded). |
| `--query <text>` | string | Case-insensitive substring filter on task name and description (client-side, applied after all pages are fetched) |

**Output:** JSON array of root tasks. Subtasks are nested under each task's `subtasks` field.

```json
[
  {
    "id": "86exa7yq5",
    "name": "Write design doc",
    "status": "in progress",
    "priority": "high",
    "parentId": null,
    "subtasks": [
      {
        "id": "86exa8ab3",
        "name": "Draft outline",
        "status": "to do",
        "parentId": "86exa7yq5",
        "subtasks": []
      }
    ]
  }
]
```

> - Automatically paginates up to 10 pages (1,000 tasks). If the limit is reached, a warning is printed to stderr and the fetched results are returned.
> - `--due-after` / `--due-before` filtering is handled server-side by the ClickUp API.
> - **Timezone:** Datetime strings without an offset (e.g. `"2026-05-01"`, `"2026-05-01T09:00"`) are interpreted using the `timezone` setting from config (default: UTC). Explicit offsets (e.g. `"2026-05-01T00:00:00Z"`) are used as-is.

```bash
clickup get-tasks
clickup get-tasks --list work
clickup get-tasks --list work --list study
clickup get-tasks --list work --status active
clickup get-tasks --due-before 2026-04-21T23:59:59+09:00
clickup get-tasks --list work --no-subtasks
clickup get-tasks --archived
clickup get-tasks --query "design"
```

---

### `create-task`

Creates a new task.

```
clickup create-task <name> --list <name> [options]
```

| Argument/Option | Type | Description |
|---|---|---|
| `name` | string | Task name (required) |
| `--list <name>` | string | Destination list name (required) |
| `--description <text>` | string | Task description |
| `--parent <taskId>` | string | Parent task ID (creates a subtask) |
| `--status <name>` | string | Status name (e.g. `"to do"`, `"in progress"`) |
| `--priority <value>` | string | `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | Due date |
| `--start-date <ISO8601>` | string | Start date |
| `--time-estimate <minutes>` | int | Time estimate in minutes |
| `--task-type <name>` | string | Task type name as defined in `taskTypes` config (case-sensitive). |

**Output:** `TaskSummary` object of the created task (same shape as `get-task`).

```bash
clickup create-task "My task" --list work

clickup create-task "Write design doc" --list work \
  --description "Architecture design" \
  --parent "86exa7yq5" \
  --status "to do" \
  --priority high \
  --due-date "2026-05-01T18:00+09:00" \
  --start-date "2026-04-25T09:00" \
  --time-estimate 120

clickup create-task "Q2 Plan" --list work --task-type milestone
```

---

### `get-task`

Fetches a single task by ID.

```
clickup get-task <taskId>
```

```bash
clickup get-task 86exa7yq5
```

```json
{
  "id": "86exa7yq5",
  "name": "English study",
  "status": "active",
  "priority": null,
  "parentId": null,
  "url": "https://app.clickup.com/t/86exa7yq5",
  "dueDate": null,
  "description": "",
  "listId": "900523741862",
  "listName": "Study",
  "createdAt": "2026-04-19T15:09:41.393Z",
  "updatedAt": "2026-04-19T16:05:33.346Z",
  "subtasks": []
}
```

---

### `update-task`

Updates a task. Only specified fields are changed.

```
clickup update-task <taskId> [options]
```

| Option | Type | Description |
|---|---|---|
| `--name <text>` | string | New task name |
| `--description <text>` | string | New description |
| `--status <name>` | string | New status |
| `--priority <value>` | string | `urgent` / `high` / `normal` / `low` |
| `--due-date <ISO8601>` | string | New due date |
| `--start-date <ISO8601>` | string | New start date |
| `--time-estimate <minutes>` | int | New time estimate in minutes |
| `--parent <taskId>` | string | New parent task ID |
| `--clear <field>` | string | Clear a field (repeatable). Clearable fields: `description`, `status`, `priority`, `due-date`, `start-date`, `time-estimate` |
| `--archive` | flag | Archive the task |
| `--unarchive` | flag | Unarchive the task |

> `name` and `parent` cannot be cleared (`name` is required by the API; removing a subtask's parent is not supported).
> `--archive` and `--unarchive` are mutually exclusive and cannot be used together.

**Output:** `TaskSummary` object of the updated task (same shape as `get-task`).

```bash
clickup update-task 86exa7yq5 --name "New name"
clickup update-task 86exa7yq5 --status "in progress" --priority high
clickup update-task 86exa7yq5 --clear due-date
clickup update-task 86exa7yq5 --name "New name" --clear description
clickup update-task 86exa7yq5 --clear due-date --clear priority
clickup update-task 86exa7yq5 --archive
clickup update-task 86exa7yq5 --unarchive
clickup update-task 86exa7yq5 --archive --status done
```

---

### `delete-task`

Deletes a task by ID. No confirmation is required.

```
clickup delete-task <taskId>
```

**Output:**

```json
{
  "deleted": true,
  "taskId": "86exa7yq5"
}
```

```bash
clickup delete-task 86exa7yq5
```

---

### `time-report`

Aggregates time entries for a period and outputs a JSON report grouped by List → Task → Breakdown.

```
clickup time-report --start <ISO8601> --end <ISO8601> [options]
```

| Option | Type | Description |
|---|---|---|
| `--start <ISO8601>` | string | Start of the period (required, inclusive) |
| `--end <ISO8601>` | string | End of the period (required, exclusive) |
| `--output`, `-o <path>` | string | Output file path. Defaults to stdout. |
| `--rows` | bool | Include normalized rows. Defaults to `true` when `--output` is set, `false` otherwise. |

```bash
# Weekly report to stdout (no rows)
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00"

# Save to file (rows included by default)
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json

# Save to file, explicitly exclude rows
clickup time-report \
  --start "2026-04-27T00:00:00+09:00" \
  --end   "2026-05-04T00:00:00+09:00" \
  --output report.json \
  --rows=false
```

**Output** (without `--rows`):

```json
{
  "generatedAt": "2026-05-07T10:00:00Z",
  "period": {
    "start": "2026-04-27T00:00:00+09:00",
    "end": "2026-05-04T00:00:00+09:00",
    "timezone": "Asia/Tokyo"
  },
  "summary": {
    "totalDurationMin": 370,
    "listCount": 2,
    "topLevelTaskCount": 3,
    "breakdownTaskCount": 2
  },
  "hierarchy": [
    {
      "listId": "900523741862",
      "listName": "Work",
      "durationMin": 270,
      "tasks": [
        {
          "taskId": "86exa7yq5",
          "taskName": "Write design doc",
          "durationMin": 150,
          "breakdown": [
            { "taskId": "86exa7yq5", "taskName": "Write design doc", "durationMin": 90 },
            { "taskId": "86exa8ab3", "taskName": "Draft outline", "durationMin": 60 }
          ]
        },
        {
          "taskId": "86exb1cd2",
          "taskName": "Code review",
          "durationMin": 120
        }
      ]
    },
    {
      "listId": "900523741899",
      "listName": "Study",
      "durationMin": 100,
      "tasks": [
        {
          "taskId": "86exc3ef4",
          "taskName": "English study",
          "durationMin": 100
        }
      ]
    }
  ]
}
```

> - `breakdown` is omitted when all time was recorded directly on the top-level task (no subtask entries).
> - `rows` field is included only when `--rows` is enabled. Each row represents one clipped time entry with `originalStart/End` and `clippedStart/End`.

---

### `show-config`

Prints the resolved configuration as JSON. The API key is masked, showing only the last 4 characters.

```
clickup show-config
clickup --config /path/to/other-config.json show-config
```

```json
{
  "apiKey": "****a1b2",
  "teamId": "90371842016",
  "lists": {
    "work": "900523741862",
    "study": "900523741899"
  },
  "timezone": "Asia/Tokyo"
}
```

> `timezone` is omitted from the output if not set in config.

---

## Error handling

Errors are written to stderr and exit with code 1.

| Case | Example message |
|---|---|
| Config file not found | `config file not found: ...` |
| Unknown list name | `Error: Unknown list name 'foo'. Available: work, study` |
| Invalid date format | `Error: '--due-after' value '...' is not a valid ISO 8601 datetime.` |
| Invalid priority | `Error: Invalid priority 'foo'. Use urgent, high, normal, or low.` |
| `taskTypes` not configured | `Error: No task types configured. Add a "taskTypes" mapping to config.json.` |
| Unknown task type | `Error: Unknown task type 'foo'. Available: milestone, project` |
| API error | `HTTP Error (404): ...` |
| Rate limited (429) | Retries up to 3 times with a warning printed to stderr. Fails with `HTTP Error (429): rate limit exceeded after 3 retries`. |
