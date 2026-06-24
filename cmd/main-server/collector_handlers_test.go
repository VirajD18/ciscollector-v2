package main

import (
	"testing"
	"time"
)

func TestConnectionStatus(t *testing.T) {
	tests := []struct {
		name      string
		lastSeen  time.Time
		threshold time.Duration
		want      string
	}{
		{
			name:      "online",
			lastSeen:  time.Now().Add(-10 * time.Second),
			threshold: 60 * time.Second,
			want:      "online",
		},
		{
			name:      "offline_after_90s",
			lastSeen:  time.Now().Add(-120 * time.Second),
			threshold: 90 * time.Second,
			want:      "offline",
		},
		{
			name:      "online_within_90s",
			lastSeen:  time.Now().Add(-60 * time.Second),
			threshold: 90 * time.Second,
			want:      "online",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := connectionStatus(tc.lastSeen, tc.threshold); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
