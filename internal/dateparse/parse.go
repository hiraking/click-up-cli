// internal/dateparse/parse.go
package dateparse

import (
	"fmt"
	"time"
)

var jst = time.FixedZone("JST", 9*60*60)

// ParseISO は ISO 8601 形式の日時文字列を time.Time に変換する。
// タイムゾーンオフセットが含まれていない場合は JST (+09:00) として解析する。
func ParseISO(s string) (time.Time, error) {
	// オフセット付き / Z 付きのフォーマット群
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	// オフセットなし → JST として解析
	noOffsetFormats := []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, f := range noOffsetFormats {
		if t, err := time.ParseInLocation(f, s, jst); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("dateparse: cannot parse %q as ISO 8601", s)
}
