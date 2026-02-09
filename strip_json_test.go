package main

import (
	"testing"
)

func Test_stripInvalidJSONPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "valid JSON array passes through",
			input: []byte(`[{"key":"value"}]`),
			want:  `[{"key":"value"}]`,
		},
		{
			name:  "valid JSON object passes through",
			input: []byte(`{"key":"value"}`),
			want:  `{"key":"value"}`,
		},
		{
			name:  "leading garbage stripped to valid JSON",
			input: []byte(`some garbage text[{"key":"value"}]`),
			want:  `[{"key":"value"}]`,
		},
		{
			name:  "empty input returns empty",
			input: []byte{},
			want:  "",
		},
		{
			name:  "nil input returns empty",
			input: nil,
			want:  "",
		},
		{
			name:  "no valid JSON returns empty",
			input: []byte("this is not json at all"),
			want:  "",
		},
		{
			name:  "BOM prefix stripped",
			input: append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"key":"value"}`)...),
			want:  `{"key":"value"}`,
		},
		{
			name:  "whitespace before JSON preserved",
			input: []byte(`  {"key":"value"}`),
			want:  `  {"key":"value"}`,
		},
		{
			name:  "tsh re-login prefix stripped",
			input: []byte("Re-login successful\n[{\"kind\":\"node\"}]"),
			want:  "\n[{\"kind\":\"node\"}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(stripInvalidJSONPrefix(tt.input))
			if got != tt.want {
				t.Errorf("stripInvalidJSONPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
