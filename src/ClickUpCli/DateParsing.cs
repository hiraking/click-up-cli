using System.Globalization;

namespace ClickUpCli;

public static class DateParsing
{
    public static DateTimeOffset ParseIsoDate(string value, string optionName)
    {
        if (!DateTimeOffset.TryParse(
                value,
                CultureInfo.InvariantCulture,
                DateTimeStyles.RoundtripKind,
                out var result))
        {
            throw new ArgumentException(
                $"'{optionName}' value '{value}' is not a valid ISO 8601 datetime.",
                optionName);
        }

        return HasExplicitOffset(value)
            ? result
            : new DateTimeOffset(result.DateTime, TimeSpan.FromHours(9));
    }

    private static bool HasExplicitOffset(string value)
    {
        var trimmed = value.Trim();
        return trimmed.EndsWith("Z", StringComparison.OrdinalIgnoreCase)
            || trimmed.IndexOf('+', 10) >= 0
            || trimmed.LastIndexOf('-') > 9;
    }
}
