// internal/dateparse/parse_test.go
package dateparse_test

import (
	"testing"
	"time"

	"github.com/hiraking/click-up-cli/internal/dateparse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseISO_WithOffset(t *testing.T) {
	// オフセット付き → そのまま解析
	s := "2026-04-19T15:09:41.393+09:00"
	got, err := dateparse.ParseISO(s, "due-after")
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.April, got.Month())
	assert.Equal(t, 19, got.Day())
	assert.Equal(t, 15, got.Hour())
	// 元のタイムゾーンオフセットが保持されている
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
}

func TestParseISO_WithZ(t *testing.T) {
	// Z 付き → UTC として解析
	s := "2026-04-19T06:09:41.393Z"
	got, err := dateparse.ParseISO(s, "due-after")
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
}

func TestParseISO_WithoutOffset(t *testing.T) {
	// オフセットなし → JST (+09:00) として解析
	s := "2026-04-19T15:09:41"
	got, err := dateparse.ParseISO(s, "due-after")
	require.NoError(t, err)
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, 15, got.Hour())
}

func TestParseISO_InvalidString(t *testing.T) {
	_, err := dateparse.ParseISO("not-a-date", "due-after")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "due-after")
	assert.Contains(t, err.Error(), "not-a-date")
}
