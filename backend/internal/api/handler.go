package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/distributed-cron-manager/internal/scheduler"
	"github.com/yourorg/distributed-cron-manager/internal/worker"
	"go.uber.org/zap"
)

type Handler struct {
	pool      *pgxpool.Pool
	sched     *scheduler.Scheduler
	workers   *worker.Pool
	log       *zap.Logger
	wsHub     *Hub
}

func NewHandler(pool *pgxpool.Pool, sched *scheduler.Scheduler, workers *worker.Pool, log *zap.Logger) *Handler {
	hub := newHub()
	go hub.run()
	return &Handler{pool: pool, sched: sched, workers: workers, log: log, wsHub: hub}
}

func (h *Handler) Register(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	v1.GET("/jobs", h.listJobs)
	v1.POST("/jobs", h.createJob)
	v1.GET("/jobs/:id", h.getJob)
	v1.PUT("/jobs/:id", h.updateJob)
	v1.DELETE("/jobs/:id", h.deleteJob)
	v1.POST("/jobs/:id/trigger", h.triggerJob)
	v1.POST("/jobs/:id/pause", h.pauseJob)
	v1.POST("/jobs/:id/resume", h.resumeJob)
	v1.GET("/jobs/:id/runs", h.getJobRuns)

	v1.GET("/runs", h.listRuns)
	v1.GET("/nodes", h.listNodes)

	r.GET("/ws", h.handleWS)
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
}

type jobInput struct {
	Name        string   `json:"name" binding:"required"`
	CronExpr    string   `json:"cron_expr" binding:"required"`
	Timezone    string   `json:"timezone"`
	Command     string   `json:"command" binding:"required"`
	MaxRetries  int      `json:"max_retries"`
	TimeoutSecs int      `json:"timeout_secs"`
	Tags        []string `json:"tags"`
	DependsOn   []string `json:"depends_on"`
}

