using System.Net;
using System.Text;
using System.Text.Json;
using ClickUpClient.Http;
using ClickUpClient.Models;

namespace ClickUpClient.Tests.Http;

public class ClickUpHttpClientCreateTaskTests
{
    private static readonly string SampleRawTask = """
        {
          "id": "abc123",
          "name": "New Task Name",
          "description": "New Task Description",
          "status": { "status": "to do", "color": "#f00", "type": "open", "orderindex": 0 },
          "parent": null,
          "top_level_parent": null,
          "priority": { "id": "3", "priority": "normal", "color": "#f00", "orderindex": "3" },
          "assignees": [],
          "due_date": "1508369194377",
          "start_date": null,
          "time_estimate": "8640000",
          "time_spent": 0,
          "date_created": "1567780450202",
          "date_updated": "1567780450202",
          "url": "https://app.clickup.com/t/abc123",
          "list": { "id": "list99", "name": "My List", "access": true },
          "team_id": "team1"
        }
        """;

    private sealed class CapturingHandler : HttpMessageHandler
    {
        public HttpRequestMessage? LastRequest { get; private set; }
        public string? LastRequestBody { get; private set; }
        private readonly string _responseBody;

        public CapturingHandler(string responseBody)
        {
            _responseBody = responseBody;
        }

        protected override async Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            LastRequest = request;
            if (request.Content is not null)
                LastRequestBody = await request.Content.ReadAsStringAsync(cancellationToken);

