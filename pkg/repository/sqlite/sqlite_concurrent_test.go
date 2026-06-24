package sqlite_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/klouddb/klouddbshield/pkg/reportstore"
	sqliterepo "github.com/klouddb/klouddbshield/pkg/repository/sqlite"
)

func TestPersistScanResultConcurrent(t *testing.T) {
	tests := []struct {
		name    string
		writers int
	}{
		{name: "fifty concurrent writers", writers: 50},
		{name: "hundred concurrent writers", writers: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path := filepath.Join(t.TempDir(), "test.db")
			repo, err := sqliterepo.Open(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			defer repo.Close()

			started := time.Now().UTC()
			var wg sync.WaitGroup
			errCh := make(chan error, tt.writers)

			for i := 0; i < tt.writers; i++ {
				i := i
				wg.Add(1)
				go func() {
					defer wg.Done()
					host := fmt.Sprintf("pg-%03d", i)
					pg := reportstore.PostgresFromTarget(host, "5432", "shielddb")
					_, err := repo.PersistScanResult(ctx, map[string]interface{}{
						"Postgres Report": map[string]interface{}{
							"result": []interface{}{
								map[string]interface{}{"Status": "Pass", "Control": "1.1"},
							},
						},
					}, reportstore.ScanResultMeta{
						RunMeta: reportstore.RunMeta{
							Trigger:    "cron",
							RunnerName: "ciscollector",
							Postgres:   pg,
							StartedAt:  started,
							FinishedAt: started,
							RunStatus:  "success",
						},
						NodeID:   fmt.Sprintf("node-%d", i),
						Hostname: host,
					})
					if err != nil {
						errCh <- err
					}
				}()
			}
			wg.Wait()
			close(errCh)
			for err := range errCh {
				t.Errorf("PersistScanResult: %v", err)
			}

			runs, err := repo.GetRuns(ctx, tt.writers+10)
			if err != nil {
				t.Fatal(err)
			}
			if len(runs) != tt.writers {
				t.Fatalf("runs = %d, want %d", len(runs), tt.writers)
			}
		})
	}
}