func (h *Handler) listJobs(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, name, cron_expr, timezone, command, max_retries, timeout_secs, tags, depends_on, enabled, paused, created_at, updated_at
		 FROM jobs ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var jobs []map[string]interface{}
	for rows.Next() {
		var j struct {
			ID, Name, CronExpr, Timezone, Command string
			MaxRetries, TimeoutSecs                int
			Tags, DependsOn                        []string
			Enabled, Paused                        bool
			CreatedAt, UpdatedAt                   time.Time
		}
		rows.Scan(&j.ID, &j.Name, &j.CronExpr, &j.Timezone, &j.Command,
			&j.MaxRetries, &j.TimeoutSecs, &j.Tags, &j.DependsOn,
			&j.Enabled, &j.Paused, &j.CreatedAt, &j.UpdatedAt)
		jobs = append(jobs, map[string]interface{}{
			"id": j.ID, "name": j.Name, "cron_expr": j.CronExpr, "timezone": j.Timezone,
			"command": j.Command, "max_retries": j.MaxRetries, "timeout_secs": j.TimeoutSecs,
			"tags": j.Tags, "depends_on": j.DependsOn, "enabled": j.Enabled, "paused": j.Paused,
			"created_at": j.CreatedAt, "updated_at": j.UpdatedAt,
		})
	}
	if jobs == nil {
		jobs = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) createJob(c *gin.Context) {
	var inp jobInput
	if err := c.ShouldBindJSON(&inp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if inp.Timezone == "" {
		inp.Timezone = "UTC"
	}
	if inp.TimeoutSecs == 0 {
		inp.TimeoutSecs = 300
	}
	if inp.Tags == nil {
		inp.Tags = []string{}
	}

	var id string
	err := h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO jobs (name, cron_expr, timezone, command, max_retries, timeout_secs, tags, depends_on)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		inp.Name, inp.CronExpr, inp.Timezone, inp.Command,
		inp.MaxRetries, inp.TimeoutSecs, inp.Tags, inp.DependsOn,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go h.sched.Reload(c.Request.Context())
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) getJob(c *gin.Context) {
	id := c.Param("id")
	var j map[string]interface{}
	row := h.pool.QueryRow(c.Request.Context(),
		`SELECT id, name, cron_expr, timezone, command, max_retries, timeout_secs, tags, depends_on, enabled, paused, created_at
		 FROM jobs WHERE id = $1`, id)
	var jj struct {
		ID, Name, CronExpr, Timezone, Command string
		MaxRetries, TimeoutSecs                int
		Tags, DependsOn                        []string
		Enabled, Paused                        bool
		CreatedAt                              time.Time
	}
	if err := row.Scan(&jj.ID, &jj.Name, &jj.CronExpr, &jj.Timezone, &jj.Command,
		&jj.MaxRetries, &jj.TimeoutSecs, &jj.Tags, &jj.DependsOn,
		&jj.Enabled, &jj.Paused, &jj.CreatedAt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	j = map[string]interface{}{
		"id": jj.ID, "name": jj.Name, "cron_expr": jj.CronExpr, "timezone": jj.Timezone,
		"command": jj.Command, "max_retries": jj.MaxRetries, "timeout_secs": jj.TimeoutSecs,
		"tags": jj.Tags, "depends_on": jj.DependsOn, "enabled": jj.Enabled,
		"paused": jj.Paused, "created_at": jj.CreatedAt,
	}
	c.JSON(http.StatusOK, j)
}

func (h *Handler) updateJob(c *gin.Context) {
	var inp jobInput
	if err := c.ShouldBindJSON(&inp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.pool.Exec(c.Request.Context(),
		`UPDATE jobs SET name=$1, cron_expr=$2, timezone=$3, command=$4,
		 max_retries=$5, timeout_secs=$6, tags=$7, depends_on=$8, updated_at=NOW()
		 WHERE id=$9`,
		inp.Name, inp.CronExpr, inp.Timezone, inp.Command,
		inp.MaxRetries, inp.TimeoutSecs, inp.Tags, inp.DependsOn, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	go h.sched.Reload(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func (h *Handler) deleteJob(c *gin.Context) {
	h.pool.Exec(c.Request.Context(), "DELETE FROM jobs WHERE id=$1", c.Param("id"))
	go h.sched.Reload(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func (h *Handler) triggerJob(c *gin.Context) {
	if err := h.sched.TriggerNow(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"status": "triggered"})
}

func (h *Handler) pauseJob(c *gin.Context) {
	h.pool.Exec(c.Request.Context(), "UPDATE jobs SET paused=true WHERE id=$1", c.Param("id"))
	go h.sched.Reload(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func (h *Handler) resumeJob(c *gin.Context) {
	h.pool.Exec(c.Request.Context(), "UPDATE jobs SET paused=false WHERE id=$1", c.Param("id"))
	go h.sched.Reload(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func (h *Handler) getJobRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, job_id, node_id, status, attempt, exit_code, log_output, scheduled_at, started_at, completed_at
		 FROM job_runs WHERE job_id=$1 ORDER BY created_at DESC LIMIT $2`, c.Param("id"), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var runs []map[string]interface{}
	for rows.Next() {
		var r struct {
			ID, JobID, NodeID, Status, LogOutput string
			Attempt, ExitCode                    int
			ScheduledAt                          time.Time
			StartedAt, CompletedAt               *time.Time
		}
		rows.Scan(&r.ID, &r.JobID, &r.NodeID, &r.Status, &r.Attempt, &r.ExitCode,
			&r.LogOutput, &r.ScheduledAt, &r.StartedAt, &r.CompletedAt)
		runs = append(runs, map[string]interface{}{
			"id": r.ID, "job_id": r.JobID, "node_id": r.NodeID, "status": r.Status,
			"attempt": r.Attempt, "exit_code": r.ExitCode, "log_output": r.LogOutput,
			"scheduled_at": r.ScheduledAt, "started_at": r.StartedAt, "completed_at": r.CompletedAt,
		})
	}
	if runs == nil {
		runs = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handler) listRuns(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT jr.id, jr.job_id, j.name, jr.node_id, jr.status, jr.attempt, jr.exit_code, jr.scheduled_at, jr.started_at, jr.completed_at
		 FROM job_runs jr JOIN jobs j ON j.id = jr.job_id
		 ORDER BY jr.created_at DESC LIMIT $1`, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var runs []map[string]interface{}
	for rows.Next() {
		var r struct {
			ID, JobID, JobName, NodeID, Status string
			Attempt, ExitCode                  int
			ScheduledAt                        time.Time
			StartedAt, CompletedAt             *time.Time
		}
		rows.Scan(&r.ID, &r.JobID, &r.JobName, &r.NodeID, &r.Status,
			&r.Attempt, &r.ExitCode, &r.ScheduledAt, &r.StartedAt, &r.CompletedAt)
		runs = append(runs, map[string]interface{}{
			"id": r.ID, "job_id": r.JobID, "job_name": r.JobName, "node_id": r.NodeID,
			"status": r.Status, "attempt": r.Attempt, "exit_code": r.ExitCode,
			"scheduled_at": r.ScheduledAt, "started_at": r.StartedAt, "completed_at": r.CompletedAt,
		})
	}
	if runs == nil {
		runs = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handler) listNodes(c *gin.Context) {
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id, hostname, is_leader, last_seen, run_count FROM nodes ORDER BY is_leader DESC, id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var nodes []map[string]interface{}
	for rows.Next() {
		var n struct {
			ID, Hostname       string
			IsLeader           bool
			LastSeen           time.Time
			RunCount           int
		}
		rows.Scan(&n.ID, &n.Hostname, &n.IsLeader, &n.LastSeen, &n.RunCount)
		nodes = append(nodes, map[string]interface{}{
			"id": n.ID, "hostname": n.Hostname, "is_leader": n.IsLeader,
			"last_seen": n.LastSeen, "run_count": n.RunCount,
		})
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, nodes)
}
