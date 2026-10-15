package agents

import (
	"context"
	"testing"

	"moly/models"
	"moly/tools"
)

// An extraction that found nothing (confidence 0) is still an extraction: Layer 1 must reuse it, not call the model again. The extractor
// here has no model client, so a second call would panic. Found in a manual chat: a long message that extracted nothing was extracted twice.
func TestLayer1ReusesAnExtractionThatFoundNothing(t *testing.T) {
	l1 := NewLayer1ContextExtractionAdapter(&ContextExtractor{}, nil)
	analysis := &models.AnalysisContext{
		CurrentMessage:    "a message the extraction found nothing in",
		ExtractionQuality: &models.ExtractionQuality{LLMExtraction: true},
	}
	lc := tools.NewLayerContext(analysis, "u", "m", "c")
	if _, err := l1.Process(context.Background(), lc); err != nil {
		t.Fatal(err)
	}
	if lc.Layer1 == nil {
		t.Fatal("layer 1 produced no result")
	}
}
