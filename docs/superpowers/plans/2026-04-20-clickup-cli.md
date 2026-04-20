# ClickUp CLI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `IClickUpClient` の2メソッド（GetTasksAsync / GetTaskAsync）をCLIサブコマンドとして公開する Console App を追加する。

**Architecture:** `System.CommandLine` を使い `get-tasks` / `get-task` の2コマンドを定義する。`AppConfig` クラスが `config.json`（gitignore済み）から teamId・apiKey・listマッピングを読み込み、`ClickUpHttpClient` に渡す。出力は `System.Text.Json` でシリアライズした JSON を stdout に書く。

**Tech Stack:** .NET 10, System.CommandLine (beta4), System.Text.Json, プロジェクト参照: ClickUpClient

---

## File Map

| 操作 | ファイル | 責務 |
|---|---|---|
| 作成 | `src/ClickUpCli/ClickUpCli.csproj` | Console App プロジェクト定義 |
| 作成 | `src/ClickUpCli/Config.cs` | `AppConfig` モデル + `ConfigLoader` |
| 作成 | `src/ClickUpCli/Program.cs` | エントリポイント、コマンド定義・ハンドラ |
| 作成 | `src/ClickUpCli/config.sample.json` | ダミー値入りサンプル（コミット対象） |
| 作成 | `src/ClickUpCli/.gitignore` | `config.json` を除外 |
| 変更 | `ClickUpClient.slnx` | CLI プロジェクトをソリューションに追加 |

---

### Task 1: プロジェクトファイルと .gitignore を作成する

**Files:**
- Create: `src/ClickUpCli/ClickUpCli.csproj`
- Create: `src/ClickUpCli/.gitignore`

- [ ] **Step 1: `src/ClickUpCli/` ディレクトリを作成し、`.csproj` を配置する**

`src/ClickUpCli/ClickUpCli.csproj` を以下の内容で作成する:

```xml
<Project Sdk="Microsoft.NET.Sdk">

  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
    <AssemblyName>clickup</AssemblyName>
  </PropertyGroup>

  <ItemGroup>
    <PackageReference Include="System.CommandLine" Version="2.0.0-beta4.22272.1" />
  </ItemGroup>

  <ItemGroup>
    <ProjectReference Include="..\ClickUpClient\ClickUpClient.csproj" />
  </ItemGroup>

</Project>
```

- [ ] **Step 2: `src/ClickUpCli/.gitignore` を作成する**

```
config.json
```

- [ ] **Step 3: ソリューションファイルにプロジェクトを追加する**

`ClickUpClient.slnx` を開き、`/src/` フォルダの `<Folder>` 要素に以下を追記する:

```xml
<Solution>
  <Folder Name="/src/">
    <Project Path="src/ClickUpClient/ClickUpClient.csproj" />
    <Project Path="src/ClickUpCli/ClickUpCli.csproj" />
  </Folder>
  <Folder Name="/tests/">
    <Project Path="tests/ClickUpClient.Tests/ClickUpClient.Tests.csproj" />
  </Folder>
</Solution>
```

- [ ] **Step 4: ビルドが通ることを確認する（Program.cs 仮作成）**

まず `src/ClickUpCli/Program.cs` を空エントリポイントで作成:

```csharp
// placeholder
```

その後:

```
cd <repo-root>
dotnet build src/ClickUpCli/ClickUpCli.csproj
```

期待: `Build succeeded`（System.CommandLine の restore も完了）

- [ ] **Step 5: コミット**

```
git add src/ClickUpCli/ClickUpCli.csproj src/ClickUpCli/.gitignore ClickUpClient.slnx src/ClickUpCli/Program.cs
git commit -m "chore: add ClickUpCli project scaffold"
```

---

### Task 2: Config.cs — 設定モデルとローダーを実装する

**Files:**
- Create: `src/ClickUpCli/Config.cs`
- Create: `src/ClickUpCli/config.sample.json`

- [ ] **Step 1: `Config.cs` を作成する**

`src/ClickUpCli/Config.cs`:

```csharp
using System.Text.Json;
using System.Text.Json.Serialization;

namespace ClickUpCli;

internal sealed class AppConfig
{
    [JsonPropertyName("apiKey")]
    public string ApiKey { get; init; } = "";

    [JsonPropertyName("teamId")]
    public string TeamId { get; init; } = "";

    [JsonPropertyName("lists")]
    public Dictionary<string, string> Lists { get; init; } = new();
}

internal static class ConfigLoader
{
    public static AppConfig Load()
    {
        var path = Path.Combine(AppContext.BaseDirectory, "config.json");
        if (!File.Exists(path))
            throw new FileNotFoundException(
                $"config.json not found at '{path}'. Copy config.sample.json to config.json and fill in your values.");

        var json = File.ReadAllText(path);
        return JsonSerializer.Deserialize<AppConfig>(json)
            ?? throw new InvalidOperationException("config.json is empty or invalid JSON.");
    }
}
```

