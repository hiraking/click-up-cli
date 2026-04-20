using ClickUpClient.Models;
using ClickUpClient.Raw;

namespace ClickUpClient.Mapping;

/// <summary>RawTask → TaskSummary への変換。</summary>
internal static class TaskMapper
{
    /// <summary>
    /// RawTask を TaskSummary に変換する。
    /// Subtasks は空リストで返す (ツリー構築は TaskTreeBuilder が担当)。
    /// </summary>
    public static TaskSummary ToSummary(RawTask raw) =>
        new(
            Id: raw.Id,
            Name: raw.Name,
            Status: raw.Status.Status,
            Priority: raw.Priority?.Priority,
            ParentId: raw.Parent,
            Url: raw.Url,
            DueDate: ParseUnixMs(raw.DueDate),
            Description: raw.Description,
            ListId: raw.List.Id,
            ListName: raw.List.Name,
            CreatedAt: ParseUnixMsRequired(raw.DateCreated),
            UpdatedAt: ParseUnixMsRequired(raw.DateUpdated),
            Subtasks: []
        );

    private static DateTimeOffset? ParseUnixMs(string? value)
    {
        if (string.IsNullOrEmpty(value)) return null;
        if (!long.TryParse(value, out var ms)) return null;
        return DateTimeOffset.FromUnixTimeMilliseconds(ms);
    }

    private static DateTimeOffset ParseUnixMsRequired(string value)
    {
        if (!long.TryParse(value, out var ms))
            throw new FormatException($"Invalid Unix ms timestamp: '{value}'");
        return DateTimeOffset.FromUnixTimeMilliseconds(ms);
    }
}
