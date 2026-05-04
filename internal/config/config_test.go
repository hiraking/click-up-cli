package config_test

import (
	"os"
	"path/filepath"
	"testing"

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
