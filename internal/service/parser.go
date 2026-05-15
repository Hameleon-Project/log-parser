package service

import (
	"context"
	"fmt"
	"log-parser/internal/model"
	"log/slog"
	"path/filepath"
	"time"
)

type ParserRepository interface {
	SaveParsedLog(ctx context.Context, filePath string, parsed *model.ParsedLog, links []model.InferredLink) (int, error)
	GetPortsByNodeID(ctx context.Context, nodeID int) ([]model.Port, error)
	GetLogByID(ctx context.Context, logID int) (*model.LogMeta, error)
	GetTopology(ctx context.Context, logID int) (*model.Topology, error)
	GetNodeByID(ctx context.Context, nodeID int) (*model.Node, error)
}

type ParserService struct {
	repo ParserRepository
}

func NewParserService(repo ParserRepository) *ParserService {
	return &ParserService{repo: repo}
}

func (s *ParserService) GetPortsByNode(ctx context.Context, nodeID int) ([]model.Port, error) {
	return s.repo.GetPortsByNodeID(ctx, nodeID)
}

func (s *ParserService) GetLogMeta(ctx context.Context, logID int) (*model.LogMeta, error) {
	return s.repo.GetLogByID(ctx, logID)
}

func (s *ParserService) GetTopology(ctx context.Context, logID int) (*model.Topology, error) {
	return s.repo.GetTopology(ctx, logID)
}

func (s *ParserService) GetNodeDetails(ctx context.Context, nodeID int) (*model.Node, error) {
	return s.repo.GetNodeByID(ctx, nodeID)
}

// ParseAndSave reads a log from data/ (plain export or .zip), parses it, persists topology (F-2, F-4, F-5).
func (s *ParserService) ParseAndSave(ctx context.Context, relPath string) (int, error) {
	start := time.Now()
	fullPath, err := ResolveUnderDataDir(".", relPath)
	if err != nil {
		return 0, err
	}

	raw, err := readLogBytes(fullPath)
	if err != nil {
		slog.Error("log_read_failed", "path", relPath, "err", err)
		return 0, fmt.Errorf("read log: %w", err)
	}

	parsed, err := parseIBDiagExport(raw)
	if err != nil {
		slog.Error("log_parse_failed", "path", relPath, "err", err, "duration_ms", time.Since(start).Milliseconds())
		return 0, err
	}

	links := InferPortLinks(parsed)
	storePath := filepath.ToSlash(filepath.Clean(relPath))
	logID, err := s.repo.SaveParsedLog(ctx, storePath, parsed, links)
	if err != nil {
		slog.Error("log_persist_failed", "path", relPath, "err", err, "duration_ms", time.Since(start).Milliseconds())
		return 0, fmt.Errorf("save parsed log: %w", err)
	}

	slog.Info("log_parsed",
		"path", relPath,
		"log_id", logID,
		"nodes", len(parsed.Nodes),
		"inferred_edges", len(links),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return logID, nil
}
