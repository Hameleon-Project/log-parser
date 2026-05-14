package service

import (
	"context"
	"log-parser/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Создаем мок для репозитория
type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) Insert(ctx context.Context, entry model.LogEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

func (m *MockRepo) GetAll(ctx context.Context, filter model.LogFilter) ([]model.LogEntry, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]model.LogEntry), args.Error(1)
}

func (m *MockRepo) SaveNode(ctx context.Context, logID int, n model.Node) (int, error) {
	args := m.Called(ctx, logID, n)
	return args.Int(0), args.Error(1)
}

func (m *MockRepo) SavePort(ctx context.Context, p model.Port) (int, error) {
	args := m.Called(ctx, p)
	return args.Int(0), args.Error(1)
}

func (m *MockRepo) CreateLink(ctx context.Context, fromID, toID int) error {
	args := m.Called(ctx, fromID, toID)
	return args.Error(0)
}

func TestParseAndSave(t *testing.T) {
	repo := new(MockRepo)
	svc := NewParserService(repo)

	t.Run("success parse", func(t *testing.T) {
		raw := "INFO: Everything is fine"

		repo.On("Insert", mock.Anything, mock.MatchedBy(func(e model.LogEntry) bool {
			return e.Level == "INFO" && e.Message == "Everything is fine"
		})).Return(nil).Once()

		err := svc.ParseAndSave(context.Background(), raw)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("invalid format", func(t *testing.T) {
		raw := "Just some text without colon"
		err := svc.ParseAndSave(context.Background(), raw)

		assert.ErrorIs(t, err, ErrInvalidFormat)
	})
}
