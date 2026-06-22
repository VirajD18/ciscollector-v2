package main

import (
	"testing"
)

func TestCollectorBodyTokenAllowed(t *testing.T) {
	tests := []struct {
		name      string
		bodyToken string
		expected  string
		want      bool
	}{
		{
			name:      "empty_body_token_allowed",
			bodyToken: "",
			expected:  "secret",
			want:      true,
		},
		{
			name:      "matching_token",
			bodyToken: "secret",
			expected:  "secret",
			want:      true,
		},
		{
			name:      "mismatch",
			bodyToken: "wrong",
			expected:  "secret",
			want:      false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := collectorBodyTokenAllowed(tc.bodyToken, tc.expected); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
