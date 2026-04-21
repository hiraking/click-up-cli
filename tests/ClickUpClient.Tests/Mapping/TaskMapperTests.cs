using ClickUpClient.Mapping;
using ClickUpClient.Raw;

namespace ClickUpClient.Tests.Mapping;

public class TaskMapperTests
{
    private static RawTask CreateRaw(
        string id = "task1",
        string name = "Task 1",
        string? parent = null,
        string? dueDate = null,
        string? description = null,
        RawPriority? priority = null,
        List<RawAssignee>? assignees = null) => new()
    {
        Id = id,
        Name = name,
        Description = description,
        Status = new RawTaskStatus { Id = "s1", Status = "in progress", Color = "#d3d3d3", Type = "custom" },
        Parent = parent,
        Priority = priority,
        Assignees = assignees ?? [],
        DueDate = dueDate,
        DateCreated = "1567780450202",
        DateUpdated = "1567780450202",
        Url = $"https://app.clickup.com/t/{id}",
        List = new RawListRef { Id = "list1", Name = "Sprint Backlog" },
        TeamId = "team1",
    };

    [Fact]
    public void ToSummary_BasicFields_AreMapped()
    {
        var raw = CreateRaw(id: "abc", name: "My Task");

        var summary = TaskMapper.ToSummary(raw);

        Assert.Equal("abc", summary.Id);
        Assert.Equal("My Task", summary.Name);
        Assert.Equal("in progress", summary.Status);
        Assert.Equal("https://app.clickup.com/t/abc", summary.Url);
        Assert.Equal("list1", summary.ListId);
        Assert.Equal("Sprint Backlog", summary.ListName);
    }

    [Fact]
    public void ToSummary_NullPriority_IsNull()
    {
        var raw = CreateRaw(priority: null);

        var summary = TaskMapper.ToSummary(raw);

        Assert.Null(summary.Priority);
    }

    [Fact]
    public void ToSummary_WithPriority_MapsDisplayName()
    {
        var raw = CreateRaw(priority: new RawPriority { Id = "3", Priority = "normal", Color = "#f8ae00" });

        var summary = TaskMapper.ToSummary(raw);

        Assert.Equal("normal", summary.Priority);
    }

    [Fact]
    public void ToSummary_WithParent_MapsParentId()
    {
        var raw = CreateRaw(parent: "parent123");

        var summary = TaskMapper.ToSummary(raw);

        Assert.Equal("parent123", summary.ParentId);
    }

    [Fact]
    public void ToSummary_NullParent_ParentIdIsNull()
    {
        var raw = CreateRaw(parent: null);

        var summary = TaskMapper.ToSummary(raw);

        Assert.Null(summary.ParentId);
    }

    [Fact]
    public void ToSummary_DueDate_ConvertsFromUnixMs()
    {
        var raw = CreateRaw(dueDate: "1508369194377");

        var summary = TaskMapper.ToSummary(raw);

        Assert.NotNull(summary.DueDate);
        Assert.Equal(DateTimeOffset.FromUnixTimeMilliseconds(1508369194377L), summary.DueDate);
    }

    [Fact]
    public void ToSummary_NullDueDate_IsNull()
    {
        var raw = CreateRaw(dueDate: null);

        var summary = TaskMapper.ToSummary(raw);

        Assert.Null(summary.DueDate);
    }

    [Fact]
    public void ToSummary_DateCreated_ConvertsFromUnixMs()
    {
        var raw = CreateRaw();

        var summary = TaskMapper.ToSummary(raw);

        Assert.Equal(DateTimeOffset.FromUnixTimeMilliseconds(1567780450202L), summary.CreatedAt);
    }

    [Fact]
    public void ToSummary_Subtasks_AreInitiallyEmpty()
    {
        var raw = CreateRaw();

        var summary = TaskMapper.ToSummary(raw);

        Assert.Empty(summary.Subtasks);
    }
}
