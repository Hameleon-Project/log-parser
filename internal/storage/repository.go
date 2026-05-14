package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log-parser/internal/model"
)

type LogRepository interface {
	Insert(ctx context.Context, entry model.LogEntry) error
	GetAll(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error)
	SaveNode(ctx context.Context, logID int, n model.Node) (int, error)
	SavePort(ctx context.Context, p model.Port) (int, error)
	CreateLink(ctx context.Context, fromID, toID int) error
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

func (r *PostgresRepo) GetAll(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error) {
	// базовый запрос
	query := `SELECT id, level, message, created_at FROM logs WHERE 1=1`
	args := []interface{}{}
	argID := 1

	// добавляем фильтр по уровню
	if filter.Level != "" {
		query += fmt.Sprintf(" AND level = $%d", argID)
		args = append(args, filter.Level)
		argID++
	}

	// добавляем сортировку и пагинацию
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argID, argID+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *PostgresRepo) SaveNode(ctx context.Context, logID int, n model.Node) (int, error) {
	var id int
	query := `INSERT INTO nodes (log_id, name, type) VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, logID, n.Name, n.Type).Scan(&id)
	return id, err
}

func (r *PostgresRepo) SavePort(ctx context.Context, p model.Port) (int, error) {
	var id int
	query := `INSERT INTO ports (node_id, name) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, p.NodeID, p.Name).Scan(&id)
	return id, err
}

func (r *PostgresRepo) CreateLink(ctx context.Context, fromID, toID int) error {
	query := `INSERT INTO links (from_port_id, to_port_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, fromID, toID)
	return err
}
