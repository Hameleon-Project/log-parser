package storage

import (
	"context"
	"database/sql"
	"log-parser/internal/model"
)

type LogRepository interface {
	Insert(ctx context.Context, entry model.LogEntry) error
}

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(db *sql.DB) *PostgresRepo {
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Insert(ctx context.Context, e model.LogEntry) error {
	query := `INSERT INTO logs (level, message) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, e.Level, e.Message)
	return err
}
