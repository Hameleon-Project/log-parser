package service

import (
	"context"
	"os"
	"testing"

	"log-parser/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepo struct {
	mock.Mock
}

func (m *MockRepo) SaveParsedLog(ctx context.Context, filePath string, parsed *model.ParsedLog, links []model.InferredLink) (int, error) {
	args := m.Called(ctx, filePath, parsed, links)
	return args.Int(0), args.Error(1)
}

func (m *MockRepo) GetPortsByNodeID(ctx context.Context, nodeID int) ([]model.Port, error) {
	args := m.Called(ctx, nodeID)
	return args.Get(0).([]model.Port), args.Error(1)
}

func (m *MockRepo) GetLogByID(ctx context.Context, logID int) (*model.LogMeta, error) {
	args := m.Called(ctx, logID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.LogMeta), args.Error(1)
}

func (m *MockRepo) GetTopology(ctx context.Context, logID int) (*model.Topology, error) {
	args := m.Called(ctx, logID)
	return args.Get(0).(*model.Topology), args.Error(1)
}

func (m *MockRepo) GetNodeByID(ctx context.Context, nodeID int) (*model.Node, error) {
	args := m.Called(ctx, nodeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Node), args.Error(1)
}

func TestParseIBDiagExport_SampleFile(t *testing.T) {
	b, err := os.ReadFile("data/ibdiagnet2.db_csv")
	if err != nil {
		t.Skip("data/ibdiagnet2.db_csv not available:", err)
	}
	parsed, err := parseIBDiagExport(b)
	assert.NoError(t, err)
	assert.NotEmpty(t, parsed.Nodes)
	assert.NotEmpty(t, parsed.Infos)
	var totalPorts int
	for _, n := range parsed.Nodes {
		totalPorts += len(n.Ports)
	}
	assert.Positive(t, totalPorts)
}

func TestParseIBDiagExport_Invalid(t *testing.T) {
	_, err := parseIBDiagExport([]byte("not a diag file"))
	assert.ErrorIs(t, err, ErrParseLog)
}

func TestParseAndSave_IntegrationMock(t *testing.T) {
	repo := new(MockRepo)
	svc := NewParserService(repo)

	b, err := os.ReadFile("data/ibdiagnet2.db_csv")
	if err != nil {
		t.Skip("sample data missing")
	}
	parsed, err := parseIBDiagExport(b)
	assert.NoError(t, err)

	repo.On("SaveParsedLog", mock.Anything, "data/ibdiagnet2.db_csv", mock.Anything, mock.Anything).Return(42, nil).Run(func(args mock.Arguments) {
		p := args.Get(2).(*model.ParsedLog)
		assert.Equal(t, len(parsed.Nodes), len(p.Nodes))
	})

	id, err := svc.ParseAndSave(context.Background(), "data/ibdiagnet2.db_csv")
	assert.NoError(t, err)
	assert.Equal(t, 42, id)
	repo.AssertExpectations(t)
}

func TestResolveUnderDataDir(t *testing.T) {
	p, err := ResolveUnderDataDir(".", "data/ibdiagnet2.db_csv")
	assert.NoError(t, err)
	assert.Contains(t, p, "ibdiagnet2.db_csv")

	_, err = ResolveUnderDataDir(".", "../etc/passwd")
	assert.ErrorIs(t, err, ErrInvalidDataPath)

	_, err = ResolveUnderDataDir(".", "other/file")
	assert.ErrorIs(t, err, ErrInvalidDataPath)
}
