namespace ClickUpClient.Models;

/// <summary>
/// エージェント向けの整形済みタスク DTO。
/// TaskSummary 自体がツリーノードを兼ねており、Subtasks に子タスクをネストで保持する。
/// </summary>
public sealed record TaskSummary(
    string Id,
    string Name,
    /// <summary>ステータス表示名 (例: "in progress", "to do")</summary>
    string Status,
    /// <summary>優先度表示名 (例: "urgent", "normal")。未設定の場合は null</summary>
    string? Priority,
    /// <summary>親タスクの ID。null = ルートタスク</summary>
    string? ParentId,
    string Url,
    DateTimeOffset? DueDate,
    string? Description,
    string ListId,
    string ListName,
    DateTimeOffset CreatedAt,
    DateTimeOffset UpdatedAt,
    IReadOnlyList<TaskSummary> Subtasks
);
