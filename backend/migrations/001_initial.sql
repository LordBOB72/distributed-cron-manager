-- +goose Up
CREATE TABLE IF NOT EXISTS jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    cron_expr       TEXT NOT NULL,
    timezone        TEXT NOT NULL DEFAULT 'UTC',
    command         TEXT NOT NULL,
    max_retries     INT NOT NULL DEFAULT 3,
    timeout_secs    INT NOT NULL DEFAULT 300,
    tags            TEXT[] NOT NULL DEFAULT '{}',
    depends_on      UUID[],
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    paused          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS job_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    node_id         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','running','success','failed','dead_letter','timed_out')),
    attempt         INT NOT NULL DEFAULT 1,
    exit_code       INT,
    log_output      TEXT,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT PRIMARY KEY,
    hostname    TEXT NOT NULL,
    is_leader   BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    run_count   INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS job_runs_job_id  ON job_runs(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS job_runs_status  ON job_runs(status);

-- +goose Down
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS jobs;
