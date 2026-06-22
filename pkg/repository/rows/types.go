package rows

import "time"

// CollectorNodeRow is a fleet node status record.
type CollectorNodeRow struct {
	NodeID        string
	Hostname      string
	IP            string
	LastSeenAt    time.Time
	CronRunning   bool
	ScheduledJobs int
	LastRunAt     *time.Time
	LastError     string
}

// CollectorRunRow is one collector execution record.
type CollectorRunRow struct {
	ID         int64
	Trigger    string
	StartedAt  time.Time
	FinishedAt *time.Time
	Features   []string
	Success    bool
	Error      string
}

// CollectorActivityRow is a collector activity event.
type CollectorActivityRow struct {
	Kind      string
	Message   string
	Level     string
	CreatedAt time.Time
}

// CollectorLogRow is a collector log line.
type CollectorLogRow struct {
	Level     string
	Message   string
	CreatedAt time.Time
}
