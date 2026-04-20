using ClickUpClient.Http;

namespace ClickUpClient.Tests.Http;

public class ClickUpHttpClientUrlTests
{
    // BuildGetTasksUrl は private なので、実際の HttpClient を使って送信された Request.RequestUri を確認する

    private sealed class CapturingHandler : HttpMessageHandler
    {
        public HttpRequestMessage? LastRequest { get; private set; }

        protected override Task<HttpResponseMessage> SendAsync(
            HttpRequestMessage request, CancellationToken cancellationToken)
        {
            LastRequest = request;
            var response = new HttpResponseMessage(System.Net.HttpStatusCode.OK)
            {
                Content = new StringContent("""{"tasks":[],"last_page":true}""",
                    System.Text.Encoding.UTF8, "application/json"),
            };
            return Task.FromResult(response);
        }
    }

    private static (ClickUpHttpClient client, CapturingHandler handler) CreateClient()
    {
        var handler = new CapturingHandler();
        var http = new HttpClient(handler)
        {
            BaseAddress = new Uri("https://api.clickup.com/api/"),
        };
        return (new ClickUpHttpClient(http), handler);
    }

    [Fact]
    public async Task GetTasksAsync_BasicParams_BuildsCorrectUrl()
    {
        var (client, handler) = CreateClient();

        await client.GetTasksAsync("team123");

        var uri = handler.LastRequest!.RequestUri!;
        Assert.Equal("/api/v2/team/team123/task", uri.AbsolutePath);
        Assert.Contains("subtasks=true", uri.Query);
        Assert.Contains("page=0", uri.Query);
    }

    [Fact]
    public async Task GetTasksAsync_WithListIds_AppendsRepeatedParams()
    {
        var (client, handler) = CreateClient();

        await client.GetTasksAsync("team1", listIds: ["list1", "list2"]);

        var query = handler.LastRequest!.RequestUri!.Query;
        Assert.Contains("list_ids[]=list1", query);
        Assert.Contains("list_ids[]=list2", query);
    }

    [Fact]
    public async Task GetTasksAsync_WithStatuses_AppendsEncodedStatuses()
    {
        var (client, handler) = CreateClient();

        await client.GetTasksAsync("team1", statuses: ["in progress", "to do"]);

        var query = handler.LastRequest!.RequestUri!.Query;
        Assert.Contains("statuses[]=in%20progress", query);
        Assert.Contains("statuses[]=to%20do", query);
    }

    [Fact]
    public async Task GetTasksAsync_WithDueDateGt_AppendsUnixMs()
    {
        var (client, handler) = CreateClient();
        var date = new DateTimeOffset(2024, 1, 1, 0, 0, 0, TimeSpan.Zero);

        await client.GetTasksAsync("team1", dueDateGt: date);

        var query = handler.LastRequest!.RequestUri!.Query;
        Assert.Contains($"due_date_gt={date.ToUnixTimeMilliseconds()}", query);
    }

    [Fact]
    public async Task GetTasksAsync_WithDueDateLt_AppendsUnixMs()
    {
        var (client, handler) = CreateClient();
        var date = new DateTimeOffset(2024, 12, 31, 0, 0, 0, TimeSpan.Zero);

        await client.GetTasksAsync("team1", dueDateLt: date);

        var query = handler.LastRequest!.RequestUri!.Query;
        Assert.Contains($"due_date_lt={date.ToUnixTimeMilliseconds()}", query);
    }

    [Fact]
    public async Task GetTasksAsync_NullFilters_NoFilterParams()
    {
        var (client, handler) = CreateClient();

        await client.GetTasksAsync("team1", listIds: null, statuses: null, dueDateGt: null, dueDateLt: null);

        var query = handler.LastRequest!.RequestUri!.Query;
        Assert.DoesNotContain("list_ids", query);
        Assert.DoesNotContain("statuses", query);
        Assert.DoesNotContain("due_date", query);
    }
}
