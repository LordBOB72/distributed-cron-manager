package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/distributed-cron-manager/internal/alert"
	"github.com/yourorg/distributed-cron-manager/internal/api"
	"github.com/yourorg/distributed-cron-manager/internal/db"
	"github.com/yourorg/distributed-cron-manager/internal/election"
	"github.com/yourorg/distributed-cron-manager/internal/scheduler"
	"github.com/yourorg/distributed-cron-manager/internal/worker"
	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewProduction()
	defer log.Sync()

	nodeID := getEnv("NODE_ID", uuid.New().String())
	dbURL  := getEnv("DATABASE_URL", "postgres://cron:cron@localhost:5432/cron?sslmode=disable")
	etcdEP := strings.Split(getEnv("ETCD_ENDPOINTS", "localhost:2379"), ",")
	port   := getEnv("PORT", "8082")

	pool, err := db.Connect(dbURL)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}
	defer pool.Close()
	if err := db.Migrate(pool); err != nil {
		log.Fatal("migration", zap.Error(err))
	}

	// heartbeat this node into the DB
	go heartbeat(context.Background(), pool, nodeID, log)

	alerter := alert.New(getEnv("ALERT_WEBHOOK_URL", ""), log)
	wp := worker.NewPool(nodeID, pool, alerter, log)
	defer wp.Shutdown()

	sched := scheduler.New(pool, wp, log)

	elector, err := election.New(nodeID, etcdEP, log)
	if err != nil {
		log.Fatal("election init", zap.Error(err))
	}
	defer elector.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go elector.Campaign(ctx)
	go sched.OnLeaderChange(ctx, elector.IsLeaderCh())

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type"},
	}))

	h := api.NewHandler(pool, sched, wp, log)
	h.Register(r)

	srv := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: r}
	go func() {
		log.Info("listening", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
}

func heartbeat(ctx context.Context, pool *pgxpool.Pool, nodeID string, log *zap.Logger) {
	hostname, _ := os.Hostname()
	tick := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-tick.C:
			pool.Exec(ctx,
				`INSERT INTO nodes (id, hostname, last_seen) VALUES ($1, $2, NOW())
				 ON CONFLICT (id) DO UPDATE SET last_seen = NOW(), hostname = EXCLUDED.hostname`,
				nodeID, hostname)
		case <-ctx.Done():
			return
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
