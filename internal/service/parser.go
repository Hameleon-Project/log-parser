package service

import (
	"context"
	"errors"
	"log-parser/internal/model"
	"log-parser/internal/storage"
	"strings"
)

type ParserService struct {
	repo storage.LogRepository
}

func NewParserService(repo storage.LogRepository) *ParserService {
	return &ParserService{repo: repo}
}

func (s *ParserService) ParseAndSave(ctx context.Context, rawLog string) error {
	parts := strings.SplitN(rawLog, ":", 2)
	if len(parts) < 2 {
		return errors.New("invalid log format: expected 'LEVEL: Message'")
	}

	entry := model.LogEntry{
		Level:   strings.TrimSpace(parts[0]),
		Message: strings.TrimSpace(parts[1]),
	}

	return s.repo.Insert(ctx, entry)
}
