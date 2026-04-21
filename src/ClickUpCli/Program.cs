using System.CommandLine;
using System.Text.Json;
using ClickUpClient;
using ClickUpClient.Http;
using ClickUpClient.Models;
using ClickUpCli;

var jsonOptions = new JsonSerializerOptions
{
    PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
    WriteIndented = true,
    Encoder = System.Text.Encodings.Web.JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
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

getTasksCommand.SetHandler(async (string[] lists, string[] statuses, string? dueAfterStr, string? dueBeforeStr, bool noSubtasks) =>
{
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
                    Environment.Exit(1);
                    return;
                }
                resolvedListIds.Add(id);
            }
        }

        DateTimeOffset? dueAfter = dueAfterStr is not null ? ParseIsoDateOrExit(dueAfterStr, "--due-after") : null;
        DateTimeOffset? dueBefore = dueBeforeStr is not null ? ParseIsoDateOrExit(dueBeforeStr, "--due-before") : null;

        using var httpClient = BuildHttpClient(config);
        IClickUpClient client = new ClickUpHttpClient(httpClient);

        var tasks = await client.GetTasksAsync(
            teamId: config.TeamId,
            // noSubtasks=true のとき includeSubtasks=false を渡す（意図的な反転）
            includeSubtasks: !noSubtasks,
            listIds: resolvedListIds,
            statuses: statuses.Length > 0 ? statuses : null,
            dueDateGt: dueAfter,
            dueDateLt: dueBefore);

        Console.WriteLine(JsonSerializer.Serialize(tasks, jsonOptions));
    }
    catch (FileNotFoundException ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
    catch (HttpRequestException ex)
    {
        var statusPart = ex.StatusCode.HasValue
            ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
            : "no status code";
        Console.Error.WriteLine($"HTTP Error ({statusPart}): {ex.Message}");
        Environment.Exit(1);
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
}, listOption, statusOption, dueAfterOption, dueBeforeOption, noSubtasksOption);

rootCommand.AddCommand(getTasksCommand);

// ── get-task ───────────────────────────────────────────────────────────────
var taskIdArgument = new Argument<string>("taskId", "ClickUp task ID");

var getTaskCommand = new Command("get-task", "Get a single task by ID as JSON");
getTaskCommand.AddArgument(taskIdArgument);

