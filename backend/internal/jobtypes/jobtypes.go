package jobtypes

// JobDef is the shared job definition used by both the scheduler and worker pool.
// Lives here to avoid a circular import between those two packages.
type JobDef struct {
	ID          string
	Name        string
	CronExpr    string
	Timezone    string
	Command     string
	MaxRetries  int
	TimeoutSecs int
	DependsOn   []string
}
