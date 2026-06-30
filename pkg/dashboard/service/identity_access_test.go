package service

import (
	"context"
	"testing"
	"time"

	"github.com/VirajD18/ciscollector-v2/pkg/postgresdb"
	"github.com/VirajD18/ciscollector-v2/pkg/reportstore"
)

func TestIdentityAccessFromUsersReport(t *testing.T) {
	db := OpenTestSQLiteDB(t)
	defer db.Close()

	pg := &postgresdb.Postgres{Host: "localhost", Port: "5432"}
	PersistTestScanResult(t, db, samplePostgresUsersReport(), reportstore.RunMeta{
		Trigger:    "test",
		RunnerName: "postgres_cis",
		Postgres:   pg,
		StartedAt:  time.Now().UTC(),
		FinishedAt: time.Now().UTC(),
		RunStatus:  "success",
	}, "test-node", pg.Host)

	resp, err := NewSQLiteService(db).Strategic(context.Background(), "30d")
	if err != nil {
		t.Fatal(err)
	}
	r := resp.Ranges["30d"]
	if len(r.Privs) != 1 {
		t.Fatalf("privs len %d", len(r.Privs))
	}
	if r.Privs[0].A != 100 {
		t.Fatalf("admin excess want 100 got %d (postgres+romin superuser)", r.Privs[0].A)
	}
	if r.Hygiene.Inactive != 100 {
		t.Fatalf("hygiene inactive=%d want 100 (password expiry not set)", r.Hygiene.Inactive)
	}
	if r.Cred.Weak != 1 {
		t.Fatalf("cred weak=%d want 1 host with weak credential hygiene", r.Cred.Weak)
	}
	if r.Cred.Hosts != 0 {
		t.Fatalf("cred leak hosts=%d want 0 without log parser", r.Cred.Hosts)
	}
}
