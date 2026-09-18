package orchestration

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"moly/database"
	"moly/extraction"
	"moly/generation"
	"moly/models"
	"moly/storage"
	"moly/tools"
)

// PipelineState - Shared state flowing through pipeline stages
type PipelineState struct {
	// Input
	UserID         string
	ConversationID string
	MessageContent string
	AuthToken      string

	// Stage 1: Validation & Context Loading
	UserContext *database.UserContextSnapshot
	Error       error

	// Stage 2: Pending Input Handling
	ResolvedPendingID int64 // If user answered a pending input

	// Stage 3: Response Generation
	Response           *models.ConversationResponse
	PendingInput       *database.PendingInput // If conflict/clarification created
	Insights           []models.Reflection
	UpdatedAboutMe     *models.AboutMe // If resolution changed user model

	// Timing
	StartTime time.Time
}

// MessagePipeline - 4-stage orchestrator
type MessagePipeline struct {
	db           *database.Database
	writer       *storage.BatchWriter
	llm          *tools.LLMClient
	ethicalGate  *generation.EthicalGate
}

// NewMessagePipeline - Create pipeline
func NewMessagePipeline(db *database.Database, llm *tools.LLMClient) *MessagePipeline {
	return &MessagePipeline{
		db:          db,
		writer:      storage.NewBatchWriter(db),
		llm:         llm,
		ethicalGate: generation.NewEthicalGate(),
	}
}

// ProcessMessage - Run complete 4-stage pipeline
func (p *MessagePipeline) ProcessMessage(userID, conversationID, messageContent string) (*models.ConversationResponse, error) {
	log.Printf("[Pipeline] Starting message processing: user=%s conv=%s", userID, conversationID)

	state := &PipelineState{
		UserID:         userID,
		ConversationID: conversationID,
		MessageContent: messageContent,
		StartTime:      time.Now(),
	}

	// Stage 1: Validate & Load Context
	if err := p.Stage1_ValidateAndLoad(state); err != nil {
		log.Printf("[Pipeline] Stage 1 failed: %v", err)
		return p.errorResponse("Failed to load context", err), err
	}

	// Stage 2: Handle Pending Input
	if err := p.Stage2_HandlePendingInput(state); err != nil {
		log.Printf("[Pipeline] Stage 2 failed: %v", err)
		return p.errorResponse("Failed to handle pending input", err), err
	}

	// Stage 3: Generate Response
	if err := p.Stage3_GenerateResponse(state); err != nil {
		log.Printf("[Pipeline] ❌ Stage 3 failed: %v", err)
		if f, errFile := os.OpenFile("/tmp/moly_pending_input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); errFile == nil {
			fmt.Fprintf(f, "[ERROR_STAGE3] %v\n", err)
			f.Close()
		}
		return p.errorResponse("Failed to generate response", err), err
	}

	// Stage 4: Save & Return
	if err := p.Stage4_SaveAndRespond(state); err != nil {
		log.Printf("[Pipeline] Stage 4 failed: %v", err)
		return p.errorResponse("Failed to save data", err), err
	}

	elapsed := time.Since(state.StartTime)
	log.Printf("[Pipeline] Complete: %dms response=%v pending=%v", elapsed.Milliseconds(), state.Response != nil, state.PendingInput != nil)

	state.Response.ProcessingTimeMs = int(elapsed.Milliseconds())
	return state.Response, nil
}

// Stage1_ValidateAndLoad - Validate token and load user context
func (p *MessagePipeline) Stage1_ValidateAndLoad(state *PipelineState) error {
	log.Printf("[Pipeline:Stage1] Loading context for user=%s conv=%s", state.UserID, state.ConversationID)

	// Ensure conversation exists (required for foreign keys)
	if err := p.ensureConversationExists(state.UserID, state.ConversationID); err != nil {
		log.Printf("[Pipeline:Stage1] WARNING: Failed to ensure conversation exists: %v", err)
		// Don't fail - continue anyway, may work without explicit conversation record
	}

	// Load complete user context (5 queries, optimized)
	ctx, err := p.db.LoadUserContext(state.UserID, state.ConversationID, 10, 5)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	state.UserContext = ctx
	log.Printf("[Pipeline:Stage1] ✓ Loaded: %d messages, %d pending, %d insights, %d contacts",
		len(ctx.RecentMessages), len(ctx.PendingInputs), len(ctx.RecentInsights), len(ctx.Contacts))

	return nil
}

