namespace ClickUpClient.Models;

/// <summary>タスク作成リクエストのパラメータ。</summary>
public sealed record CreateTaskRequest(string Name)
{
    public string? Description { get; init; }

    /// <summary>ステータス表示名 (例: "to do", "in progress")</summary>
    public string? Status { get; init; }

    public TaskPriority? Priority { get; init; }

    /// <summary>
    /// 期日。TimeOfDay が Zero でない場合は時刻あり (due_date_time=true) として扱う。
    /// </summary>
    public DateTimeOffset? DueDate { get; init; }

    /// <summary>
    /// 開始日。TimeOfDay が Zero でない場合は時刻あり (start_date_time=true) として扱う。
    /// </summary>
    public DateTimeOffset? StartDate { get; init; }

    /// <summary>見積もり時間。API には ms 整数として送信される。</summary>
    public TimeSpan? TimeEstimate { get; init; }
}
