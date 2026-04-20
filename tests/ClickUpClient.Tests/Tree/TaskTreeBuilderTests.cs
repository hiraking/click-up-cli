using ClickUpClient.Raw;
using ClickUpClient.Tree;

namespace ClickUpClient.Tests.Tree;

public class TaskTreeBuilderTests
{
    private static RawTask MakeTask(string id, string name, string? parent = null) => new()
    {
        Id = id,
        Name = name,
        Parent = parent,
        Status = new RawTaskStatus { Id = "s1", Status = "to do", Color = "#f8ae00", Type = "open" },
        Assignees = [],
        DateCreated = "1567780450202",
        DateUpdated = "1567780450202",
        Url = $"https://app.clickup.com/t/{id}",
        List = new RawListRef { Id = "list1", Name = "Test List" },
        TeamId = "team1",
    };

    [Fact]
    public void Build_EmptyList_ReturnsEmpty()
    {
        var result = TaskTreeBuilder.Build([]);

        Assert.Empty(result);
    }

    [Fact]
    public void Build_SingleRootTask_ReturnsSingleNode()
    {
        var tasks = new[] { MakeTask("root1", "Root Task") };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Single(result);
        Assert.Equal("root1", result[0].Id);
        Assert.Empty(result[0].Subtasks);
    }

    [Fact]
    public void Build_ParentAndChild_ChildIsNestedUnderParent()
    {
        var tasks = new[]
        {
            MakeTask("parent1", "Parent"),
            MakeTask("child1", "Child", parent: "parent1"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Single(result);
        var parent = result[0];
        Assert.Equal("parent1", parent.Id);
        Assert.Single(parent.Subtasks);
        Assert.Equal("child1", parent.Subtasks[0].Id);
    }

    [Fact]
    public void Build_MultipleRoots_AllAppearAtTopLevel()
    {
        var tasks = new[]
        {
            MakeTask("root1", "Root 1"),
            MakeTask("root2", "Root 2"),
            MakeTask("root3", "Root 3"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Equal(3, result.Count);
        Assert.Contains(result, t => t.Id == "root1");
        Assert.Contains(result, t => t.Id == "root2");
        Assert.Contains(result, t => t.Id == "root3");
    }

    [Fact]
    public void Build_DeepNesting_IsCorrectlyOrganized()
    {
        // root → child → grandchild
        var tasks = new[]
        {
            MakeTask("root1", "Root"),
            MakeTask("child1", "Child", parent: "root1"),
            MakeTask("grandchild1", "Grandchild", parent: "child1"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Single(result);
        var root = result[0];
        Assert.Equal("root1", root.Id);
        Assert.Single(root.Subtasks);

        var child = root.Subtasks[0];
        Assert.Equal("child1", child.Id);
        Assert.Single(child.Subtasks);

        var grandchild = child.Subtasks[0];
        Assert.Equal("grandchild1", grandchild.Id);
        Assert.Empty(grandchild.Subtasks);
    }

    [Fact]
    public void Build_MultipleChildrenUnderSameParent_AllNested()
    {
        var tasks = new[]
        {
            MakeTask("parent1", "Parent"),
            MakeTask("child1", "Child 1", parent: "parent1"),
            MakeTask("child2", "Child 2", parent: "parent1"),
            MakeTask("child3", "Child 3", parent: "parent1"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Single(result);
        Assert.Equal(3, result[0].Subtasks.Count);
    }

    [Fact]
    public void Build_OrphanedSubtask_TreatedAsRoot()
    {
        // parent が取得済みリストに存在しない → ルート扱い
        var tasks = new[]
        {
            MakeTask("child1", "Orphan", parent: "nonexistent-parent"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Single(result);
        Assert.Equal("child1", result[0].Id);
    }

    [Fact]
    public void Build_MixedRootsAndChildren_CorrectStructure()
    {
        var tasks = new[]
        {
            MakeTask("root1", "Root 1"),
            MakeTask("root2", "Root 2"),
            MakeTask("child1a", "Child 1A", parent: "root1"),
            MakeTask("child1b", "Child 1B", parent: "root1"),
            MakeTask("child2a", "Child 2A", parent: "root2"),
        };

        var result = TaskTreeBuilder.Build(tasks);

        Assert.Equal(2, result.Count);
        var r1 = result.First(t => t.Id == "root1");
        var r2 = result.First(t => t.Id == "root2");
        Assert.Equal(2, r1.Subtasks.Count);
        Assert.Single(r2.Subtasks);
    }
}
