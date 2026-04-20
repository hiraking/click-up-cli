namespace ClickUpClient.Raw;

internal sealed class RawGetTasksResponse
{
    public List<RawTask> Tasks { get; init; } = [];

    public bool LastPage { get; init; }
}
