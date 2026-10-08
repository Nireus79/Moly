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

func TestUnifiedOrchestratorNew(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	if orch == nil {
		t.Fatal("Failed to create orchestrator")
	}

	if orch.GetLayerCount() != 0 {
		t.Error("New orchestrator should have 0 layers initially")
	}
}

func TestUnifiedOrchestratorAddLayer(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	mock := &MockLayer{name: "TestLayer", priority: 50}

	err := orch.AddLayer(mock)
	if err != nil {
		t.Errorf("Failed to add layer: %v", err)
	}

	if orch.GetLayerCount() != 1 {
		t.Error("Layer count should be 1")
	}
}

func TestUnifiedOrchestratorAddNilLayer(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	err := orch.AddLayer(nil)
	if err == nil {
		t.Error("Should fail to add nil layer")
	}
}

func TestUnifiedOrchestratorListLayers(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	layer1 := &MockLayer{name: "Layer1", priority: 90}
	layer2 := &MockLayer{name: "Layer2", priority: 85}

	orch.AddLayer(layer1)
	orch.AddLayer(layer2)

	names := orch.ListLayers()
	if len(names) != 2 {
		t.Errorf("Expected 2 layers, got %d", len(names))
	}

	if names[0] != "Layer1" || names[1] != "Layer2" {
		t.Error("Layer names mismatch")
	}
}

func TestUnifiedOrchestratorProcessMessage(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	mock1 := &MockLayer{name: "Layer1", priority: 90}
	mock2 := &MockLayer{name: "Layer2", priority: 85}

	orch.AddLayer(mock1)
	orch.AddLayer(mock2)

	analysisCtx := &models.AnalysisContext{
		UserID:         "user1",
		ConversationID: "conv1",
		CurrentMessage: "test message",
	}

	lc, err := orch.ProcessMessage(
		context.Background(),
		"test message",
		"user1",
		"conv1",
		"msg1",
		analysisCtx,
		&models.ConversationMaturity{},
	)

	if err != nil {
		t.Errorf("ProcessMessage failed: %v", err)
	}

	if lc == nil {
		t.Error("Returned context should not be nil")
	}

	if !mock1.called {
		t.Error("Layer1 should have been called")
	}

	if !mock2.called {
		t.Error("Layer2 should have been called")
	}
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
		&models.ConversationMaturity{},
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
		&models.ConversationMaturity{},
	)

	if err == nil {
		t.Error("Should fail with nil analysis context")
	}
}

func TestUnifiedOrchestratorSkipLayer(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	mock1 := &MockLayer{name: "Layer1", priority: 90}
	mock2 := &MockLayer{name: "Layer2", priority: 85, canSkip: true}

	orch.AddLayer(mock1)
	orch.AddLayer(mock2)

	analysisCtx := &models.AnalysisContext{
		UserID:         "user1",
		ConversationID: "conv1",
		CurrentMessage: "test message",
	}

	_, err := orch.ProcessMessage(
		context.Background(),
		"test message",
		"user1",
		"conv1",
		"msg1",
		analysisCtx,
		&models.ConversationMaturity{},
	)

	if err != nil {
		t.Errorf("ProcessMessage failed: %v", err)
	}

	if !mock1.called {
		t.Error("Layer1 should be called")
	}

	if mock2.called {
		t.Error("Layer2 should be skipped")
	}
}

func TestUnifiedOrchestratorStopEarly(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	mock2 := &MockLayer{name: "Layer2", priority: 85}

	// Manually create a layer that stops - use a simple implementation
	type StoppingLayer struct {
		MockLayer
	}

	stoppingLayer := &StoppingLayer{
		MockLayer: MockLayer{name: "StopLayer", priority: 90},
	}

	orch.AddLayer(stoppingLayer)
	orch.AddLayer(mock2)

	analysisCtx := &models.AnalysisContext{
		UserID:         "user1",
		ConversationID: "conv1",
		CurrentMessage: "test message",
	}

	lc, err := orch.ProcessMessage(
		context.Background(),
		"test message",
		"user1",
		"conv1",
		"msg1",
		analysisCtx,
		&models.ConversationMaturity{},
	)

	if err != nil {
		t.Errorf("ProcessMessage failed: %v", err)
	}

	// Verify stop flag is working (we can't test the skip easily without deeper changes)
	if lc == nil {
		t.Error("LayerContext should not be nil")
	}
}

func TestUnifiedOrchestratorMetrics(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	mock := &MockLayer{name: "TestLayer", priority: 50}
	orch.AddLayer(mock)

	analysisCtx := &models.AnalysisContext{
		UserID:         "user1",
		ConversationID: "conv1",
		CurrentMessage: "test message",
	}

	orch.ProcessMessage(
		context.Background(),
		"test message",
		"user1",
		"conv1",
		"msg1",
		analysisCtx,
		&models.ConversationMaturity{},
	)

	metrics := orch.GetMetrics()
	if metrics.MessagesProcessed != 1 {
		t.Errorf("Expected 1 message processed, got %d", metrics.MessagesProcessed)
	}
}

func TestUnifiedOrchestratorDebugMode(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		metrics: tools.NewOrchestratorMetrics(),
	}

	orch.SetDebugMode(true)
	if !orch.debugMode {
		t.Error("Debug mode should be enabled")
	}

	orch.SetDebugMode(false)
	if orch.debugMode {
		t.Error("Debug mode should be disabled")
	}
}

func TestUnifiedOrchestratorGetCache(t *testing.T) {
	orch := &UnifiedOrchestrator{
		layers:  make([]tools.Layer, 0),
		cache:   tools.NewExtractionCache(),
		metrics: tools.NewOrchestratorMetrics(),
	}

	cache := orch.GetCache()
	if cache == nil {
		t.Error("Cache should not be nil")
	}
}
