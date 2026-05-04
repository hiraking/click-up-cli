package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// AppConfig はアプリケーション設定を表す。
type AppConfig struct {
	APIKey string            `mapstructure:"apiKey"`
	TeamID string            `mapstructure:"teamId"`
	Lists  map[string]string `mapstructure:"lists"`
}

// Load は設定を読み込む。path が空のときはファイルを読まず env var のみを使う。
// path が指定された場合はファイルを必須とする。
// CLICKUP_API_KEY / CLICKUP_TEAM_ID 環境変数はファイルの値を上書きする。
// apiKey または teamId が空の場合はバリデーションエラーを返す。
func Load(path string) (*AppConfig, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)

		if err := v.ReadInConfig(); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
			}
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if key := os.Getenv("CLICKUP_API_KEY"); key != "" {
		cfg.APIKey = key
	}
	if team := os.Getenv("CLICKUP_TEAM_ID"); team != "" {
		cfg.TeamID = team
	}

	if cfg.APIKey == "" {
		return nil, errors.New("config: apiKey is required")
	}
	if cfg.TeamID == "" {
		return nil, errors.New("config: teamId is required")
	}

	return &cfg, nil
}
