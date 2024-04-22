package worker

import "github.com/yourorg/distributed-cron-manager/internal/jobtypes"

// schedulerJob wraps jobtypes.JobDef to satisfy the JobDef interface expected by Pool.Enqueue.
type schedulerJob struct {
	jobtypes.JobDef
}

func (s schedulerJob) GetID() string          { return s.ID }
func (s schedulerJob) GetName() string        { return s.Name }
func (s schedulerJob) GetCommand() string     { return s.Command }
func (s schedulerJob) GetMaxRetries() int     { return s.MaxRetries }
func (s schedulerJob) GetTimeoutSecs() int    { return s.TimeoutSecs }
func (s schedulerJob) GetDependsOn() []string { return s.DependsOn }

// Wrap converts a jobtypes.JobDef into the interface Pool.Enqueue expects.
func Wrap(j jobtypes.JobDef) schedulerJob {
	return schedulerJob{j}
}
