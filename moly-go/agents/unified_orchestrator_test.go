package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

// MockLayer is a test layer
type MockLayer struct {
	name     string
	priority int
	canSkip  bool
	err      error
	called   bool
}

func (m *MockLayer) Name() string {
	return m.name
}

func (m *MockLayer) Priority() int {
	return m.priority
}

func (m *MockLayer) CanSkip(lc *tools.LayerContext) bool {
	return m.canSkip
}

func (m *MockLayer) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	m.called = true
	if m.err != nil {
		return lc, m.err
	}
	return lc, nil
}

func TestUnifiedOrchestratorProcessMessageEmptyMessage(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	_, err := orch.ProcessMessage(
		context.Background(),
		"",
		"user1",
		"conv1",
		"msg1",
		&models.AnalysisContext{},
	)

	if err == nil {
		t.Error("Should fail with empty message")
	}
}

func TestUnifiedOrchestratorProcessMessageNilContext(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	_, err := orch.ProcessMessage(
		context.Background(),
		"test",
		"user1",
		"conv1",
		"msg1",
		nil,
	)

	if err == nil {
		t.Error("Should fail with nil analysis context")
	}
}
