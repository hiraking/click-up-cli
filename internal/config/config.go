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

// Load は指定パスの JSON 設定ファイルを読み込み AppConfig を返す。
// ファイルが存在しない場合は os.ErrNotExist でラップしたエラーを返す。
// apiKey または teamId が空の場合はバリデーションエラーを返す。
func Load(path string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
		}
		// viper が返す *os.PathError をチェック
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			return nil, fmt.Errorf("config file not found: %w", os.ErrNotExist)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.APIKey == "" {
		return nil, errors.New("config: apiKey is required")
	}
	if cfg.TeamID == "" {
		return nil, errors.New("config: teamId is required")
	}

	return &cfg, nil
}
