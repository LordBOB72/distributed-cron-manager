package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"github.com/yourorg/distributed-cron-manager/internal/jobtypes"
	"github.com/yourorg/distributed-cron-manager/internal/worker"
	"go.uber.org/zap"
)

// Scheduler only runs on the leader node. On leader loss it stops; on gain it reloads from DB.
type Scheduler struct {
	pool   *pgxpool.Pool
	worker *worker.Pool
	log    *zap.Logger

	mu       sync.Mutex
	cronInst *cron.Cron
	running  bool
}

func New(pool *pgxpool.Pool, wp *worker.Pool, log *zap.Logger) *Scheduler {
	return &Scheduler{pool: pool, worker: wp, log: log}
}

func (s *Scheduler) OnLeaderChange(ctx context.Context, isLeader <-chan bool) {
	for {
		select {
		case leader, ok := <-isLeader:
			if !ok {
				return
			}
			if leader {
				s.start(ctx)
			} else {
				s.stop()
			}
		case <-ctx.Done():
			s.stop()
			return
		}
	}
}

func (s *Scheduler) start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.log.Info("scheduler starting")
	s.cronInst = cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))

	jobs, err := s.loadJobs(ctx)
	if err != nil {
		s.log.Error("load jobs failed", zap.Error(err))
		return
	}

	for _, j := range jobs {
		j := j // capture loop var
		loc, err := time.LoadLocation(j.Timezone)
		if err != nil {
			loc = time.UTC
		}
		expr := fmt.Sprintf("TZ=%s %s", loc.String(), j.CronExpr)
		if _, err := s.cronInst.AddFunc(expr, func() {
			s.worker.Enqueue(ctx, worker.Wrap(j))
		}); err != nil {
			s.log.Error("schedule job failed", zap.String("job", j.Name), zap.Error(err))
		}
	}

	s.cronInst.Start()
	s.running = true
	s.log.Info("scheduler running", zap.Int("jobs", len(jobs)))
}

func (s *Scheduler) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	stopCtx := s.cronInst.Stop()
	<-stopCtx.Done()
	s.running = false
	s.log.Info("scheduler stopped")
}

// TriggerNow bypasses the cron schedule and executes the job immediately.
func (s *Scheduler) TriggerNow(ctx context.Context, jobID string) error {
	j, err := s.getJob(ctx, jobID)
	if err != nil {
		return err
	}
	s.worker.Enqueue(ctx, worker.Wrap(j))
	return nil
}

// Reload restarts the cron engine — called after any job create/update/delete.
func (s *Scheduler) Reload(ctx context.Context) {
	s.stop()
	s.start(ctx)
}

func (s *Scheduler) loadJobs(ctx context.Context) ([]jobtypes.JobDef, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, cron_expr, timezone, command, max_retries, timeout_secs, depends_on
		 FROM jobs WHERE enabled = true AND paused = false`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []jobtypes.JobDef
	for rows.Next() {
		var j jobtypes.JobDef
		if err := rows.Scan(&j.ID, &j.Name, &j.CronExpr, &j.Timezone,
			&j.Command, &j.MaxRetries, &j.TimeoutSecs, &j.DependsOn); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (s *Scheduler) getJob(ctx context.Context, id string) (jobtypes.JobDef, error) {
	var j jobtypes.JobDef
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, cron_expr, timezone, command, max_retries, timeout_secs, depends_on
		 FROM jobs WHERE id = $1`, id).
		Scan(&j.ID, &j.Name, &j.CronExpr, &j.Timezone,
			&j.Command, &j.MaxRetries, &j.TimeoutSecs, &j.DependsOn)
	return j, err
}
