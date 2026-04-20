namespace ClickUpClient.Raw;

internal sealed class RawTaskStatus
{
    public string Id { get; init; } = string.Empty;

    /// <summary>表示名 (例: "in progress", "to do")</summary>
    public string Status { get; init; } = string.Empty;

    public string Color { get; init; } = string.Empty;

    public string Type { get; init; } = string.Empty;
}
