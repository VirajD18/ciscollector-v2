package service

import (
	"testing"
)

func TestGroupFleetCISRows(t *testing.T) {
	instanceDBs := map[string][]string{
		"localhost:5432": {"hej", "hej1", "hej3"},
	}
	tests := []struct {
		name    string
		rows    [][]string
		want    int
		inst    string
		dbLabel string
		posture string
	}{
		{
			name: "groups three databases on one instance",
			rows: [][]string{
				{"localhost:5432/hej", "69%", "8", "Open"},
				{"localhost:5432/hej1", "69%", "8", "Open"},
				{"localhost:5432/hej3", "69%", "8", "Open"},
			},
			want:    1,
			inst:    "localhost:5432",
			dbLabel: "3 (hej, hej1, hej3)",
			posture: "3/3 Failing",
		},
		{
			name: "partial failing uses instance total",
			rows: [][]string{
				{"localhost:5432/hej", "55%", "10", "Open"},
			},
			want:    1,
			inst:    "localhost:5432",
			dbLabel: "3 (hej, hej1, hej3)",
			posture: "1/3 Failing",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := groupFleetCISRows(tc.rows, instanceDBs)
			if len(got) != tc.want {
				t.Fatalf("rows=%d want %d: %v", len(got), tc.want, got)
			}
			if tc.want == 0 {
				return
			}
			row := got[0]
			if row[0] != tc.inst {
				t.Fatalf("instance=%q want %q", row[0], tc.inst)
			}
			if row[1] != tc.dbLabel {
				t.Fatalf("databases=%q want %q", row[1], tc.dbLabel)
			}
			if row[4] != tc.posture {
				t.Fatalf("posture=%q want %q", row[4], tc.posture)
			}
		})
	}
}

func TestFleetUniqueInstances(t *testing.T) {
	tests := []struct {
		name  string
		hosts map[string]bool
		want  int
	}{
		{
			name: "three db keys one instance",
			hosts: map[string]bool{
				"localhost:5432/hej":  true,
				"localhost:5432/hej1": true,
			},
			want: 1,
		},
		{
			name: "two instances",
			hosts: map[string]bool{
				"localhost:5432/hej": true,
				"localhost:5433/hej": true,
			},
			want: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := fleetUniqueInstances(tc.hosts); got != tc.want {
				t.Fatalf("fleetUniqueInstances()=%d want %d", got, tc.want)
			}
		})
	}
}

func TestFleetUserTableRowsWithDatabase(t *testing.T) {
	rows := [][]string{{"postgres", "localhost:5432/hej", "View detail"}}
	got := fleetUserTableRowsWithDatabase(rows, 1)
	if len(got) != 1 || len(got[0]) != 4 {
		t.Fatalf("unexpected shape: %v", got)
	}
	if got[0][1] != "localhost:5432" || got[0][2] != "hej" {
		t.Fatalf("got %v", got[0])
	}
}

func TestFleetRowsWithDatabaseColumn(t *testing.T) {
	rows := [][]string{{"localhost:5432/hej", "SSL off", "Critical", "Open"}}
	got := fleetRowsWithDatabaseColumn(rows, 0)
	if len(got) != 1 || len(got[0]) != 5 {
		t.Fatalf("unexpected shape: %v", got)
	}
	if got[0][0] != "localhost:5432" || got[0][1] != "hej" {
		t.Fatalf("got %v", got[0])
	}
}
