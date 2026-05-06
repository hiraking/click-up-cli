// cmd/clickup/helpers_test.go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pk_abcdefgh1234", "****1234"},
		{"12345", "****2345"},
		{"abcd", "****"},
		{"abc", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, maskAPIKey(tt.input))
		})
	}
}
