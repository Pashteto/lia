package hasher_test

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"

	"gateguard/internal/pkg/hasher"
)

const base62Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func TestHashToFixed8Chars(t *testing.T) {
	tests := []struct {
		input        string
		expectedHash string
	}{
		{"example_input_string", "bTnKYTvW"},
		{"another_example", "1ZUacHwJ"},
		{"1234567890", "6zaw2xo2"},
		{"", "03GHTys4"},
		{"special_characters_!@#", "S3G7JQmN"},
	}

	for _, test := range tests {
		result := hasher.HashToFixed8Chars(test.input)
		assert.Equal(t, 8, utf8.RuneCountInString(result), "result should be 8 characters long")
		assert.Equal(t, test.expectedHash, result, "should equal to expected hash")

		for _, char := range result {
			assert.Contains(t, base62Chars, string(char), "result should only contain base62 characters")
		}
	}
}
