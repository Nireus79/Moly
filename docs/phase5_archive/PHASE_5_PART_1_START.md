# Phase 5 Part 1: START HERE

**Status**: 🟡 READY TO IMPLEMENT  
**Effort**: 4 hours  
**Files**: 2 (linguistic_parser.go + linguistic_parser_test.go)  
**Output**: 4 rules, 8 tests, 50 LOC added  

---

## Quick Start

### File 1: Add to `moly-go/tools/linguistic_parser.go`

**Location**: After line 410 (after `extractLookingFor()`)

**Code to Add** (4 functions, ~50 LOC):

```go
// extractFocusDirective handles patterns like "X is my focus", "focus on X", "don't focus on X"
// Returns: {subject: "user", property: "X", type: "focus", confidence: 0.90-0.95, is_negated: bool}
func (lp *LinguisticParser) extractFocusDirective(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)
	
	// Pattern 1: "X is my focus" or "my focus is X"
	if strings.Contains(lower, "is my focus") || strings.Contains(lower, "my focus is") {
		// Extract X (word before "is my focus")
		pattern := regexp.MustCompile(`(.+?)\s+is\s+my\s+focus`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			// Get last word if multiple
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[len(words)-1]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.95,
				RawMatch:   matches[0],
			})
		}
	}
	
	// Pattern 2: "focus on X" or "let's focus on X"
	if strings.Contains(lower, "focus on ") {
		pattern := regexp.MustCompile(`focus\s+on\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				// Take first 1-2 words
				if len(words) == 1 {
					property = words[0]
				} else if len(words) >= 2 && (words[0] == "my" || words[0] == "the") {
					property = words[1]
				} else {
					property = words[0]
				}
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.90,
				RawMatch:   matches[0],
			})
		}
	}
	
	// Pattern 3: "don't focus on X" or "can't focus on X"
	if strings.Contains(lower, "don't focus") || strings.Contains(lower, "can't focus") || 
	   strings.Contains(lower, "won't focus") {
		pattern := regexp.MustCompile(`(?:don't|can't|won't)\s+focus\s+on\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[0]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "focus",
				Confidence: 0.92,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}
	
	return results
}

// extractConstraint handles "remember X", "keep in mind X", "don't forget X"
// Returns: {subject: "user", property: "X", type: "constraint", confidence: 0.85-0.90}
func (lp *LinguisticParser) extractConstraint(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)
	
	// Pattern: "remember X" or "keep in mind X" or "don't forget X"
	var pattern *regexp.Regexp
	
	if strings.Contains(lower, "remember ") {
		pattern = regexp.MustCompile(`remember\s+(?:to\s+)?([a-z\s]+?)(?:\.|,|$)`)
	} else if strings.Contains(lower, "keep in mind") {
		pattern = regexp.MustCompile(`keep\s+in\s+mind\s+([a-z\s]+?)(?:\.|,|$)`)
	} else if strings.Contains(lower, "don't forget") {
		pattern = regexp.MustCompile(`don't\s+forget\s+([a-z\s]+?)(?:\.|,|$)`)
	}
	
	if pattern != nil {
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "constraint",
				Confidence: 0.85,
				RawMatch:   matches[0],
			})
		}
	}
	
	return results
}

// extractPriority handles "my priority is X" or "prioritize X"
// Returns: {subject: "user", property: "X", type: "priority", confidence: 0.90-0.95}
func (lp *LinguisticParser) extractPriority(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)
	
	// Pattern: "my priority is X" or "priority is X"
	if strings.Contains(lower, "priority is ") {
		pattern := regexp.MustCompile(`priority\s+is\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[0]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "priority",
				Confidence: 0.95,
				RawMatch:   matches[0],
			})
		}
	}
	
	// Pattern: "prioritize X"
	if strings.Contains(lower, "prioritize ") {
		pattern := regexp.MustCompile(`prioritize\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			words := strings.Fields(property)
			if len(words) > 0 {
				property = words[0]
			}
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "priority",
				Confidence: 0.90,
				RawMatch:   matches[0],
			})
		}
	}
	
	return results
}

// extractNegatedDirective handles "not interested in X", "don't want X"
// Returns: {subject: "user", property: "X", type: "interest", confidence: 0.85-0.90, is_negated: true}
func (lp *LinguisticParser) extractNegatedDirective(message string) []ExtractionResult {
	var results []ExtractionResult
	lower := strings.ToLower(message)
	
	// Pattern: "not interested in X"
	if strings.Contains(lower, "not interested in") {
		pattern := regexp.MustCompile(`not\s+interested\s+in\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "interest",
				Confidence: 0.90,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}
	
	// Pattern: "don't want X"
	if strings.Contains(lower, "don't want") {
		pattern := regexp.MustCompile(`don't\s+want\s+([a-z\s]+?)(?:\.|,|$)`)
		matches := pattern.FindStringSubmatch(lower)
		if len(matches) > 1 {
			property := strings.TrimSpace(matches[1])
			results = append(results, ExtractionResult{
				Subject:    "user",
				Property:   property,
				Type:       "preference",
				Confidence: 0.88,
				IsNegated:  true,
				RawMatch:   matches[0],
			})
		}
	}
	
	return results
}
```

**Also Add**: Call these functions in `Parse()` method around line 100:

```go
// In Parse() method, after all other rules (around line 140):
results = append(results, lp.extractFocusDirective(message)...)
results = append(results, lp.extractConstraint(message)...)
results = append(results, lp.extractPriority(message)...)
results = append(results, lp.extractNegatedDirective(message)...)
```

---

### File 2: Add to `moly-go/tools/linguistic_parser_test.go`

**Location**: End of file (after all existing tests)

**Code to Add** (8 test functions, ~120 LOC):

```go
func TestExtractFocusDirective(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Test 1: "Lace is my focus"
	results := parser.extractFocusDirective("Lace is my focus")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from 'Lace is my focus'")
	} else if results[0].Property != "lace" {
		t.Errorf("Expected property 'lace', got '%s'", results[0].Property)
	} else if results[0].Confidence < 0.90 {
		t.Errorf("Expected confidence > 0.90, got %.2f", results[0].Confidence)
	}
	
	// Test 2: "focus on communication"
	results = parser.extractFocusDirective("Let's focus on communication")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from 'focus on communication'")
	}
	
	// Test 3: "don't focus on drama"
	results = parser.extractFocusDirective("I don't focus on drama")
	if len(results) == 0 {
		t.Errorf("Expected focus extraction from negated focus")
	} else if !results[0].IsNegated {
		t.Errorf("Expected IsNegated=true for 'don't focus on drama'")
	}
}

