using System.CommandLine;
using System.CommandLine.Invocation;
using System.CommandLine.IO;
using System.Text.Json;
using ClickUpClient;
using ClickUpClient.Http;
using ClickUpClient.Models;

namespace ClickUpCli;

internal static class CliApplication
{
    public static RootCommand CreateRootCommand(
        Func<AppConfig>? loadConfig = null,
        Func<AppConfig, IClickUpClient>? createClient = null,
        JsonSerializerOptions? jsonOptions = null)
    {
        loadConfig ??= ConfigLoader.Load;
        createClient ??= CreateClient;
        jsonOptions ??= CreateJsonOptions();

        var rootCommand = new RootCommand("ClickUp API CLI wrapper");
        rootCommand.AddCommand(CreateGetTasksCommand(loadConfig, createClient, jsonOptions));
        rootCommand.AddCommand(CreateGetTaskCommand(loadConfig, createClient, jsonOptions));
        rootCommand.AddCommand(CreateCreateTaskCommand(loadConfig, createClient, jsonOptions));
        return rootCommand;
    }

    private static Command CreateGetTasksCommand(
        Func<AppConfig> loadConfig,
        Func<AppConfig, IClickUpClient> createClient,
        JsonSerializerOptions jsonOptions)
    {
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

        var command = new Command("get-tasks", "Get tasks as a JSON tree");
        command.AddOption(listOption);
        command.AddOption(statusOption);
        command.AddOption(dueAfterOption);
        command.AddOption(dueBeforeOption);
        command.AddOption(noSubtasksOption);

        command.SetHandler(async context =>
        {
            try
            {
                var lists = context.ParseResult.GetValueForOption(listOption) ?? [];
                var statuses = context.ParseResult.GetValueForOption(statusOption) ?? [];
                var dueAfterStr = context.ParseResult.GetValueForOption(dueAfterOption);
                var dueBeforeStr = context.ParseResult.GetValueForOption(dueBeforeOption);
                var noSubtasks = context.ParseResult.GetValueForOption(noSubtasksOption);

                var config = loadConfig();

                List<string>? resolvedListIds = null;
                if (lists.Length > 0)
                {
                    resolvedListIds = new List<string>(lists.Length);
                    foreach (var name in lists)
                    {
                        if (!config.Lists.TryGetValue(name, out var id))
                        {
                            Fail(context,
                                $"Error: Unknown list name '{name}'. Available: {string.Join(", ", config.Lists.Keys)}");
                            return;
                        }

                        resolvedListIds.Add(id);
                    }
                }

                DateTimeOffset? dueAfter = dueAfterStr is not null ? DateParsing.ParseIsoDate(dueAfterStr, "--due-after") : null;
                DateTimeOffset? dueBefore = dueBeforeStr is not null ? DateParsing.ParseIsoDate(dueBeforeStr, "--due-before") : null;

                var client = createClient(config);
                var tasks = await client.GetTasksAsync(
                    teamId: config.TeamId,
                    includeSubtasks: !noSubtasks,
                    listIds: resolvedListIds,
                    statuses: statuses.Length > 0 ? statuses : null,
                    dueDateGt: dueAfter,
                    dueDateLt: dueBefore);

                WriteLine(context.Console.Out, JsonSerializer.Serialize(tasks, jsonOptions));
            }
            catch (FileNotFoundException ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
            catch (HttpRequestException ex)
            {
                var statusPart = ex.StatusCode.HasValue
                    ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
                    : "no status code";
                Fail(context, $"HTTP Error ({statusPart}): {ex.Message}");
            }
            catch (Exception ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
        });

        return command;
    }

    private static Command CreateGetTaskCommand(
        Func<AppConfig> loadConfig,
        Func<AppConfig, IClickUpClient> createClient,
        JsonSerializerOptions jsonOptions)
    {
        var taskIdArgument = new Argument<string>("taskId", "ClickUp task ID");

        var command = new Command("get-task", "Get a single task by ID as JSON");
        command.AddArgument(taskIdArgument);

        command.SetHandler(async context =>
        {
            try
            {
                var taskId = context.ParseResult.GetValueForArgument(taskIdArgument);
                var config = loadConfig();
                var client = createClient(config);
                var task = await client.GetTaskAsync(taskId);

                WriteLine(context.Console.Out, JsonSerializer.Serialize(task, jsonOptions));
            }
            catch (FileNotFoundException ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
            catch (HttpRequestException ex)
            {
                var statusPart = ex.StatusCode.HasValue
                    ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
                    : "no status code";
                Fail(context, $"HTTP Error ({statusPart}): {ex.Message}");
            }
            catch (Exception ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
        });

        return command;
    }

    private static Command CreateCreateTaskCommand(
        Func<AppConfig> loadConfig,
        Func<AppConfig, IClickUpClient> createClient,
        JsonSerializerOptions jsonOptions)
    {
        var nameArgument = new Argument<string>("name", "Task name");

        var listOption = new Option<string>(
            name: "--list",
            description: "List name defined in config.json.")
        { IsRequired = true };

        var descriptionOption = new Option<string?>(
            name: "--description",
            description: "Task description.");

        var parentOption = new Option<string?>(
            name: "--parent",
            description: "Parent task ID. Provide to create the task as a subtask.");

        var statusOption = new Option<string?>(
            name: "--status",
            description: "Status name (e.g. \"to do\", \"in progress\").");

        var priorityOption = new Option<string?>(
            name: "--priority",
            description: "Priority: urgent, high, normal, or low.");

        var dueDateOption = new Option<string?>(
            name: "--due-date",
            description: "Due date as ISO 8601. Timezone-less values are treated as JST (+09:00).");

        var startDateOption = new Option<string?>(
            name: "--start-date",
            description: "Start date as ISO 8601. Timezone-less values are treated as JST (+09:00).");

        var timeEstimateOption = new Option<int?>(
            name: "--time-estimate",
            description: "Time estimate in minutes.");

        var command = new Command("create-task", "Create a new task and output it as JSON");
        command.AddArgument(nameArgument);
        command.AddOption(listOption);
        command.AddOption(descriptionOption);
        command.AddOption(parentOption);
        command.AddOption(statusOption);
        command.AddOption(priorityOption);
        command.AddOption(dueDateOption);
        command.AddOption(startDateOption);
        command.AddOption(timeEstimateOption);

        command.SetHandler(async context =>
        {
            try
            {
                var name = context.ParseResult.GetValueForArgument(nameArgument);
                var list = context.ParseResult.GetValueForOption(listOption)!;
                var description = context.ParseResult.GetValueForOption(descriptionOption);
                var parentId = context.ParseResult.GetValueForOption(parentOption);
                var status = context.ParseResult.GetValueForOption(statusOption);
                var priorityStr = context.ParseResult.GetValueForOption(priorityOption);
                var dueDateStr = context.ParseResult.GetValueForOption(dueDateOption);
                var startDateStr = context.ParseResult.GetValueForOption(startDateOption);
                var timeEstimateMinutes = context.ParseResult.GetValueForOption(timeEstimateOption);

                var config = loadConfig();

                if (!config.Lists.TryGetValue(list, out var listId))
                {
                    Fail(context,
                        $"Error: Unknown list name '{list}'. Available: {string.Join(", ", config.Lists.Keys)}");
                    return;
                }

                if (parentId is not null && string.IsNullOrWhiteSpace(parentId))
                {
                    Fail(context, "Error: '--parent' must not be empty or whitespace.");
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
                        Fail(context,
                            $"Error: Invalid priority '{priorityStr}'. Use urgent, high, normal, or low.");
                        return;
                    }
                }

                DateTimeOffset? dueDate = dueDateStr is not null ? DateParsing.ParseIsoDate(dueDateStr, "--due-date") : null;
                DateTimeOffset? startDate = startDateStr is not null ? DateParsing.ParseIsoDate(startDateStr, "--start-date") : null;

                var client = createClient(config);
                var request = new CreateTaskRequest(name)
                {
                    ParentId = parentId,
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
                WriteLine(context.Console.Out, JsonSerializer.Serialize(task, jsonOptions));
            }
            catch (FileNotFoundException ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
            catch (HttpRequestException ex)
            {
                var statusPart = ex.StatusCode.HasValue
                    ? $"{(int)ex.StatusCode.Value} {ex.StatusCode.Value}"
                    : "no status code";
                Fail(context, $"HTTP Error ({statusPart}): {ex.Message}");
            }
            catch (Exception ex)
            {
                Fail(context, $"Error: {ex.Message}");
            }
        });

        return command;
    }

    private static JsonSerializerOptions CreateJsonOptions() => new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
        WriteIndented = true,
        Encoder = System.Text.Encodings.Web.JavaScriptEncoder.UnsafeRelaxedJsonEscaping,
    };

    private static IClickUpClient CreateClient(AppConfig config) => new ClickUpHttpClient(CreateHttpClient(config));

    private static HttpClient CreateHttpClient(AppConfig config)
    {
        var client = new HttpClient { BaseAddress = new Uri("https://api.clickup.com/api/") };
        client.DefaultRequestHeaders.Add("Authorization", config.ApiKey);
        return client;
    }

    private static void Fail(InvocationContext context, string message)
    {
        WriteLine(context.Console.Error, message);
        context.ExitCode = 1;
    }

    private static void WriteLine(IStandardStreamWriter writer, string value) =>
        writer.Write(value + Environment.NewLine);
}
