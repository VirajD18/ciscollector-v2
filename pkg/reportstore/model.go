package reportstore

import (
	"time"

	"github.com/klouddb/klouddbshield/pkg/postgresdb"
)

// RunMeta describes one persisted scan (cron tick or manual --json).
type RunMeta struct {
	Trigger     string // cron | manual
	RunnerName  string
	FeaturesRun []string
	Postgres    *postgresdb.Postgres
	StartedAt   time.Time
	FinishedAt  time.Time
	RunStatus   string // success | partial | failed
}

// RunRow is a stored run with optional denormalized overview fields.
type RunRow struct {
	ID           string
	StartedAt    time.Time
	FinishedAt   time.Time
	Trigger      string
	RunnerName   string
	TargetType   string
	TargetID     string
	TargetHost   string
	TargetPort   string
	TargetDB     string
	RunStatus    string
	FeaturesRun  []string
	OverallScore float64
	TotalPass    int
	TotalFail    int
	Report       map[string]interface{}
	PiiReport    map[string]interface{}
	PiiScannedAt time.Time
}
