using ClickUpClient.Mapping;
using ClickUpClient.Models;
using ClickUpClient.Raw;

namespace ClickUpClient.Tree;

/// <summary>
/// フラットなタスクリストを親子関係に基づくツリー構造に変換する。
/// </summary>
internal static class TaskTreeBuilder
{
    /// <summary>
    /// RawTask のフラットリストからツリー構造を構築する。
    /// </summary>
    /// <param name="tasks">GetTasks で取得したフラットなタスクリスト</param>
    /// <returns>ルートタスクのみのリスト。各 TaskSummary の Subtasks に子が再帰的にネストされている</returns>
    public static IReadOnlyList<TaskSummary> Build(IEnumerable<RawTask> tasks)
    {
        var rawList = tasks.ToList();

        // まず全タスクを TaskSummary (Subtasks 空) に変換し、ID でルックアップできるようにする
        var summaryMap = rawList
            .Select(TaskMapper.ToSummary)
            .ToDictionary(s => s.Id);

        // 各タスクの子リストを構築
        var childrenMap = new Dictionary<string, List<TaskSummary>>();

        foreach (var summary in summaryMap.Values)
        {
            if (summary.ParentId is not null && summaryMap.ContainsKey(summary.ParentId))
            {
                if (!childrenMap.TryGetValue(summary.ParentId, out var children))
                {
                    children = [];
                    childrenMap[summary.ParentId] = children;
                }
                children.Add(summary);
            }
        }

        // 再帰的に Subtasks を埋めたツリーノードを構築する
        TaskSummary BuildNode(TaskSummary summary)
        {
            if (!childrenMap.TryGetValue(summary.Id, out var children))
                return summary;

            var subtasks = children
                .OrderBy(c => c.Id)
                .Select(BuildNode)
                .ToList()
                .AsReadOnly();

            return summary with { Subtasks = subtasks };
        }

        // parent が null または親が取得済みリストに存在しないものをルートとして扱う
        return summaryMap.Values
            .Where(s => s.ParentId is null || !summaryMap.ContainsKey(s.ParentId))
            .OrderBy(s => s.Id)
            .Select(BuildNode)
            .ToList()
            .AsReadOnly();
    }
}
