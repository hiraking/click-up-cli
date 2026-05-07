package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// ErrConfigNotFound は設定ファイルが見つからないことを示すセンチネルエラー。
var ErrConfigNotFound = errors.New("config: file not found")

// AppConfig はアプリケーション設定を表す。
type AppConfig struct {
	APIKey    string            `mapstructure:"apiKey"`
	TeamID    string            `mapstructure:"teamId"`
	Lists     map[string]string `mapstructure:"lists"`
	Timezone  string            `mapstructure:"timezone"`
	TaskTypes map[string]int    `mapstructure:"taskTypes"`
}

// TimezoneLocation は Timezone フィールドを *time.Location に変換して返す。
// Timezone が空のときは time.UTC を返す。
// Load() でバリデーション済みのため panic しない。
func (c *AppConfig) TimezoneLocation() *time.Location {
	if c.Timezone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// Load は設定を読み込む。path が空のときはファイルを読まず env var のみを使う。
// path が指定された場合はファイルを必須とする。
// CLICKUP_API_KEY / CLICKUP_TEAM_ID 環境変数はファイルの値を上書きする。
// apiKey または teamId が空の場合はバリデーションエラーを返す。
// timezone が空でない場合、有効な IANA タイムゾーン名かどうかを検証する。
func Load(path string) (*AppConfig, error) {
	v := viper.New()

	if path != "" {
		v.SetConfigFile(path)

		if err := v.ReadInConfig(); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", ErrConfigNotFound)
			}
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
				return nil, fmt.Errorf("config file not found: %w", ErrConfigNotFound)
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
	if cfg.Timezone != "" {
		if _, err := time.LoadLocation(cfg.Timezone); err != nil {
			return nil, fmt.Errorf("config: invalid timezone %q: %w", cfg.Timezone, err)
		}
	}

	return &cfg, nil
}
