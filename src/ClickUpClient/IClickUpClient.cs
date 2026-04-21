using ClickUpClient.Models;

namespace ClickUpClient;

/// <summary>ClickUp API クライアントのインターフェース。</summary>
public interface IClickUpClient
{
    /// <summary>
    /// ワークスペース内のタスク一覧をツリー構造で取得する。
    /// GET /v2/team/{teamId}/task
    /// </summary>
    /// <param name="teamId">ワークスペース (Team) ID</param>
    /// <param name="includeSubtasks">true の場合、サブタスクもフラットに取得してツリーに組み込む</param>
    /// <param name="page">ページ番号 (0 始まり)</param>
    /// <param name="listIds">絞り込むリスト ID のリスト。null の場合は絞り込みなし</param>
    /// <param name="statuses">絞り込むステータス名のリスト (例: ["in progress", "to do"])。null の場合は絞り込みなし</param>
    /// <param name="dueDateGt">この日時より後の due_date を持つタスクに絞り込む</param>
    /// <param name="dueDateLt">この日時より前の due_date を持つタスクに絞り込む</param>
    /// <param name="ct">キャンセルトークン</param>
    Task<IReadOnlyList<TaskSummary>> GetTasksAsync(
        string teamId,
        bool includeSubtasks = true,
        int page = 0,
        IReadOnlyList<string>? listIds = null,
        IReadOnlyList<string>? statuses = null,
        DateTimeOffset? dueDateGt = null,
        DateTimeOffset? dueDateLt = null,
        CancellationToken ct = default);

    /// <summary>
    /// 指定した task ID のタスクを取得する。
    /// GET /v2/task/{taskId}
    /// </summary>
    Task<TaskSummary> GetTaskAsync(string taskId, CancellationToken ct = default);

    /// <summary>
    /// 指定したリストにタスクを新規作成する。
    /// POST /v2/list/{listId}/task
    /// </summary>
    /// <param name="listId">タスクを作成するリスト ID</param>
    /// <param name="request">作成パラメータ</param>
    /// <param name="ct">キャンセルトークン</param>
    Task<TaskSummary> CreateTaskAsync(
        string listId,
        CreateTaskRequest request,
        CancellationToken ct = default);
}
