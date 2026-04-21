using ClickUpCli;

namespace ClickUpClient.Tests.Http;

public class DateParsingTests
{
    [Theory]
    [InlineData("2026-05-01", 2026, 5, 1, 0, 0, 0)]
    [InlineData("2026-05-01T09:00", 2026, 5, 1, 9, 0, 0)]
    public void ParseIsoDate_WithoutOffset_UsesJst(
        string input,
        int year,
        int month,
        int day,
        int hour,
        int minute,
        int second)
    {
        var result = DateParsing.ParseIsoDate(input, "--date");

        Assert.Equal(
            new DateTimeOffset(year, month, day, hour, minute, second, TimeSpan.FromHours(9)),
            result);
    }

    [Fact]
    public void ParseIsoDate_WithZuluOffset_PreservesUtc()
    {
        var result = DateParsing.ParseIsoDate("2026-05-01T00:00:00Z", "--date");

        Assert.Equal(DateTimeOffset.Parse("2026-05-01T00:00:00Z"), result);
    }

    [Fact]
    public void ParseIsoDate_WithExplicitPositiveOffset_PreservesOffset()
    {
        var result = DateParsing.ParseIsoDate("2026-05-01T09:00:00+09:00", "--date");

        Assert.Equal(DateTimeOffset.Parse("2026-05-01T09:00:00+09:00"), result);
    }

    [Fact]
    public void ParseIsoDate_WithExplicitNegativeOffset_PreservesOffset()
    {
        var result = DateParsing.ParseIsoDate("2026-05-01T00:00:00-07:00", "--date");

        Assert.Equal(DateTimeOffset.Parse("2026-05-01T00:00:00-07:00"), result);
    }
}
