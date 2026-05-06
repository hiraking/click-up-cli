// internal/dateparse/parse.go
package dateparse

import (
	"fmt"
	"time"
)

// ParseISO は ISO 8601 形式の日時文字列を time.Time に変換する。
// タイムゾーンオフセットが含まれていない場合は loc として解析する。loc が nil のときは time.UTC を使用する。
// optionName はエラーメッセージに使用するオプション名（例: "due-after"）。
func ParseISO(value, optionName string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.UTC
	}

	// オフセット付き / Z 付きのフォーマット群
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}

	// オフセットなし → loc として解析
	noOffsetFormats := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range noOffsetFormats {
		if t, err := time.ParseInLocation(f, value, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf(
		"Error: '--%s' value '%s' is not a valid ISO 8601 datetime.", optionName, value)
}
