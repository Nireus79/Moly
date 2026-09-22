package tools

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/database"
)

// InlineConflictResolver detects and resolves conflicts within conversation flow
type InlineConflictResolver struct {
	db        *database.Database
	llmClient LLMProvider
}

// NewInlineConflictResolver creates a new resolver
func NewInlineConflictResolver(db *database.Database) *InlineConflictResolver {
	return &InlineConflictResolver{db: db, llmClient: nil}
}

// NewInlineConflictResolverWithLLM creates a resolver with LLM capability
func NewInlineConflictResolverWithLLM(db *database.Database, llm LLMProvider) *InlineConflictResolver {
	return &InlineConflictResolver{db: db, llmClient: llm}
}

// ConflictAwareResponse wraps a response with conflict info
type ConflictAwareResponse struct {
	Response           string
	HasPendingConflict bool
	ConflictMessage    string // Question to ask user
	ConflictID         int64  // For tracking resolution
	ResolutionApplied  bool   // Whether a conflict was resolved
}

// CheckAndAskForConflicts detects pending conflicts and generates questions
func (r *InlineConflictResolver) CheckAndAskForConflicts(userID string) *ConflictAwareResponse {
	log.Printf("[InlineConflictResolver] Checking for pending conflicts for user %s", userID)

	conflictRepo := r.db.GetContextConflictRepository()
	conflicts, err := conflictRepo.GetUnresolved(userID)
	if err != nil {
		log.Printf("[InlineConflictResolver] Error loading conflicts: %v", err)
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	if len(conflicts) == 0 {
		log.Printf("[InlineConflictResolver] No pending conflicts")
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	// CRITICAL FIX: Filter out false "intention" conflicts
	// Different message intents (greet, ask, inform) are NOT conflicts
	// Real conflicts are VALUE/PREFERENCE mismatches (how you want to communicate, relationship types, etc.)
	validConflicts := []*database.ContextConflict{}
	for _, c := range conflicts {
		// SKIP: intention conflicts are NOT real conflicts (just different message intents)
		if c.ConflictType == "intention" {
			log.Printf("[InlineConflictResolver] Skipping false 'intention' conflict %d: different message intents are normal", c.ID)
			continue
		}
		validConflicts = append(validConflicts, c)
	}

	if len(validConflicts) == 0 {
		log.Printf("[InlineConflictResolver] No REAL conflicts found (skipped %d false intention conflicts)", len(conflicts))
		return &ConflictAwareResponse{
			HasPendingConflict: false,
			ResolutionApplied:  false,
		}
	}

	// Found real conflicts - generate question for the first one
	conflict := validConflicts[0]
	log.Printf("[InlineConflictResolver] Found REAL conflict %d: %s", conflict.ID, conflict.ConflictType)

	question := r.generateConflictQuestion(conflict)

	return &ConflictAwareResponse{
		HasPendingConflict: true,
		ConflictMessage:    question,
		ConflictID:         conflict.ID,
		ResolutionApplied:  false,
	}
}

// generateConflictQuestion creates a natural conversational question for the user
func (r *InlineConflictResolver) generateConflictQuestion(conflict *database.ContextConflict) string {
	contextExtractor := NewContextExtractorHelper()
	oldContextDesc := contextExtractor.GetContextDescription(r.getContextFromDetails(conflict, "oldContext"))
	newContextDesc := contextExtractor.GetContextDescription(r.getContextFromDetails(conflict, "newContext"))

	switch conflict.ConflictType {
	case "aboutme_communication_style":
		return fmt.Sprintf(
			"I remember you saying you prefer '%v' %s. "+
				"Now you're telling me '%v' %s. "+
				"Are both true for different situations, or has your preference changed?",
			conflict.SavedValue, oldContextDesc,
			conflict.ExtractedValue, newContextDesc,
		)

	case "contact_relationship":
		contactName := r.getContactFromDetails(conflict)
		return fmt.Sprintf(
			"I had noted that %s is your '%v'. "+
				"Now you're mentioning they're your '%v'. "+
				"Did that change, or does it depend on context?",
			contactName, conflict.SavedValue, conflict.ExtractedValue,
		)

	case "contact_characteristics":
		contactName := r.getContactFromDetails(conflict)
		oldChars := r.formatArray(conflict.SavedValue)
		newChars := r.formatArray(conflict.ExtractedValue)
		return fmt.Sprintf(
			"I used to know %s as: %s. "+
				"Now you're describing them as: %s. "+
				"Should I update how I see them?",
			contactName, oldChars, newChars,
		)

	case "intention":
		return fmt.Sprintf(
			"Earlier you mentioned your goal was '%v'. "+
				"Now you're saying '%v'. "+
				"Are you shifting priorities, or are these goals for different situations?",
			conflict.SavedValue, conflict.ExtractedValue,
		)

	default:
		return fmt.Sprintf(
			"I noticed you said '%v' before, but now '%v'. "+
				"Which is accurate?",
			conflict.SavedValue, conflict.ExtractedValue,
		)
	}
}

// ParseResolutionFromResponse analyzes user's response using LLM reasoning
func (r *InlineConflictResolver) ParseResolutionFromResponse(userResponse string, conflict *database.ContextConflict) string {
	log.Printf("[InlineConflictResolver] Parsing resolution from response: %.100s...", userResponse)

	// Use LLM for intelligent understanding of user's choice
	if r.llmClient != nil {
		return r.parseResolutionWithLLM(userResponse, conflict)
	}

	// Fallback: basic heuristics if no LLM
	log.Printf("[InlineConflictResolver] No LLM available, using basic heuristics")
	return r.parseResolutionBasic(userResponse)
}

// parseResolutionWithLLM uses LLM to understand user's resolution choice
func (r *InlineConflictResolver) parseResolutionWithLLM(userResponse string, conflict *database.ContextConflict) string {
	ctx := context.Background()

	prompt := fmt.Sprintf(`User is resolving a conflict about "%s":
- Previously said: "%s"
- Now says: "%s"

Their response to "are both true, or has it changed?": "%s"

Determine which resolution they're choosing:
- "keep_saved": They confirm the old value is still correct
- "use_extracted": They've changed, new value is correct
- "merge": Both are true in different contexts
- "unclear": Their response doesn't clearly indicate which

Respond with ONLY: keep_saved|use_extracted|merge|unclear`,
		conflict.ConflictType, conflict.SavedValue, conflict.ExtractedValue, userResponse)

	req := &LLMRequest{
		SystemPrompt: `You understand user intent. Analyze their response to conflict resolution questions.
Determine if they: (1) confirm old value is right, (2) accept new value, (3) say both are true, or (4) are unclear.`,
		UserPrompt:  prompt,
		Temperature: 0.2,
		MaxTokens:   20,
	}

	resp, err := r.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[InlineConflictResolver] LLM error: %v, using basic heuristics", err)
		return r.parseResolutionBasic(userResponse)
	}

	resolution := strings.ToLower(strings.TrimSpace(resp.Content))
	log.Printf("[InlineConflictResolver] LLM determined resolution: %s", resolution)

	// Map response to resolution types
	switch resolution {
	case "keep_saved":
		return "keep_saved"
	case "use_extracted":
		return "use_extracted"
	case "merge":
		return "merge"
	default:
		log.Printf("[InlineConflictResolver] LLM response unclear: %s", resolution)
		return ""
	}
}

// parseResolutionBasic uses basic heuristics when LLM unavailable
func (r *InlineConflictResolver) parseResolutionBasic(userResponse string) string {
	lower := strings.ToLower(userResponse)

	// Simple checks for clear signals
	if strings.Contains(lower, "both") || strings.Contains(lower, "depends on") || strings.Contains(lower, "it depends") {
		log.Printf("[InlineConflictResolver] Basic heuristic: merge (both are true)")
		return "merge"
	}
	if strings.Contains(lower, "changed") || strings.Contains(lower, "new") || strings.Contains(lower, "different now") {
		log.Printf("[InlineConflictResolver] Basic heuristic: use_extracted (changed)")
		return "use_extracted"
	}
	if strings.Contains(lower, "still") || strings.Contains(lower, "original") || strings.Contains(lower, "was right") {
		log.Printf("[InlineConflictResolver] Basic heuristic: keep_saved (still right)")
		return "keep_saved"
	}

	// Unclear
	log.Printf("[InlineConflictResolver] Basic heuristic: unclear response")
	return ""
}

// ApplyConflictResolution applies the parsed resolution
func (r *InlineConflictResolver) ApplyConflictResolution(userID string, conflictID int64, resolution string) *ResolutionResult {
	if resolution == "" {
		// User's response wasn't clear enough
		return &ResolutionResult{
			Success: false,
			Message: "I couldn't tell from your response - could you clarify?",
		}
	}

	// Load the conflict
	conflictRepo := r.db.GetContextConflictRepository()
	conflicts, err := conflictRepo.GetUnresolved(userID)
	if err != nil {
		return &ResolutionResult{
			Success: false,
			Message: "Failed to load conflicts",
		}
	}

	var conflict *database.ContextConflict
	for _, c := range conflicts {
		if c.ID == conflictID {
			conflict = c
			break
		}
	}

	if conflict == nil {
		return &ResolutionResult{
			Success: false,
			Message: "Conflict not found",
		}
	}

	// Apply resolution
	handler := NewConflictResolutionHandler(r.db)
	result := handler.ApplyResolution(conflict, resolution)

	return result
}

// Helper functions

func (r *InlineConflictResolver) getContextFromDetails(conflict *database.ContextConflict, key string) string {
	if conflict.ResolutionDetails != nil {
		if ctx, ok := conflict.ResolutionDetails[key].(string); ok {
			return ctx
		}
	}
	return "general"
}

func (r *InlineConflictResolver) getContactFromDetails(conflict *database.ContextConflict) string {
	if conflict.ResolutionDetails != nil {
		if contact, ok := conflict.ResolutionDetails["contact"].(string); ok {
			return contact
		}
	}
	return "this person"
}

func (r *InlineConflictResolver) formatArray(value interface{}) string {
	if arrVal, ok := value.([]interface{}); ok {
		var items []string
		for _, v := range arrVal {
			items = append(items, fmt.Sprintf("%v", v))
		}
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%v", value)
}

func (r *InlineConflictResolver) countKeywords(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		// Count occurrences of keyword as whole words
		if strings.Contains(text, keyword) {
			count++
		}
	}
	return count
}