// ensureConversationExists - Create conversation record if it doesn't exist
func (p *MessagePipeline) ensureConversationExists(userID, conversationID string) error {
	log.Printf("[Pipeline:EnsureConv] Creating/verifying conversation: user=%s conv=%s", userID, conversationID)

	query := `
		INSERT OR IGNORE INTO conversations (id, user_id, name, created_at)
		VALUES (?, ?, ?, ?)
	`
	result, err := p.db.Exec(query, conversationID, userID, conversationID, time.Now().Unix())
	if err != nil {
		log.Printf("[Pipeline:EnsureConv] ERROR: %v", err)
		return fmt.Errorf("failed to create conversation: %w", err)
	}

	rows, err := result.RowsAffected()
	log.Printf("[Pipeline:EnsureConv] Rows affected: %d", rows)
	return nil
}

// Stage2_HandlePendingInput - Check if user is answering a pending input
func (p *MessagePipeline) Stage2_HandlePendingInput(state *PipelineState) error {
	log.Printf("[Pipeline:Stage2] Checking pending inputs: %d unresolved", len(state.UserContext.PendingInputs))

	// If no pending inputs, continue
	if len(state.UserContext.PendingInputs) == 0 {
		log.Printf("[Pipeline:Stage2] No pending inputs, continuing")
		return nil
	}

	firstPending := state.UserContext.PendingInputs[0]
	log.Printf("[Pipeline:Stage2] Found pending: type=%s subtype=%s", firstPending.Type, firstPending.Subtype)

	// Try to parse answer from user message
	answer := p.parseAnswerFromMessage(state.MessageContent, firstPending)

	if answer == "" {
		// User didn't answer the pending question, continue with normal flow
		log.Printf("[Pipeline:Stage2] User didn't answer pending question, continuing")
		return nil
	}

	// User answered! Process resolution
	log.Printf("[Pipeline:Stage2] User answered: %s", answer)

	// Resolve in database
	if err := p.db.GetPendingInputRepository().Resolve(firstPending.ID, answer); err != nil {
		return fmt.Errorf("failed to resolve pending input: %w", err)
	}

	// Mark as applied
	if err := p.db.GetPendingInputRepository().MarkApplied(firstPending.ID); err != nil {
		return fmt.Errorf("failed to mark applied: %w", err)
	}

	// Remove from context so Stage 3 doesn't re-detect
	state.ResolvedPendingID = firstPending.ID
	state.UserContext.PendingInputs = state.UserContext.PendingInputs[1:]

	log.Printf("[Pipeline:Stage2] ✓ Pending input resolved and marked applied")
	return nil
}

// parseAnswerFromMessage - Detect if message answers pending input
func (p *MessagePipeline) parseAnswerFromMessage(msg string, pending database.PendingInput) string {
	switch pending.Type {
	case "conflict":
		// Check if msg contains conflict resolution markers
		if containsAny(msg, []string{"both", "different", "different situations"}) {
			return "merge"
		}
		if containsAny(msg, []string{"preference", "changed", "shift"}) {
			return "update"
		}

	case "clarification":
		// Message itself is the clarification answer
		if len(msg) > 0 {
			return msg
		}

	case "approval":
		// Check for yes/no
		if containsAny(msg, []string{"yes", "approve", "true", "correct"}) {
			return "approve"
		}
		if containsAny(msg, []string{"no", "reject", "false", "incorrect"}) {
			return "reject"
		}
	}

	return ""
}

// containsAny - Check if string contains any of the substrings (case-insensitive)
func containsAny(text string, substrings []string) bool {
	lower := toLower(text)
	for _, substr := range substrings {
		if contains(lower, toLower(substr)) {
			return true
		}
	}
	return false
}

