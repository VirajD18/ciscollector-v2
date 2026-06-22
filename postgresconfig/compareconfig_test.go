package postgresconfig

import "testing"

func TestNormalizeGucValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "on uppercase", in: "ON", want: "on"},
		{name: "on mixed", in: " On ", want: "on"},
		{name: "true maps on", in: "true", want: "on"},
		{name: "yes maps on", in: "yes", want: "on"},
		{name: "off lowercase", in: "off", want: "off"},
		{name: "false maps off", in: "false", want: "off"},
		{name: "size lowercased", in: "128MB", want: "128mb"},
		{name: "empty", in: "  ", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeGucValue(tt.in); got != tt.want {
				t.Fatalf("NormalizeGucValue(%q)=%q want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCompareAgainstBaseline(t *testing.T) {
	baseline := map[string]string{
		"ssl":             "on",
		"shared_buffers":  "128MB",
		"max_connections": "200",
	}

	tests := []struct {
		name       string
		live       map[string]string
		wantStatus map[string]DriftStatus
	}{
		{
			name: "all match case insensitive",
			live: map[string]string{
				"ssl":             "ON",
				"shared_buffers":  "128mb",
				"max_connections": "200",
			},
			wantStatus: map[string]DriftStatus{
				"ssl": DriftMatch, "shared_buffers": DriftMatch, "max_connections": DriftMatch,
			},
		},
		{
			name: "one drift",
			live: map[string]string{
				"ssl":             "off",
				"shared_buffers":  "128MB",
				"max_connections": "200",
			},
			wantStatus: map[string]DriftStatus{
				"ssl": DriftDiff, "shared_buffers": DriftMatch, "max_connections": DriftMatch,
			},
		},
		{
			name: "missing key",
			live: map[string]string{
				"shared_buffers":  "128MB",
				"max_connections": "200",
			},
			wantStatus: map[string]DriftStatus{
				"ssl": DriftMissing, "shared_buffers": DriftMatch, "max_connections": DriftMatch,
			},
		},
		{
			name:       "empty live map",
			live:       map[string]string{},
			wantStatus: map[string]DriftStatus{"ssl": DriftMissing, "shared_buffers": DriftMissing, "max_connections": DriftMissing},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := CompareAgainstBaseline(baseline, tt.live)
			if len(rows) != len(tt.wantStatus) {
				t.Fatalf("len(rows)=%d want %d", len(rows), len(tt.wantStatus))
			}
			for _, row := range rows {
				want, ok := tt.wantStatus[row.GUC]
				if !ok {
					t.Fatalf("unexpected guc %q", row.GUC)
				}
				if row.Status != want {
					t.Fatalf("%s: status=%q want %q (baseline=%q live=%q)", row.GUC, row.Status, want, row.Baseline, row.Live)
				}
			}
		})
	}
}