            return new HttpResponseMessage(HttpStatusCode.OK)
            {
                Content = new StringContent(_responseBody, Encoding.UTF8, "application/json"),
            };
        }
    }

    private static (ClickUpHttpClient client, CapturingHandler handler) CreateClient()
    {
        var handler = new CapturingHandler(SampleRawTask);
        var http = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://api.clickup.com/api/"),
        };
        return (new ClickUpHttpClient(http), handler);
    }

    [Fact]
    public async Task CreateTaskAsync_PostsToCorrectUrl()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Test Task"));

        Assert.Equal(HttpMethod.Post, handler.LastRequest!.Method);
        Assert.Equal("/api/v2/list/list42/task", handler.LastRequest.RequestUri!.AbsolutePath);
    }

    [Fact]
    public async Task CreateTaskAsync_SerializesNameInBody()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42", new CreateTaskRequest("My New Task"));

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal("My New Task", body.GetProperty("name").GetString());
    }

    [Fact]
    public async Task CreateTaskAsync_OmitsNullFieldsFromBody()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task Only Name"));

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.False(body.TryGetProperty("description", out _));
        Assert.False(body.TryGetProperty("status", out _));
        Assert.False(body.TryGetProperty("priority", out _));
        Assert.False(body.TryGetProperty("due_date", out _));
    }

    [Fact]
    public async Task CreateTaskAsync_SerializesPriority()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { Priority = TaskPriority.High });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(2, body.GetProperty("priority").GetInt32());
    }

    [Fact]
    public async Task CreateTaskAsync_SerializesParent()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { ParentId = "parent-123" });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal("parent-123", body.GetProperty("parent").GetString());
    }

    [Fact]
    public async Task CreateTaskAsync_SerializesParentAlongsideStartAndDueDates()
    {
        var (client, handler) = CreateClient();
        var startDate = new DateTimeOffset(2026, 4, 25, 9, 0, 0, TimeSpan.FromHours(9));
        var dueDate = new DateTimeOffset(2026, 5, 1, 18, 30, 0, TimeSpan.FromHours(9));

        await client.CreateTaskAsync(
            "list42",
            new CreateTaskRequest("Task")
            {
                ParentId = "parent-456",
                StartDate = startDate,
                DueDate = dueDate,
            });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal("parent-456", body.GetProperty("parent").GetString());
        Assert.Equal(startDate.ToUnixTimeMilliseconds(), body.GetProperty("start_date").GetInt64());
        Assert.True(body.GetProperty("start_date_time").GetBoolean());
        Assert.Equal(dueDate.ToUnixTimeMilliseconds(), body.GetProperty("due_date").GetInt64());
        Assert.True(body.GetProperty("due_date_time").GetBoolean());
    }

    [Fact]
    public async Task CreateTaskAsync_DueDateWithTime_SetsDueDateTimeTrue()
    {
        var (client, handler) = CreateClient();
        // 2026-05-01 09:00 JST
        var dueDate = new DateTimeOffset(2026, 5, 1, 9, 0, 0, TimeSpan.FromHours(9));

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { DueDate = dueDate });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(dueDate.ToUnixTimeMilliseconds(), body.GetProperty("due_date").GetInt64());
        Assert.True(body.GetProperty("due_date_time").GetBoolean());
    }

    [Fact]
    public async Task CreateTaskAsync_DueDateWithoutTime_SetsDueDateTimeFalse()
    {
        var (client, handler) = CreateClient();
        // date-only: midnight JST
        var dueDate = new DateTimeOffset(2026, 5, 1, 0, 0, 0, TimeSpan.FromHours(9));

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { DueDate = dueDate });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(dueDate.ToUnixTimeMilliseconds(), body.GetProperty("due_date").GetInt64());
        Assert.False(body.GetProperty("due_date_time").GetBoolean());
    }

    [Fact]
    public async Task CreateTaskAsync_StartDateWithTime_SetsStartDateTimeTrue()
    {
        var (client, handler) = CreateClient();
        var startDate = new DateTimeOffset(2026, 4, 25, 9, 0, 0, TimeSpan.FromHours(9));

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { StartDate = startDate });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(startDate.ToUnixTimeMilliseconds(), body.GetProperty("start_date").GetInt64());
        Assert.True(body.GetProperty("start_date_time").GetBoolean());
    }

    [Fact]
    public async Task CreateTaskAsync_StartDateWithoutTime_SetsStartDateTimeFalse()
    {
        var (client, handler) = CreateClient();
        var startDate = new DateTimeOffset(2026, 4, 25, 0, 0, 0, TimeSpan.FromHours(9));

        await client.CreateTaskAsync("list42", new CreateTaskRequest("Task") { StartDate = startDate });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(startDate.ToUnixTimeMilliseconds(), body.GetProperty("start_date").GetInt64());
        Assert.False(body.GetProperty("start_date_time").GetBoolean());
    }

    [Fact]
    public async Task CreateTaskAsync_TimeEstimate_ConvertsToMs()
    {
        var (client, handler) = CreateClient();

        await client.CreateTaskAsync("list42",
            new CreateTaskRequest("Task") { TimeEstimate = TimeSpan.FromMinutes(90) });

        var body = JsonDocument.Parse(handler.LastRequestBody!).RootElement;
        Assert.Equal(90 * 60 * 1000, body.GetProperty("time_estimate").GetInt32());
    }

    [Fact]
    public async Task CreateTaskAsync_TimeEstimateTooLarge_Throws()
    {
        var (client, _) = CreateClient();

        await Assert.ThrowsAsync<ArgumentOutOfRangeException>(() =>
            client.CreateTaskAsync(
                "list42",
                new CreateTaskRequest("Task")
                {
                    TimeEstimate = TimeSpan.FromMilliseconds((double)int.MaxValue + 1),
                }));
    }

    [Fact]
    public async Task CreateTaskAsync_ReturnsTaskSummaryFromResponse()
    {
        var (client, _) = CreateClient();

        var result = await client.CreateTaskAsync("list42", new CreateTaskRequest("New Task Name"));

        Assert.Equal("abc123", result.Id);
        Assert.Equal("New Task Name", result.Name);
        Assert.Equal("to do", result.Status);
        Assert.Equal("normal", result.Priority);
    }
}