func TestExtractConstraint(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Test 1: "remember to be patient"
	results := parser.extractConstraint("Remember to be patient with me")
	if len(results) == 0 {
		t.Errorf("Expected constraint extraction from 'remember to be patient'")
	}
	
	// Test 2: "keep in mind I'm nervous"
	results = parser.extractConstraint("Keep in mind I'm nervous about this")
	if len(results) == 0 {
		t.Errorf("Expected constraint extraction from 'keep in mind'")
	}
}

func TestExtractPriority(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Test 1: "my priority is intimacy"
	results := parser.extractPriority("My priority is intimacy in our relationship")
	if len(results) == 0 {
		t.Errorf("Expected priority extraction from 'my priority is intimacy'")
	} else if results[0].Confidence < 0.90 {
		t.Errorf("Expected high confidence, got %.2f", results[0].Confidence)
	}
	
	// Test 2: "prioritize communication"
	results = parser.extractPriority("Please prioritize communication")
	if len(results) == 0 {
		t.Errorf("Expected priority extraction from 'prioritize'")
	}
}

func TestExtractNegatedDirective(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Test 1: "not interested in casual"
	results := parser.extractNegatedDirective("I'm not interested in casual encounters")
	if len(results) == 0 {
		t.Errorf("Expected negated directive extraction")
	} else if !results[0].IsNegated {
		t.Errorf("Expected IsNegated=true")
	}
	
	// Test 2: "don't want drama"
	results = parser.extractNegatedDirective("I don't want drama in this")
	if len(results) == 0 {
		t.Errorf("Expected negated directive from 'don't want'")
	}
}

// Integration tests
func TestMetaInstructionExtraction(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Test: Full meta-instruction message
	message := "Lace is my focus and remember to be patient"
	results := parser.Parse(message)
	
	// Should extract both focus and constraint
	hasFocus := false
	hasConstraint := false
	
	for _, r := range results {
		if r.Type == "focus" {
			hasFocus = true
		}
		if r.Type == "constraint" {
			hasConstraint = true
		}
	}
	
	if !hasFocus {
		t.Errorf("Expected to find focus directive in compound instruction")
	}
	if !hasConstraint {
		t.Errorf("Expected to find constraint in compound instruction")
	}
}

func TestNegationPreservation(t *testing.T) {
	parser := NewLinguisticParser()
	
	// Negated and non-negated should be different
	posResults := parser.extractConstraint("remember patience")
	negResults := parser.extractNegatedDirective("not interested in X")
	
	if len(posResults) == 0 || len(negResults) == 0 {
		t.Errorf("Failed to extract in negation test")
	}
	
	if negResults[0].IsNegated != true {
		t.Errorf("Expected negated to be true for negated directive")
	}
}
```

---

## Checklist

**Before Starting:**
- [ ] Read SELF_AWARENESS_OPTIMIZATION.md (understand why)
- [ ] Read PHASE_5_IMPLEMENTATION_PLAN.md (understand details)

**Implementation:**
- [ ] Copy 4 functions to linguistic_parser.go (lines after 410)
- [ ] Add 4 calls to Parse() method
- [ ] Copy 8 test functions to linguistic_parser_test.go
- [ ] Run: `go build ./...` (should pass)
- [ ] Run: `go test ./tools -v` (all tests should pass)

**Verification:**
- [ ] 8/8 tests passing
- [ ] No compiler warnings
- [ ] Build succeeds
- [ ] No regressions in existing tests

**Commit:**
```bash
git add moly-go/tools/linguistic_parser.go
git add moly-go/tools/linguistic_parser_test.go
git commit -m "Phase 5 Part 1: Add meta-instruction grammar rules

New grammar rules for self-awareness optimization:
  • extractFocusDirective: 'X is my focus', 'focus on X'
  • extractConstraint: 'remember X', 'keep in mind X'  
  • extractPriority: 'my priority is X'
  • extractNegatedDirective: 'not interested in X', 'don't want X'

Tests: 8 new tests covering all patterns
Impact: Foundation for simplified MetaInstructionDetector"
```

---

## Success Criteria

✅ All 8 tests pass  
✅ Build clean (no warnings)  
✅ No regressions in existing tests  
✅ Functions properly extract meta-instructions  
✅ Negation preserved correctly  
✅ Confidence scores reasonable (0.85-0.95)  

---

## What's Next

After Phase 1 completes:
- [ ] Phase 2: Refactor MetaInstructionDetector (Day 1, 6 hrs)
- [ ] Phase 3: Integration testing (Day 2, 4 hrs)  
- [ ] Phase 4: Optimization & deployment (Day 2, 3 hrs)

---

**Phase 1 Effort**: 4 hours  
**Expected Output**: 50 LOC, 8 tests, foundation ready  
**Next Milestone**: Phase 2 begins after Phase 1 commit  

Ready to start? Let's build Phase 5! 🚀
