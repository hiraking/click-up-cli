# CLI Hierarchy Redesign

## Problem

The current command structure is flat (`get-tasks`, `create-task`, `update-task`, etc.), which does not scale and lacks the discoverability of a resource-oriented hierarchy. The goal is to restructure commands into `clickup <resource> <action>` form.

## Approach

Adopt a **subdirectory-per-resource** layout where each resource group is a separate Go package that exports a `NewCmd(*string) *cobra.Command`. The `*string` parameter is a pointer to the global `configPath` variable declared in `main.go`, allowing all subcommands to resolve the config file path without global state leaking across packages.

Shared CLI utilities (JSON output, config loading, formatting helpers) move to a new `cmdutil` package consumed by all resource packages.

The final output remains a single compiled binary. No user-visible behavior changes except the command names.

## New Command Structure

| New command | Old command | Notes |
|---|---|---|
| `clickup task list` | `clickup get-tasks` | All flags unchanged |
| `clickup task get <id>` | `clickup get-task <id>` | Unchanged |
| `clickup task create <name>` | `clickup create-task <name>` | Unchanged |
| `clickup task update <id>` | `clickup update-task <id>` | Unchanged |
| `clickup task delete <id>` | `clickup delete-task <id>` | Unchanged |
| `clickup time report` | `clickup time-report` | All flags unchanged |
| `clickup config show` | `clickup show-config` | Unchanged |

No backward compatibility aliases. Migration is a clean cut.

## Directory Structure

```
cmd/clickup/
  main.go              ← package main: rootCmd, global --config flag, AddCommand wiring
  cmdutil/
    config.go          ← LoadConfig(configPath string), ResolveConfigPath(configPath string)
    output.go          ← PrintJSON(v any) error
    format.go          ← AvailableListNames(map[string]string), MaskAPIKey(string)
    format_test.go
  task/                ← package task
    cmd.go             ← NewCmd(configPath *string) *cobra.Command
    list.go            ← task list (was get-tasks)
    get.go             ← task get (was get-task)
    create.go          ← task create (was create-task)
    update.go          ← task update (was update-task)
    delete.go          ← task delete (was delete-task)
    create_test.go
    update_test.go
  time/                ← package timecmd (named timecmd to avoid stdlib "time" conflict)
    cmd.go             ← NewCmd(configPath *string) *cobra.Command
    report.go          ← time report (was time-report)
  config/              ← package configcmd (named configcmd to avoid ambiguity with internal/config)
    cmd.go             ← NewCmd(configPath *string) *cobra.Command
    show.go            ← config show (was show-config)
```

## Wiring Pattern

```go
// main.go
var configPath string

rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
rootCmd.AddCommand(task.NewCmd(&configPath))
rootCmd.AddCommand(timecmd.NewCmd(&configPath))
rootCmd.AddCommand(configcmd.NewCmd(&configPath))
```

Each resource package wires its subcommands internally:

```go
// task/cmd.go
func NewCmd(configPath *string) *cobra.Command {
    cmd := &cobra.Command{Use: "task", Short: "Manage tasks"}
    cmd.AddCommand(newListCmd(configPath))
    cmd.AddCommand(newGetCmd(configPath))
    cmd.AddCommand(newCreateCmd(configPath))
    cmd.AddCommand(newUpdateCmd(configPath))
    cmd.AddCommand(newDeleteCmd(configPath))
    return cmd
}
```

## File Migration Map

| Current file | Destination |
|---|---|
| `helpers.go` | split into `cmdutil/config.go`, `cmdutil/output.go`, `cmdutil/format.go` |
| `helpers_test.go` | `cmdutil/format_test.go` |
| `get_tasks.go` | `task/list.go` |
| `get_task.go` | `task/get.go` |
| `create_task.go` | `task/create.go` |
| `create_task_test.go` | `task/create_test.go` |
| `update_task.go` | `task/update.go` |
| `update_task_test.go` | `task/update_test.go` |
| `delete_task.go` | `task/delete.go` |
| `time_report.go` | `time/report.go` |
| `show_config.go` | `config/show.go` |
| `main.go` | `main.go` (restructured, no move) |

New files: `task/cmd.go`, `time/cmd.go`, `config/cmd.go`

## Package Naming Notes

- `cmd/clickup/time/` declares `package timecmd` to avoid shadowing the stdlib `time` package.
- `cmd/clickup/config/` declares `package configcmd` to avoid ambiguity with `internal/config` in import lists.
- In `main.go`, these are imported with their package aliases: `timecmd` and `configcmd`.

## Error Handling

No changes. All error propagation behavior remains identical.

## Testing

Existing tests in `create_task_test.go`, `update_task_test.go`, and `helpers_test.go` move to their new packages. Package names in test files are updated to match their new location. No new test cases are added as part of this refactor; the goal is structural only.

## README

The `## Commands` section of `README.md` is rewritten to reflect the new command names. All flags, arguments, output shapes, and examples are updated.

## Out of Scope

- New features or flags
- Changes to `internal/` packages
- Behavioral changes to any command
