using System.Text.Json;
using System.Text.Json.Serialization;

namespace ClickUpCli;

internal sealed class AppConfig
{
    [JsonPropertyName("apiKey")]
    public string ApiKey { get; init; } = "";

    [JsonPropertyName("teamId")]
    public string TeamId { get; init; } = "";

    [JsonPropertyName("lists")]
    public Dictionary<string, string> Lists { get; init; } = new();
}

internal static class ConfigLoader
{
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        ReadCommentHandling = JsonCommentHandling.Skip,
        AllowTrailingCommas = true,
    };

    public static AppConfig Load()
    {
        var path = Path.Combine(AppContext.BaseDirectory, "config.json");
        if (!File.Exists(path))
            throw new FileNotFoundException(
                $"config.json not found at '{path}'. Copy config.sample.json to config.json and fill in your values.");

        var json = File.ReadAllText(path);
        var config = JsonSerializer.Deserialize<AppConfig>(json, JsonOptions)
            ?? throw new InvalidOperationException("config.json is empty or invalid JSON.");

        if (string.IsNullOrWhiteSpace(config.ApiKey))
            throw new InvalidOperationException("config.json: 'apiKey' is required.");
        if (string.IsNullOrWhiteSpace(config.TeamId))
            throw new InvalidOperationException("config.json: 'teamId' is required.");

        return config;
    }
}