// toLower - Simple case conversion
func toLower(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

// contains - Simple substring search
func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Stage3_GenerateResponse - Generate response with LLM and context awareness
func (p *MessagePipeline) Stage3_GenerateResponse(state *PipelineState) error {
	log.Printf("[Pipeline:Stage3] Generating response with LLM integration")

	ctx := context.Background()

	detector := extraction.NewConflictDetector()

	// Extract context from message using LLM (or fallback to heuristic)
	var extracted *extraction.ExtractedContext
	if p.llm != nil {
		llmExtractor := extraction.NewLLMMessageExtractor(p.llm)
		var err error
		extracted, err = llmExtractor.Extract(ctx, state.MessageContent, state.UserContext.AboutMe)
		if err != nil {
			log.Printf("[Pipeline:Stage3] WARNING: LLM extraction failed, using heuristic: %v", err)
			extractor := extraction.NewMessageExtractor()
			extracted = extractor.Extract(state.MessageContent, state.UserContext.AboutMe)
		}
	} else {
		log.Printf("[Pipeline:Stage3] WARNING: LLM client not available, using heuristic extraction")
		extractor := extraction.NewMessageExtractor()
		extracted = extractor.Extract(state.MessageContent, state.UserContext.AboutMe)
	}

	// Detect conflicts
	conflicts := detector.Detect(extracted, state.UserContext.AboutMe)

	if len(conflicts) > 0 && detector.HasSignificantConflict(conflicts) {
		log.Printf("[Pipeline:Stage3] Conflict detected: %s", conflicts[0].Type)

		// Generate conflict resolution request
		state.Response = &models.ConversationResponse{
			Phase:    "responding",
			Response: detector.ConflictToQuestion(conflicts[0]),
			Metadata: map[string]interface{}{
				"type":             "conflict_question",
				"conflict_type":    conflicts[0].Type,
				"conflict_field":   conflicts[0].Field,
				"stored_value":     conflicts[0].StoredValue,
				"extracted_value":  conflicts[0].ExtractedValue,
			},
		}

		// Create pending input for this conflict
		generator := generation.NewResponseGenerator()
		state.PendingInput = generator.GeneratePendingInputForConflict(
			state.UserID, state.ConversationID, conflicts[0])

		log.Printf("[Pipeline:Stage3] ✓ Conflict question generated, pending input created")
		return nil
	}

	// No conflict: generate conversational response
	var resp *models.ConversationResponse
	if p.llm != nil {
		llmGenerator := generation.NewLLMResponseGenerator(p.llm)
		var err error
		resp, err = llmGenerator.GenerateResponse(ctx, state.MessageContent, extracted, state.UserContext)
		if err != nil {
			return fmt.Errorf("failed to generate LLM response: %w", err)
		}

		// Apply ethical gate to response
		resp = p.ethicalGate.ApplyGate(resp, state.MessageContent)

		// Extract insights from message using LLM (will be stored as reflection)
		if len(state.MessageContent) > 0 {
			insight, err := llmGenerator.GenerateInsight(ctx, state.MessageContent, extracted, state.UserID, state.ConversationID)
			if err != nil {
				log.Printf("[Pipeline:Stage3] WARNING: Failed to extract insight: %v", err)
			} else if insight != nil {
				state.Insights = append(state.Insights, *insight)
			}
		}
	} else {
		// LLM unavailable: use heuristic response generator
		log.Printf("[Pipeline:Stage3] LLM not available, using heuristic response generation")
		respGenerator := generation.NewResponseGenerator()
		var err error
		resp, err = respGenerator.GenerateResponse(state.MessageContent, state.UserContext)
		if err != nil {
			return fmt.Errorf("failed to generate heuristic response: %w", err)
		}

		// Apply ethical gate to response
		resp = p.ethicalGate.ApplyGate(resp, state.MessageContent)
	}

	state.Response = resp
	log.Printf("[Pipeline:Stage3] ✓ Response generated: %s", resp.Metadata["type"])

	// Debug: write to file
	if f, err := os.OpenFile("/tmp/moly_pending_input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "[STAGE3_COMPLETE_SUCCESS]\n")
		f.Close()
	}

	return nil
}

// Stage4_SaveAndRespond - Save all data in transaction
func (p *MessagePipeline) Stage4_SaveAndRespond(state *PipelineState) error {
	msg := "[STAGE4_ENTERED]"
	log.Printf("[Pipeline:Stage4] %s", msg)

	// Write to debug file to verify this function is called
	if f, err := os.OpenFile("/tmp/moly_pending_input.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprintf(f, "%s\n", msg)
		f.Close()
	}

	log.Printf("[Pipeline:Stage4] Saving: pending=%v response=%v", state.PendingInput != nil, state.Response != nil)

	// Create user message to save
	userMessage := &models.ChatMessage{
		ID:             fmt.Sprintf("%s-%d", state.ConversationID, time.Now().UnixNano()),
		UserID:         state.UserID,
		ConversationID: state.ConversationID,
		Role:           "user",
		Content:        state.MessageContent,
		CreatedAt:      time.Now().Unix(),
	}

	batchReq := storage.BatchWriteRequest{
		UserID:         state.UserID,
		ConversationID: state.ConversationID,
		Message:        userMessage,
		Response:       state.Response,
		Insights:       state.Insights,
		PendingInput:   state.PendingInput,
		UpdatedAboutMe: state.UpdatedAboutMe,
	}

	log.Printf("[Pipeline:Stage4] About to call WriteBatch (pending=%v)", batchReq.PendingInput != nil)
	if err := p.writer.WriteBatch(batchReq); err != nil {
		log.Printf("[Pipeline:Stage4] ❌ WriteBatch failed: %v", err)
		return fmt.Errorf("failed to write batch: %w", err)
	}

	log.Printf("[Pipeline:Stage4] ✓ All data saved successfully")
	return nil
}

// errorResponse - Create error response
func (p *MessagePipeline) errorResponse(message string, err error) *models.ConversationResponse {
	return &models.ConversationResponse{
		Phase:    "error",
		Response: message,
		Error:    fmt.Sprintf("%v", err),
		Metadata: make(map[string]interface{}),
	}
}
