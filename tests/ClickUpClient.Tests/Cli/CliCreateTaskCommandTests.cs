using System.CommandLine;
using System.CommandLine.IO;
using ClickUpClient.Models;
using ClickUpCli;

namespace ClickUpClient.Tests.Cli;

public class CliCreateTaskCommandTests
{
    [Fact]
    public async Task CreateTaskCommand_BindsParentOptionIntoCreateTaskRequest()
    {
        var client = new RecordingClickUpClient();
        var console = new TestConsole();
        var command = CliApplication.CreateRootCommand(
            () => CreateConfig(),
            _ => client);

        var exitCode = await command.InvokeAsync(
            ["create-task", "Child Task", "--list", "inbox", "--parent", "parent-123"],
            console);

        Assert.Equal(0, exitCode);
        Assert.Equal("list-42", client.LastCreateListId);
        Assert.NotNull(client.LastCreateRequest);
        Assert.Equal("Child Task", client.LastCreateRequest!.Name);
        Assert.Equal("parent-123", client.LastCreateRequest.ParentId);
    }

    [Fact]
    public async Task CreateTaskCommand_RejectsWhitespaceOnlyParentOption()
    {
        var client = new RecordingClickUpClient();
        var console = new TestConsole();
        var command = CliApplication.CreateRootCommand(
            () => CreateConfig(),
            _ => client);

        var exitCode = await command.InvokeAsync(
            ["create-task", "Child Task", "--list", "inbox", "--parent", "   "],
            console);

        Assert.Equal(1, exitCode);
        Assert.Null(client.LastCreateRequest);
        Assert.Contains("Error: '--parent' must not be empty or whitespace.", console.Error.ToString());
    }

    private static AppConfig CreateConfig() => new()
    {
        ApiKey = "test-api-key",
        TeamId = "team-1",
        Lists = new Dictionary<string, string>
        {
            ["inbox"] = "list-42",
        },
    };

    private sealed class RecordingClickUpClient : IClickUpClient
    {
        public string? LastCreateListId { get; private set; }
        public CreateTaskRequest? LastCreateRequest { get; private set; }

        public Task<TaskSummary> CreateTaskAsync(string listId, CreateTaskRequest request, CancellationToken ct = default)
        {
            LastCreateListId = listId;
            LastCreateRequest = request;

            return Task.FromResult(new TaskSummary(
                Id: "task-1",
                Name: request.Name,
                Status: "to do",
                Priority: request.Priority?.ToString().ToLowerInvariant(),
                ParentId: request.ParentId,
                Url: "https://app.clickup.com/t/task-1",
                DueDate: request.DueDate,
                Description: request.Description,
                ListId: listId,
                ListName: "Inbox",
                CreatedAt: DateTimeOffset.UtcNow,
                UpdatedAt: DateTimeOffset.UtcNow,
                Subtasks: []));
        }

        public Task<TaskSummary> GetTaskAsync(string taskId, CancellationToken ct = default) =>
            throw new NotSupportedException();

        public Task<IReadOnlyList<TaskSummary>> GetTasksAsync(
            string teamId,
            bool includeSubtasks = true,
            int page = 0,
            IReadOnlyList<string>? listIds = null,
            IReadOnlyList<string>? statuses = null,
            DateTimeOffset? dueDateGt = null,
            DateTimeOffset? dueDateLt = null,
            CancellationToken ct = default) =>
            throw new NotSupportedException();
    }
}
