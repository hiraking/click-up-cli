namespace ClickUpClient.Raw;

/// <summary>POST /v2/list/{listId}/task のリクエストボディ。SnakeCaseLower で JSON シリアライズされる。</summary>
internal sealed class RawCreateTaskBody
{
    public string Name { get; init; } = string.Empty;
    public string? Parent { get; init; }
    public string? Description { get; init; }
    public string? Status { get; init; }

    /// <summary>1=urgent, 2=high, 3=normal, 4=low。null = 未設定</summary>
    public int? Priority { get; init; }

    /// <summary>Unix ミリ秒</summary>
    public long? DueDate { get; init; }

    /// <summary>true = 時刻あり</summary>
    public bool? DueDateTime { get; init; }

    /// <summary>Unix ミリ秒</summary>
    public long? StartDate { get; init; }

    /// <summary>true = 時刻あり</summary>
    public bool? StartDateTime { get; init; }

    /// <summary>ミリ秒</summary>
    public int? TimeEstimate { get; init; }
}
