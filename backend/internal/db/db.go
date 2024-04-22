package db

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed ../../migrations/*.sql
var migrationsFS embed.FS

func Connect(dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return pool, nil
}

func Migrate(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrationsFS)
	goose.SetDialect("postgres")
	return goose.Up(db, "migrations")
}