- [ ] **Step 2: `config.sample.json` を作成する**

`src/ClickUpCli/config.sample.json`:

```json
{
  "apiKey": "pk_YOUR_API_KEY_HERE",
  "teamId": "YOUR_TEAM_ID_HERE",
  "lists": {
    "my-list": "LIST_ID_HERE"
  }
}
```

- [ ] **Step 3: ビルドが通ることを確認する**

```
dotnet build src/ClickUpCli/ClickUpCli.csproj
```

期待: `Build succeeded`

- [ ] **Step 4: コミット**

```
git add src/ClickUpCli/Config.cs src/ClickUpCli/config.sample.json
git commit -m "feat: add AppConfig model and ConfigLoader"
```

---

### Task 3: Program.cs — 全コマンドを実装する

**Files:**
- Modify: `src/ClickUpCli/Program.cs`

- [ ] **Step 1: Program.cs を以下の完全な実装に置き換える**

`src/ClickUpCli/Program.cs`:

```csharp
using System.CommandLine;
using System.Text.Json;
using ClickUpClient.Http;
using ClickUpCli;

var jsonOptions = new JsonSerializerOptions
{
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
    WriteIndented = true,
};

var rootCommand = new RootCommand("ClickUp API CLI wrapper");

// ── get-tasks ──────────────────────────────────────────────────────────────
var listOption = new Option<string[]>(
    name: "--list",
    description: "List name(s) defined in config.json (repeatable). Omit for all lists.")
{
    AllowMultipleArgumentsPerToken = true,
};
listOption.Arity = ArgumentArity.ZeroOrMore;

var statusOption = new Option<string[]>(
    name: "--status",
    description: "Status name(s) to filter by, e.g. \"in progress\" (repeatable).")
{
    AllowMultipleArgumentsPerToken = true,
};
statusOption.Arity = ArgumentArity.ZeroOrMore;

var dueAfterOption = new Option<string?>(
    name: "--due-after",
    description: "ISO 8601 datetime. Return only tasks with due date after this value.");

var dueBeforeOption = new Option<string?>(
    name: "--due-before",
    description: "ISO 8601 datetime. Return only tasks with due date before this value.");

var noSubtasksOption = new Option<bool>(
    name: "--no-subtasks",
    description: "Exclude subtasks from results.");

var getTasksCommand = new Command("get-tasks", "Get tasks as a JSON tree");
getTasksCommand.AddOption(listOption);
getTasksCommand.AddOption(statusOption);
getTasksCommand.AddOption(dueAfterOption);
getTasksCommand.AddOption(dueBeforeOption);
getTasksCommand.AddOption(noSubtasksOption);

getTasksCommand.SetHandler(async (InvocationContext ctx) =>
{
    var lists = ctx.ParseResult.GetValueForOption(listOption) ?? [];
    var statuses = ctx.ParseResult.GetValueForOption(statusOption) ?? [];
    var dueAfterStr = ctx.ParseResult.GetValueForOption(dueAfterOption);
    var dueBeforeStr = ctx.ParseResult.GetValueForOption(dueBeforeOption);
    var noSubtasks = ctx.ParseResult.GetValueForOption(noSubtasksOption);

    try
    {
        var config = ConfigLoader.Load();

        List<string>? resolvedListIds = null;
        if (lists.Length > 0)
        {
            resolvedListIds = new List<string>(lists.Length);
            foreach (var name in lists)
            {
                if (!config.Lists.TryGetValue(name, out var id))
                {
                    Console.Error.WriteLine(
                        $"Error: Unknown list name '{name}'. Available: {string.Join(", ", config.Lists.Keys)}");
                    ctx.ExitCode = 1;
                    return;
                }
                resolvedListIds.Add(id);
            }
        }

        DateTimeOffset? dueAfter = dueAfterStr is not null ? DateTimeOffset.Parse(dueAfterStr) : null;
        DateTimeOffset? dueBefore = dueBeforeStr is not null ? DateTimeOffset.Parse(dueBeforeStr) : null;

        using var httpClient = BuildHttpClient(config);
        var client = new ClickUpHttpClient(httpClient);

        var tasks = await client.GetTasksAsync(
            teamId: config.TeamId,
            includeSubtasks: !noSubtasks,
            listIds: resolvedListIds,
            statuses: statuses.Length > 0 ? statuses : null,
            dueDateGt: dueAfter,
            dueDateLt: dueBefore,
            ct: ctx.GetCancellationToken());

        Console.WriteLine(JsonSerializer.Serialize(tasks, jsonOptions));
    }
    catch (FileNotFoundException ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        ctx.ExitCode = 1;
    }
    catch (HttpRequestException ex)
    {
        Console.Error.WriteLine($"HTTP Error: {ex.StatusCode} {ex.Message}");
        ctx.ExitCode = 1;
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        ctx.ExitCode = 1;
    }
});

rootCommand.AddCommand(getTasksCommand);

// ── get-task ───────────────────────────────────────────────────────────────
var taskIdArgument = new Argument<string>("taskId", "ClickUp task ID");

var getTaskCommand = new Command("get-task", "Get a single task by ID as JSON");
getTaskCommand.AddArgument(taskIdArgument);

getTaskCommand.SetHandler(async (InvocationContext ctx) =>
{
    var taskId = ctx.ParseResult.GetValueForArgument(taskIdArgument);

    try
    {
        var config = ConfigLoader.Load();

        using var httpClient = BuildHttpClient(config);
        var client = new ClickUpHttpClient(httpClient);

        var task = await client.GetTaskAsync(taskId, ctx.GetCancellationToken());

        Console.WriteLine(JsonSerializer.Serialize(task, jsonOptions));
    }
    catch (FileNotFoundException ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        ctx.ExitCode = 1;
    }
    catch (HttpRequestException ex)
    {
        Console.Error.WriteLine($"HTTP Error: {ex.StatusCode} {ex.Message}");
        ctx.ExitCode = 1;
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        ctx.ExitCode = 1;
    }
});

rootCommand.AddCommand(getTaskCommand);

return await rootCommand.InvokeAsync(args);

static HttpClient BuildHttpClient(AppConfig config)
{
    var client = new HttpClient { BaseAddress = new Uri("https://api.clickup.com/api/") };
    client.DefaultRequestHeaders.Add("Authorization", config.ApiKey);
    return client;
}
```

