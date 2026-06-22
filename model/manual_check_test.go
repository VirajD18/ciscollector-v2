package model

import (
	"strings"
	"testing"
)

func TestManualCheckTableDescriptionAndList_Text(t *testing.T) {
	tests := []struct {
		name string
		in   ManualCheckTableDescriptionAndList
		want []string
	}{
		{
			name: "list only no table",
			in: ManualCheckTableDescriptionAndList{
				Description: "Review accounts",
				List:        []string{"alice", "bob"},
			},
			want: []string{"Review accounts", "alice", "bob"},
		},
		{
			name: "table only",
			in: ManualCheckTableDescriptionAndList{
				Table: &SimpleTable{
					Columns: []string{"rolname"},
					Rows:    [][]interface{}{{"admin"}},
				},
			},
			want: []string{"rolname", "admin"},
		},
		{
			name: "description list and table",
			in: ManualCheckTableDescriptionAndList{
				Description: "NOTE - review",
				List:        []string{"hint"},
				Table: &SimpleTable{
					Columns: []string{"col"},
					Rows:    [][]interface{}{{"val"}},
				},
			},
			want: []string{"NOTE - review", "hint", "col", "val"},
		},
		{
			name: "error list path",
			in: ManualCheckTableDescriptionAndList{
				Description: "NOTE - Ensure excessive administrative privileges are revoked",
				List:        []string{"permission denied for table pg_roles"},
			},
			want: []string{
				"NOTE - Ensure excessive administrative privileges are revoked",
				"permission denied for table pg_roles",
			},
		},
		{
			name: "empty",
			in:   ManualCheckTableDescriptionAndList{},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Text()
			for _, sub := range tt.want {
				if !strings.Contains(strings.ToLower(got), strings.ToLower(sub)) {
					t.Fatalf("Text() = %q, want substring %q", got, sub)
				}
			}
		})
	}
}

func TestManualCheckTableDescriptionAndList_TextNilTableNoPanic(t *testing.T) {
	m := ManualCheckTableDescriptionAndList{
		Description: "desc",
		List:        []string{"one"},
	}
	if m.Table != nil {
		t.Fatal("expected nil table in fixture")
	}
	_ = m.Text()
}