getTaskCommand.SetHandler(async (string taskId) =>
{
    try
    {
        var config = ConfigLoader.Load();

        using var httpClient = BuildHttpClient(config);
        IClickUpClient client = new ClickUpHttpClient(httpClient);

        var task = await client.GetTaskAsync(taskId);

        Console.WriteLine(JsonSerializer.Serialize(task, jsonOptions));
    }
    catch (FileNotFoundException ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
    catch (HttpRequestException ex)
    {
        var statusPart = ex.StatusCode.HasValue
            ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
            : "no status code";
        Console.Error.WriteLine($"HTTP Error ({statusPart}): {ex.Message}");
        Environment.Exit(1);
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
}, taskIdArgument);

rootCommand.AddCommand(getTaskCommand);

// ── create-task ────────────────────────────────────────────────────────────
var createNameArgument = new Argument<string>("name", "Task name");

var createListOption = new Option<string>(
    name: "--list",
    description: "List name defined in config.json.")
{ IsRequired = true };

var createDescriptionOption = new Option<string?>(
    name: "--description",
    description: "Task description.");

var createStatusOption = new Option<string?>(
    name: "--status",
    description: "Status name (e.g. \"to do\", \"in progress\").");

var createPriorityOption = new Option<string?>(
    name: "--priority",
    description: "Priority: urgent, high, normal, or low.");

var createDueDateOption = new Option<string?>(
    name: "--due-date",
    description: "Due date as ISO 8601. Timezone-less values are treated as JST (+09:00).");

var createStartDateOption = new Option<string?>(
    name: "--start-date",
    description: "Start date as ISO 8601. Timezone-less values are treated as JST (+09:00).");

var createTimeEstimateOption = new Option<int?>(
    name: "--time-estimate",
    description: "Time estimate in minutes.");

var createTaskCommand = new Command("create-task", "Create a new task and output it as JSON");
createTaskCommand.AddArgument(createNameArgument);
createTaskCommand.AddOption(createListOption);
createTaskCommand.AddOption(createDescriptionOption);
createTaskCommand.AddOption(createStatusOption);
createTaskCommand.AddOption(createPriorityOption);
createTaskCommand.AddOption(createDueDateOption);
createTaskCommand.AddOption(createStartDateOption);
createTaskCommand.AddOption(createTimeEstimateOption);

createTaskCommand.SetHandler(async (
    string name,
    string list,
    string? description,
    string? status,
    string? priorityStr,
    string? dueDateStr,
    string? startDateStr,
    int? timeEstimateMinutes) =>
{
    try
    {
        var config = ConfigLoader.Load();

        if (!config.Lists.TryGetValue(list, out var listId))
        {
            Console.Error.WriteLine(
                $"Error: Unknown list name '{list}'. Available: {string.Join(", ", config.Lists.Keys)}");
            Environment.Exit(1);
            return;
        }

        TaskPriority? priority = null;
        if (priorityStr is not null)
        {
            priority = priorityStr.ToLowerInvariant() switch
            {
                "urgent" => TaskPriority.Urgent,
                "high" => TaskPriority.High,
                "normal" => TaskPriority.Normal,
                "low" => TaskPriority.Low,
                _ => null,
            };
            if (priority is null)
            {
                Console.Error.WriteLine(
                    $"Error: Invalid priority '{priorityStr}'. Use urgent, high, normal, or low.");
                Environment.Exit(1);
                return;
            }
        }

        DateTimeOffset? dueDate = dueDateStr is not null ? ParseIsoDateOrExit(dueDateStr, "--due-date") : null;
        DateTimeOffset? startDate = startDateStr is not null ? ParseIsoDateOrExit(startDateStr, "--start-date") : null;

        using var httpClient = BuildHttpClient(config);
        IClickUpClient client = new ClickUpHttpClient(httpClient);

        var request = new CreateTaskRequest(name)
        {
            Description = description,
            Status = status,
            Priority = priority,
            DueDate = dueDate,
            StartDate = startDate,
            TimeEstimate = timeEstimateMinutes.HasValue
                ? TimeSpan.FromMinutes(timeEstimateMinutes.Value)
                : null,
        };

        var task = await client.CreateTaskAsync(listId, request);

        Console.WriteLine(JsonSerializer.Serialize(task, jsonOptions));
    }
    catch (FileNotFoundException ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
    catch (HttpRequestException ex)
    {
        var statusPart = ex.StatusCode.HasValue
            ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
            : "no status code";
        Console.Error.WriteLine($"HTTP Error ({statusPart}): {ex.Message}");
        Environment.Exit(1);
    }
    catch (Exception ex)
    {
        Console.Error.WriteLine($"Error: {ex.Message}");
        Environment.Exit(1);
    }
},
createNameArgument, createListOption, createDescriptionOption, createStatusOption,
createPriorityOption, createDueDateOption, createStartDateOption, createTimeEstimateOption);

rootCommand.AddCommand(createTaskCommand);

return await rootCommand.InvokeAsync(args);

static DateTimeOffset ParseIsoDateOrExit(string value, string optionName)
{
    try
    {
        return DateParsing.ParseIsoDate(value, optionName);
    }
    catch (ArgumentException ex)
    {
        Console.Error.WriteLine(
            $"Error: {ex.Message}");
        Environment.Exit(1);
        return default;
    }
}

static HttpClient BuildHttpClient(AppConfig config)
{
    var client = new HttpClient { BaseAddress = new Uri("https://api.clickup.com/api/") };
    client.DefaultRequestHeaders.Add("Authorization", config.ApiKey);
    return client;
}