- [ ] **Step 2: ビルドが通ることを確認する**

```
dotnet build src/ClickUpCli/ClickUpCli.csproj
```

期待: `Build succeeded`

- [ ] **Step 3: コミット**

```
git add src/ClickUpCli/Program.cs
git commit -m "feat: implement get-tasks and get-task commands"
```

---

### Task 4: config.json を作成してスモークテストを実施する

**Files:**
- Create: `src/ClickUpCli/config.json` (gitignore対象・コミット不要)

> **注意:** このタスクは実際のClickUp APIキーが必要。CIでは実行しない。

- [ ] **Step 1: config.sample.json をコピーして実際の値を入力する**

```
cp src/ClickUpCli/config.sample.json src/ClickUpCli/config.json
```

`src/ClickUpCli/config.json` を開き、以下を実際の値に書き換える:
- `apiKey`: ClickUp の Personal API Token（Settings → Apps → API Token）
- `teamId`: ワークスペースのチームID（URL の `/w/{teamId}/` から確認）
- `lists`: 使いたいリスト名とそのリストID（ClickUp URL の `/v/{listId}` から確認）

- [ ] **Step 2: publish して実行可能バイナリを確認する**

```
dotnet publish src/ClickUpCli/ClickUpCli.csproj -c Release -o out/clickup
```

期待: `out/clickup/` に `clickup.exe`（Windows）が生成される

- [ ] **Step 3: config.json なしのエラーを確認する**

一時的に config.json をリネームしてエラーハンドリングを確認:

```
mv src/ClickUpCli/config.json src/ClickUpCli/config.json.bak
out/clickup/clickup get-tasks
mv src/ClickUpCli/config.json.bak src/ClickUpCli/config.json
```

期待: stderr に `Error: config.json not found at ...` が出力され、exit code 1

- [ ] **Step 4: get-tasks を実行して JSON が返ることを確認する**

```
out/clickup/clickup get-tasks
```

期待: stdout に `TaskSummary` の JSON 配列が出力される（空配列 `[]` も正常）

- [ ] **Step 5: get-tasks --list でフィルタが効くことを確認する**

`config.json` に定義したリスト名を使用（例: `"my-list"`）:

```
out/clickup/clickup get-tasks --list my-list
```

期待: stdout に絞り込まれた JSON が出力される

不明なリスト名を指定した場合のエラー確認:

```
out/clickup/clickup get-tasks --list nonexistent
```

期待: stderr に `Error: Unknown list name 'nonexistent'. Available: ...` が出力され exit code 1

- [ ] **Step 6: get-task を実行して単一タスクのJSONが返ることを確認する**

Step 4 の出力から任意のタスクIDをコピーして使用:

```
out/clickup/clickup get-task <taskId>
```

期待: stdout に単一 `TaskSummary` の JSON オブジェクトが出力される

- [ ] **Step 7: 最終コミット**

```
git add docs/superpowers/plans/2026-04-20-clickup-cli.md
git commit -m "docs: add ClickUp CLI implementation plan"
```
