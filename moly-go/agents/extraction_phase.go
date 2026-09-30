package agents

import (
	"context"
	"fmt"
	"log"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// ExtractionPhaseInput contains everything needed for extraction phase
type ExtractionPhaseInput struct {
	UserID         string
	ConversationID string
	MessageID      string
	Message        string
	MessageCount   int
	RecentMessages []models.Message
	UserProfile    *models.AboutMe
	Cache          *tools.LLMCache
}

// ExtractionPhaseOutput contains all results from extraction phase
type ExtractionPhaseOutput struct {
	Artifact               *models.ExtractionArtifact     // The extraction results
	AnalysisContext       *models.AnalysisContext         // Context built from extraction
	Conflicts             []ConflictDetectorResult        // Conflicts detected
	AmbiguousEntities     []models.ExtractedEntity        // Entities needing clarification
	HighConfidenceContacts []models.ExtractedEntity       // High-confidence contacts
}

// ExtractionPhase: Layer 0 of orchestrator - centralized extraction
type ExtractionPhase struct {
	intentDetector    *LLMIntentDetector
	extractionStore   *tools.ExtractionStore
	conflictDetector  *ConflictDetector
	database          *database.Database
}

// NewExtractionPhase creates a new extraction phase
func NewExtractionPhase(
	intentDetector *LLMIntentDetector,
	extractionStore *tools.ExtractionStore,
	conflictDetector *ConflictDetector,
	db *database.Database,
) *ExtractionPhase {
	return &ExtractionPhase{
		intentDetector:   intentDetector,
		extractionStore:  extractionStore,
		conflictDetector: conflictDetector,
		database:         db,
	}
}

// Run executes the extraction phase
func (ep *ExtractionPhase) Run(ctx context.Context, input *ExtractionPhaseInput) (*ExtractionPhaseOutput, error) {
	if input == nil {
		return nil, fmt.Errorf("extraction phase input is nil")
	}

	if input.Message == "" {
		log.Printf("[ExtractionPhase] Skipping extraction for empty message")
		return &ExtractionPhaseOutput{
			Artifact:         nil,
			AnalysisContext:  nil,
			Conflicts:        []ConflictDetectorResult{},
			AmbiguousEntities: []models.ExtractedEntity{},
		}, nil
	}

	log.Printf("[ExtractionPhase] Starting extraction phase for user=%s, message_len=%d", input.UserID, len(input.Message))

	// Step 1: Extract with SmartExtractEntities (LLM or fallback)
	smartResult := ep.intentDetector.SmartExtractEntities(ctx, input.Message, input.Cache)
	if smartResult == nil || smartResult.Artifact == nil {
		log.Printf("[ExtractionPhase] SmartExtractEntities returned nil artifact")
		return nil, fmt.Errorf("extraction failed: no artifact produced")
	}

	artifact := smartResult.Artifact
	artifact.UserID = input.UserID
	artifact.ConversationID = input.ConversationID
	artifact.MessageID = input.MessageID

	log.Printf("[ExtractionPhase] ✓ Extracted %d entities (source=%s, avg_confidence=%.2f)",
		len(artifact.Entities), artifact.Source, artifact.AverageConfidence)

	// Step 2: Save to extraction store (5-min TTL)
	ep.extractionStore.Save(artifact)
	log.Printf("[ExtractionPhase] ✓ Saved extraction to store")

	// Step 3: Detect conflicts with database
	conflicts := []ConflictDetectorResult{}
	if ep.conflictDetector != nil {
		detectedConflicts, err := ep.conflictDetector.DetectConflicts(
			ctx,
			input.UserID,
			input.ConversationID,
			artifact.Entities,
		)
		if err != nil {
			log.Printf("[ExtractionPhase] Warning: Conflict detection failed: %v", err)
		} else {
			conflicts = detectedConflicts
			if len(conflicts) > 0 {
				log.Printf("[ExtractionPhase] ✓ Detected %d conflicts", len(conflicts))
			}
		}
	}

	// Step 4: Get ambiguous entities
	ambiguousEntities := artifact.AmbiguousEntities()
	if len(ambiguousEntities) > 0 {
		log.Printf("[ExtractionPhase] ⚠ Identified %d ambiguous entities needing clarification", len(ambiguousEntities))
	}

	// Step 5: Get high-confidence contacts
	highConfidenceContacts := artifact.HighConfidenceContacts()
	if len(highConfidenceContacts) > 0 {
		log.Printf("[ExtractionPhase] ✓ Identified %d high-confidence contacts", len(highConfidenceContacts))
	}

	// Step 6: Build AnalysisContext FROM extraction (not from DB!)
	analysisCtx := ep.buildContextFromExtraction(artifact, input)

	log.Printf("[ExtractionPhase] ✓ Built AnalysisContext from extraction (extraction_source=%s)", artifact.Source)

	return &ExtractionPhaseOutput{
		Artifact:               artifact,
		AnalysisContext:       analysisCtx,
		Conflicts:             conflicts,
		AmbiguousEntities:     ambiguousEntities,
		HighConfidenceContacts: highConfidenceContacts,
	}, nil
}

// buildContextFromExtraction builds AnalysisContext directly from extraction results
// This is the KEY FIX: Don't rebuild from database, use the LLM extraction results
func (ep *ExtractionPhase) buildContextFromExtraction(
	artifact *models.ExtractionArtifact,
	input *ExtractionPhaseInput,
) *models.AnalysisContext {

	ctx := &models.AnalysisContext{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageCount:   input.MessageCount,

		// Built from extraction, not database rebuild
		ExtractedEntities:   artifact.Entities,
		ExtractedSource:     artifact.Source,
		ExtractedConfidence: artifact.AverageConfidence,
		ExtractionQuality: &models.ExtractionQuality{
			SubjectAttributed: artifact.SubjectAttributed,
			NegationPreserved: artifact.NegationPreserved,
			LLMExtraction:     artifact.LLMSuccess,
		},

		// Contacts from extraction (high-confidence)
		Contacts:      []models.Contact{},

		// Recent messages
		RecentMessages: input.RecentMessages,

		// User profile
		UserProfile: input.UserProfile,

		// Extraction timing
		ExtractionDuration: artifact.Duration,
	}

	// Add contacts from extraction (only high-confidence ones)
	for _, entity := range artifact.HighConfidenceContacts() {
		contact := models.Contact{
			Name:       entity.Value,
			Confidence: entity.Confidence,
		}
		ctx.Contacts = append(ctx.Contacts, contact)
	}

	// Add preferences from extraction
	preferences := []string{}
	characteristics := []string{}
	for _, entity := range artifact.Entities {
		if entity.Confidence >= 0.70 {
			switch entity.Type {
			case "preference", "negation":
				preferences = append(preferences, entity.Value)
			case "characteristic":
				characteristics = append(characteristics, entity.Value)
			}
		}
	}
	ctx.ExtractedPreferences = preferences
	ctx.ExtractedCharacteristics = characteristics

	log.Printf("[ExtractionPhase] Context built: %d entities, %d contacts, %d preferences, %d characteristics",
		len(artifact.Entities), len(ctx.Contacts), len(preferences), len(characteristics))

	return ctx
}
