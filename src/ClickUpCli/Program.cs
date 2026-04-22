using System.CommandLine;
using ClickUpCli;

return await CliApplication.CreateRootCommand().InvokeAsync(args);
