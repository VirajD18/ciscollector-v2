PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;

CREATE TABLE IF NOT EXISTS metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id               TEXT PRIMARY KEY,
    started_at       TEXT NOT NULL,
    finished_at      TEXT NOT NULL,
    trigger          TEXT NOT NULL DEFAULT 'cron',
    runner_name      TEXT NOT NULL DEFAULT '',
    target_type      TEXT NOT NULL DEFAULT 'postgres',
    target_id        TEXT NOT NULL,
    target_host      TEXT NOT NULL,
    target_port      TEXT NOT NULL DEFAULT '',
    target_db        TEXT NOT NULL DEFAULT '',
    run_status       TEXT NOT NULL,
    features_run     TEXT,
    overall_score    REAL,
    total_pass       INTEGER DEFAULT 0,
    total_fail       INTEGER DEFAULT 0,
    -- report_json      BLOB NOT NULL
    report_json      JSON NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_runs_target_id ON runs(target_id, started_at DESC);
