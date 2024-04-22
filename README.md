# distributed-cron-manager

Fault-tolerant distributed job scheduler. Exactly-once execution across nodes via etcd leader election and distributed locking. Full web UI with real-time run status over WebSocket.

## What it does

Nodes elect a leader via etcd. Only the leader schedules jobs. All nodes execute work pulled from a shared queue. If the leader dies, another node takes over within ~15 seconds and resumes scheduling from where it left off.

Supports job dependency chains, timezone-aware cron expressions, exponential backoff retry, dead-letter after N failures, and per-job execution logs in the UI.

## Stack

- **Backend**: Go + Gin, etcd (leader election), PostgreSQL (job state + run history)
- **Frontend**: React + TypeScript + Vite + Tailwind
- **Real-time**: WebSocket push for live run status

## Running locally

```bash
docker-compose up postgres etcd -d

cd backend && go run ./cmd/server

cd frontend && npm install && npm run dev
```

Or everything at once:
```bash
docker-compose up --build
```

UI at http://localhost:5173.

## Multi-node setup

Run multiple backend instances pointing at the same DB and etcd cluster. Each gets a unique `NODE_ID`. They'll elect a leader automatically and you'll see all of them in the Nodes view.

```bash
NODE_ID=node-1 PORT=8082 go run ./cmd/server
NODE_ID=node-2 PORT=8083 go run ./cmd/server
```

## Environment variables

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8082` | |
| `DATABASE_URL` | `postgres://cron:cron@localhost:5432/cron?sslmode=disable` | |
| `ETCD_ENDPOINTS` | `localhost:2379` | Comma-separated |
| `NODE_ID` | random UUID | Set explicitly in multi-node deployments |
| `ALERT_WEBHOOK_URL` | `` | Slack-compatible webhook for dead-letter alerts |

## Cron expression format

Standard 5-field cron (`MIN HOUR DOM MON DOW`). Timezone set per job. Uses `robfig/cron` under the hood — supports `@hourly`, `@daily` etc.

## Dependency chains

Set `depends_on` to a list of job IDs. The job will only run if the most recent run of each dependency completed with `success`. Otherwise it skips and logs a warning.
