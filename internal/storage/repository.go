package storage

import (
	"context"
	"database/sql"
	"log-parser/internal/model"
)

type LogRepository interface {
	Insert(ctx context.Context, entry model.LogEntry) error
	GetAll(ctx context.Context) ([]model.LogEntry, error)
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

func (r *PostgresRepo) GetAll(ctx context.Context) ([]model.LogEntry, error) {
	query := `select id, level, message, created_at from logs order by created_at desc`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.LogEntry
	for rows.Next() {
		var e model.LogEntry
		if err := rows.Scan(&e.ID, &e.Level, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
