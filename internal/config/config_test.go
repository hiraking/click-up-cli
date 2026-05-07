package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hiraking/click-up-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_EmptyPath_EnvVarsOnly(t *testing.T) {
	t.Setenv("CLICKUP_API_KEY", "pk_env_key")
	t.Setenv("CLICKUP_TEAM_ID", "env_team_id")

	cfg, err := config.Load("")
	require.NoError(t, err)
	assert.Equal(t, "pk_env_key", cfg.APIKey)
	assert.Equal(t, "env_team_id", cfg.TeamID)
	assert.Empty(t, cfg.Lists)
}

func TestLoad_EmptyPath_MissingEnvVars(t *testing.T) {
	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	_, err := config.Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiKey is required")
}

func TestLoad_EnvVarOverridesFile_APIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"apiKey":"file_key","teamId":"file_team"}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "env_key")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "env_key", cfg.APIKey)
	assert.Equal(t, "file_team", cfg.TeamID)
}

func TestLoad_FileOnly_EnvVarsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"apiKey":"file_key","teamId":"file_team","lists":{"work":"123"}}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "file_key", cfg.APIKey)
	assert.Equal(t, "file_team", cfg.TeamID)
	assert.Equal(t, map[string]string{"work": "123"}, cfg.Lists)
}

func TestLoad_ValidTimezone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"apiKey":"k","teamId":"t","timezone":"Asia/Tokyo"}`), 0600))
	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tokyo", cfg.Timezone)
}

func TestLoad_InvalidTimezone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path,
		[]byte(`{"apiKey":"k","teamId":"t","timezone":"Not/A/Zone"}`), 0600))
	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	_, err := config.Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid timezone "Not/A/Zone"`)
}

func TestTimezoneLocation_EmptyReturnsUTC(t *testing.T) {
	cfg := &config.AppConfig{}
	assert.Equal(t, time.UTC, cfg.TimezoneLocation())
}

func TestTimezoneLocation_ValidZone(t *testing.T) {
	cfg := &config.AppConfig{Timezone: "Asia/Tokyo"}
	loc := cfg.TimezoneLocation()
	require.NotNil(t, loc)
	assert.Equal(t, "Asia/Tokyo", loc.String())
}

func TestLoad_TaskTypes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
		"apiKey": "pk_key",
		"teamId": "team",
		"taskTypes": { "milestone": 1, "project": 1001 }
	}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"milestone": 1, "project": 1001}, cfg.TaskTypes)
}

func TestLoad_TaskTypes_Absent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"apiKey":"pk_key","teamId":"team"}`), 0600))

	t.Setenv("CLICKUP_API_KEY", "")
	t.Setenv("CLICKUP_TEAM_ID", "")

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Empty(t, cfg.TaskTypes)
}
