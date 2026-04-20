using System.Net.Http.Json;
using System.Text;
using System.Text.Json;
using ClickUpClient.Mapping;
using ClickUpClient.Models;
using ClickUpClient.Raw;
using ClickUpClient.Tree;

namespace ClickUpClient.Http;

/// <summary>
/// ClickUp REST API v2 の HTTP クライアント実装。
/// HttpClient はコンストラクタで受け取る (DI フレンドリー)。
/// Raw モデルへのデシリアライズ・TaskSummary への変換・ツリー構築をすべて内部で行い、
/// 外部には TaskSummary のみを返す。
/// </summary>
public sealed class ClickUpHttpClient : IClickUpClient
{
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
    };

    private readonly HttpClient _http;

    /// <param name="httpClient">
    /// BaseAddress を https://api.clickup.com/api/ に、
    /// Authorization ヘッダーを個人 API トークンに設定済みの HttpClient を渡す。
    /// </param>
    public ClickUpHttpClient(HttpClient httpClient)
    {
        _http = httpClient;
    }

    /// <inheritdoc/>
    public async Task<IReadOnlyList<TaskSummary>> GetTasksAsync(
        string teamId,
        bool includeSubtasks = true,
        int page = 0,
        IReadOnlyList<string>? listIds = null,
        IReadOnlyList<string>? statuses = null,
        DateTimeOffset? dueDateGt = null,
        DateTimeOffset? dueDateLt = null,
        CancellationToken ct = default)
    {
        var url = BuildGetTasksUrl(teamId, includeSubtasks, page, listIds, statuses, dueDateGt, dueDateLt);
        var response = await _http.GetAsync(url, ct).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        var raw = await response.Content
            .ReadFromJsonAsync<RawGetTasksResponse>(JsonOptions, ct)
            .ConfigureAwait(false)
            ?? throw new InvalidOperationException("API returned null response.");

        return TaskTreeBuilder.Build(raw.Tasks);
    }

    /// <inheritdoc/>
    public async Task<TaskSummary> GetTaskAsync(string taskId, CancellationToken ct = default)
    {
        var url = $"v2/task/{taskId}";
        var response = await _http.GetAsync(url, ct).ConfigureAwait(false);
        response.EnsureSuccessStatusCode();

        var raw = await response.Content
            .ReadFromJsonAsync<RawTask>(JsonOptions, ct)
            .ConfigureAwait(false)
            ?? throw new InvalidOperationException("API returned null response.");

        return TaskMapper.ToSummary(raw);
    }

    private static string BuildGetTasksUrl(
        string teamId,
        bool includeSubtasks,
        int page,
        IReadOnlyList<string>? listIds,
        IReadOnlyList<string>? statuses,
        DateTimeOffset? dueDateGt,
        DateTimeOffset? dueDateLt)
    {
        var sb = new StringBuilder($"v2/team/{teamId}/task");
        var query = new List<string>
        {
            $"subtasks={includeSubtasks.ToString().ToLower()}",
            $"page={page}",
        };

        if (listIds is { Count: > 0 })
            foreach (var id in listIds)
                query.Add($"list_ids[]={Uri.EscapeDataString(id)}");

        if (statuses is { Count: > 0 })
            foreach (var status in statuses)
                query.Add($"statuses[]={Uri.EscapeDataString(status)}");

        if (dueDateGt.HasValue)
            query.Add($"due_date_gt={dueDateGt.Value.ToUnixTimeMilliseconds()}");

        if (dueDateLt.HasValue)
            query.Add($"due_date_lt={dueDateLt.Value.ToUnixTimeMilliseconds()}");

        sb.Append('?');
        sb.Append(string.Join('&', query));
        return sb.ToString();
    }
}
