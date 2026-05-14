package service

import (
	"context"
	"errors"
	"fmt"
	"log-parser/internal/model"
	"log-parser/internal/storage"
	"regexp"
	"strings"
)

var (
	ErrEmptyLog      = errors.New("log message cannot be empty")
	ErrInvalidFormat = errors.New("invalid log format: expected 'LEVEL: Message'")
	nodeRegex        = regexp.MustCompile(`Node:\s+(\w+)\s+Type:\s+(\w+)`)
	linkRegex        = regexp.MustCompile(`Link:\s+(\w+):(\w+)\s+<->\s+(\w+):(\w+)`)
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

func (s *ParserService) GetLogs(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.GetAll(ctx, filter)
}

func (s *ParserService) ParseTopology(ctx context.Context, logID int, rawData string) error {
	// мапа для кэширования ID узлов и портов, чтобы не лезть в базу лишний раз
	nodeCache := make(map[string]int)
	portCache := make(map[string]int)

	// парсим узлы
	nodesMatches := nodeRegex.FindAllStringSubmatch(rawData, -1)
	for _, match := range nodesMatches {
		name := match[1]
		nodeType := match[2]

		node := model.Node{
			Name: name,
			Type: nodeType,
		}

		nodeID, err := s.repo.SaveNode(ctx, logID, node)
		if err != nil {
			return fmt.Errorf("failed to save node %s: %w", name, err)
		}
		nodeCache[name] = nodeID
	}

	// парсим связи и создаем порты
	linksMatches := linkRegex.FindAllStringSubmatch(rawData, -1)
	for _, match := range linksMatches {
		nodeAName, portAName := match[1], match[2]
		nodeBName, portBName := match[3], match[4]

		// порт А
		portAID, err := s.getOrCreatePort(ctx, nodeCache[nodeAName], portAName, portCache)
		if err != nil {
			return err
		}

		// порт Б
		portBID, err := s.getOrCreatePort(ctx, nodeCache[nodeBName], portBName, portCache)
		if err != nil {
			return err
		}

		// создаем связь в таблице links
		err = s.repo.CreateLink(ctx, portAID, portBID)
		if err != nil {
			return fmt.Errorf("failed to create link: %w", err)
		}
	}

	return nil
}

// функция, чтобы не дублировать порты
func (s *ParserService) getOrCreatePort(ctx context.Context, nodeID int, portName string, cache map[string]int) (int, error) {
	key := fmt.Sprintf("%d:%s", nodeID, portName)
	if id, ok := cache[key]; ok {
		return id, nil
	}

	portID, err := s.repo.SavePort(ctx, model.Port{
		NodeID: nodeID,
		Name:   portName,
	})
	if err != nil {
		return 0, err
	}

	cache[key] = portID
	return portID, nil
}
