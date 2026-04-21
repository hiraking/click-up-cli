namespace ClickUpClient.Models;

/// <summary>タスクの優先度。数値は ClickUp API の priority フィールド値に対応する。</summary>
public enum TaskPriority
{
    Urgent = 1,
    High = 2,
    Normal = 3,
    Low = 4,
}
