package worker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/distributed-cron-manager/internal/alert"
	"go.uber.org/zap"
)

type JobDef interface {
	GetID() string
	GetName() string
	GetCommand() string
	GetMaxRetries() int
	GetTimeoutSecs() int
	GetDependsOn() []string
}

type Pool struct {
	nodeID  string
	pool    *pgxpool.Pool
	alerter *alert.Alerter
	log     *zap.Logger

	queue    chan jobTask
	wg       sync.WaitGroup
	runCount atomic.Int64
}

type jobTask struct {
	job JobDef
	ctx context.Context
}

func NewPool(nodeID string, pool *pgxpool.Pool, alerter *alert.Alerter, log *zap.Logger) *Pool {
	concurrency := runtime.NumCPU()
	p := &Pool{
		nodeID:  nodeID,
		pool:    pool,
		alerter: alerter,
		log:     log,
		queue:   make(chan jobTask, 256),
	}
	for i := 0; i < concurrency; i++ {
		p.wg.Add(1)
		go p.consume()
	}
	return p
}

func (p *Pool) Enqueue(ctx context.Context, job interface{ GetID() string; GetName() string; GetCommand() string; GetMaxRetries() int; GetTimeoutSecs() int; GetDependsOn() []string }) {
	select {
	case p.queue <- jobTask{job: job, ctx: ctx}:
	default:
		p.log.Warn("worker queue full, dropping job", zap.String("job", job.GetName()))
	}
}

func (p *Pool) RunCount() int64 { return p.runCount.Load() }

func (p *Pool) Shutdown() {
	close(p.queue)
	p.wg.Wait()
}

func (p *Pool) consume() {
	defer p.wg.Done()
	for task := range p.queue {
		p.run(task)
	}
}

func (p *Pool) run(task jobTask) {
	job := task.job
	runID := uuid.New().String()
	scheduledAt := time.Now()

	// check dependency completion before running
	if len(job.GetDependsOn()) > 0 {
		if err := p.checkDeps(task.ctx, job.GetDependsOn()); err != nil {
			p.log.Warn("dependency not met, skipping", zap.String("job", job.GetName()), zap.Error(err))
			return
		}
	}

	p.upsertRun(task.ctx, runID, job.GetID(), "running", scheduledAt, 1, "", 0)
	p.runCount.Add(1)

	var lastErr error
	for attempt := 1; attempt <= max(job.GetMaxRetries(), 1); attempt++ {
		if attempt > 1 {
			// exponential backoff between retries
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			p.upsertRun(task.ctx, runID, job.GetID(), "running", scheduledAt, attempt, "", 0)
		}

		output, exitCode, err := p.execute(task.ctx, job.GetCommand(), job.GetTimeoutSecs())
		if err == nil {
			p.upsertRun(task.ctx, runID, job.GetID(), "success", scheduledAt, attempt, output, exitCode)
			p.log.Info("job success", zap.String("job", job.GetName()), zap.Int("attempt", attempt))
			return
		}
		lastErr = err
		p.log.Warn("job attempt failed", zap.String("job", job.GetName()), zap.Int("attempt", attempt), zap.Error(err))
		p.upsertRun(task.ctx, runID, job.GetID(), "failed", scheduledAt, attempt, output, exitCode)
	}

	// exhausted retries — dead letter
	p.upsertRun(task.ctx, runID, job.GetID(), "dead_letter", scheduledAt, job.GetMaxRetries(), lastErr.Error(), 1)
	p.alerter.Send(task.ctx, fmt.Sprintf("Job %q dead-lettered after %d attempts: %v", job.GetName(), job.GetMaxRetries(), lastErr))
}

func (p *Pool) execute(ctx context.Context, command string, timeoutSecs int) (string, int, error) {
	timeout := time.Duration(timeoutSecs) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := buf.String()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return output, exitCode, fmt.Errorf("timed out after %ds", timeoutSecs)
	}
	return output, exitCode, err
}

func (p *Pool) upsertRun(ctx context.Context, runID, jobID, status string, scheduledAt time.Time, attempt int, output string, exitCode int) {
	var completedAt interface{}
	if status == "success" || status == "failed" || status == "dead_letter" || status == "timed_out" {
		completedAt = time.Now()
	}

	_, err := p.pool.Exec(ctx,
		`INSERT INTO job_runs (id, job_id, node_id, status, attempt, exit_code, log_output, scheduled_at, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)
		 ON CONFLICT (id) DO UPDATE
		   SET status = EXCLUDED.status,
		       attempt = EXCLUDED.attempt,
		       exit_code = EXCLUDED.exit_code,
		       log_output = EXCLUDED.log_output,
		       completed_at = EXCLUDED.completed_at`,
		runID, jobID, p.nodeID, status, attempt, exitCode, output, scheduledAt, completedAt,
	)
	if err != nil {
		p.log.Error("upsert run failed", zap.Error(err))
	}
}

func (p *Pool) checkDeps(ctx context.Context, depIDs []string) error {
	for _, depID := range depIDs {
		var status string
		err := p.pool.QueryRow(ctx,
			`SELECT status FROM job_runs WHERE job_id = $1 ORDER BY created_at DESC LIMIT 1`,
			depID).Scan(&status)
		if err != nil || status != "success" {
			return fmt.Errorf("dependency %s not yet succeeded (status: %s)", depID, status)
		}
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
