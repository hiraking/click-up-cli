namespace ClickUpClient.Raw;

internal sealed class RawAssignee
{
    public int Id { get; init; }

    public string Username { get; init; } = string.Empty;

    public string? Email { get; init; }
}
