namespace ClickUpClient.Raw;

internal sealed class RawPriority
{
    public string Id { get; init; } = string.Empty;

    /// <summary>表示名 (例: "urgent", "high", "normal", "low")</summary>
    public string Priority { get; init; } = string.Empty;

    public string Color { get; init; } = string.Empty;
}
