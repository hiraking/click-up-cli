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
	s := "2026-04-19T15:09:41.393+09:00"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.April, got.Month())
	assert.Equal(t, 19, got.Day())
	assert.Equal(t, 15, got.Hour())
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
}

func TestParseISO_WithZ(t *testing.T) {
	s := "2026-04-19T06:09:41.393Z"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
}

func TestParseISO_WithoutOffset_DefaultsToUTC(t *testing.T) {
	// nil loc → UTC
	s := "2026-04-19T15:09:41"
	got, err := dateparse.ParseISO(s, "due-after", nil)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, got.Location())
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, 15, got.Hour())
}

func TestParseISO_WithoutOffset_UsesProvidedLoc(t *testing.T) {
	// 明示的なタイムゾーンを渡した場合はそれを使う
	jst := time.FixedZone("JST", 9*60*60)
	s := "2026-04-19T15:09:41"
	got, err := dateparse.ParseISO(s, "due-after", jst)
	require.NoError(t, err)
	_, offset := got.Zone()
	assert.Equal(t, 9*3600, offset)
	assert.Equal(t, 15, got.Hour())
}

func TestParseISO_InvalidString(t *testing.T) {
	_, err := dateparse.ParseISO("not-a-date", "due-after", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "due-after")
	assert.Contains(t, err.Error(), "not-a-date")
}
