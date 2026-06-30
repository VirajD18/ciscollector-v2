package main

import (
	"context"
	"database/sql"
	"testing"

	dashboardsvc "github.com/VirajD18/ciscollector-v2/pkg/dashboard/service"
	mainserversvc "github.com/VirajD18/ciscollector-v2/pkg/mainserver/service"
	sqliterepo "github.com/VirajD18/ciscollector-v2/pkg/repository/sqlite"
)

func testAppFromSQLiteDB(t *testing.T, db *sql.DB, dbPath string, kshieldCfg string) *App {
	t.Helper()
	ctx := context.Background()
	repo := sqliterepo.FromDB(db)
	if err := repo.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	app := &App{
		DBFilePath:        dbPath,
		Repo:              repo,
		Svc:               mainserversvc.New(repo),
		DashboardSvc:      dashboardsvc.NewWithConfig(repo, kshieldCfg),
		KshieldConfigPath: kshieldCfg,
		WSHub:             NewHub(),
	}
	return app
}
