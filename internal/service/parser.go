package service

import (
	"context"
	"errors"
	"log-parser/internal/model"
	"log-parser/internal/storage"
	"strings"
)

var (
	ErrEmptyLog      = errors.New("log message cannot be empty")
	ErrInvalidFormat = errors.New("invalid log format: expected 'LEVEL: Message'")
)

type ParserService struct {
	repo storage.LogRepository
}

func NewParserService(repo storage.LogRepository) *ParserService {
	return &ParserService{repo: repo}
}

func (s *ParserService) ParseAndSave(ctx context.Context, rawLog string) error {
	rawLog = strings.TrimSpace(rawLog)
	if rawLog == "" {
		return ErrEmptyLog
	}

	parts := strings.SplitN(rawLog, ":", 2)
	if len(parts) < 2 {
		return ErrInvalidFormat
	}

	level := strings.ToUpper(strings.TrimSpace(parts[0]))
	message := strings.TrimSpace(parts[1])

	if message == "" {
		return errors.New("log message content is empty")
	}

	entry := model.LogEntry{
		Level:   level,
		Message: message,
	}

	return s.repo.Insert(ctx, entry)
}

func (s *ParserService) GetLogs(ctx context.Context) ([]model.LogEntry, error) {
	return s.repo.GetAll(ctx)
}
