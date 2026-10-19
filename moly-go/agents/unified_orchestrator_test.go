package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

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
