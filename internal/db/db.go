// Package db содержит функцию подключения к PostgreSQL.
package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool создаёт пул соединений к psql.
// DSN берётся из переменной окружения POSTGRES_DSN.
func NewPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("env POSTGRES_DSN is not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка при подключении к постгрес: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("постгрес не отвечает. Ошибка: %w", err)
	}

	return pool, nil
}
