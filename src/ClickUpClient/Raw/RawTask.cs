namespace ClickUpClient.Raw;

internal sealed class RawTask
{
    public string Id { get; init; } = string.Empty;

    public string Name { get; init; } = string.Empty;

    public string? Description { get; init; }

    public RawTaskStatus Status { get; init; } = new();

    /// <summary>null = ルートタスク / 文字列 = 親タスクの ID</summary>
    public string? Parent { get; init; }

    public string? TopLevelParent { get; init; }

    /// <summary>null = 優先度未設定</summary>
    public RawPriority? Priority { get; init; }

    public List<RawAssignee> Assignees { get; init; } = [];

    /// <summary>Unix ミリ秒文字列 (例: "1508369194377")。null あり</summary>
    public string? DueDate { get; init; }

    /// <summary>Unix ミリ秒文字列。null あり</summary>
    public string? StartDate { get; init; }

    /// <summary>ミリ秒文字列 (例: "8640000")。null あり</summary>
    public string? TimeEstimate { get; init; }

    /// <summary>ミリ秒数値。null あり</summary>
    public long? TimeSpent { get; init; }

    /// <summary>Unix ミリ秒文字列</summary>
    public string DateCreated { get; init; } = string.Empty;

    /// <summary>Unix ミリ秒文字列</summary>
    public string DateUpdated { get; init; } = string.Empty;

    public string Url { get; init; } = string.Empty;

    public RawListRef List { get; init; } = new();

    public string TeamId { get; init; } = string.Empty;
}
