# ProfileService Implementation ✅

## Status: COMPLETE & READY

ProfileService provides read API for accessing user profile data with confidence scores.

## What It Does

**Input**: User ID  
**Process**: Query all profile tables and aggregate data  
**Output**: Complete UserProfile snapshot with confidence scores

```
GetUserProfile(userID)
    ├→ GetAboutMe(userID) → AboutMe profile
    ├→ GetContacts(userID) → Contact list with patterns
    ├→ GetPatterns(userID) → Observed patterns
    ├→ GetGoals(userID) → Communication goals
    ├→ GetLearnings(userID) → Learned insights
    ├→ GetReflections(userID) → Journal entries
    └→ calculateConfidenceStats() → Aggregate stats
```

## Architecture

### Main Component

**ProfileService**
- `db`: Database connection
- Read API for all profile components

### Key Methods

**Query Methods** (read-only)
```go
GetUserProfile(userID) → *UserProfile, error
GetAboutMe(userID) → *AboutMeProfile, error
GetContacts(userID) → []ContactProfile, error
GetPatterns(userID) → []PatternProfile, error
GetGoals(userID) → []GoalProfile, error
GetLearnings(userID) → []LearningProfile, error
GetReflections(userID) → []ReflectionEntry, error
```

**Write Methods**
```go
AddReflection(userID, content, entryType, tags, aboutContactID) → int, error
UpdateGoalProgress(userID, goalID, progressNotes, status) → error
ConfirmLearning(userID, learningID) → error
RejectLearning(userID, learningID) → error
```

**Helper Methods**
```go
calculateConfidenceStats(profile) → Updates ConfidenceScores
getContactPatterns(contactID) → []CommPattern, error
```

## Data Types

### UserProfile (Complete Snapshot)
```go
type UserProfile struct {
    UserID           string
    AboutMe          *AboutMeProfile
    Contacts         []ContactProfile
    Patterns         []PatternProfile
    Goals            []GoalProfile
    Learnings        []LearningProfile
    Reflections      []ReflectionEntry
    LastUpdated      int64
    ConfidenceScores map[string]ConfidenceStats
}
```

### AboutMeProfile
- CommunicationStyle: string
- TonePreference: string
- CoreValues: []string
- Preferences: map[string]interface{}
- Confidence: float64
- ExtractedFromCount: int
- LastUpdated: int64

### ContactProfile
- ID, Name, RelationshipType, Context
- FirstMentioned, TimesMentioned, LastMentioned: timestamps
- CommPatterns: []CommPattern (patterns with this contact)

### PatternProfile
- Pattern, Category, Confidence
- ObservationCount (reinforcement tracking)
- IsActive, IsGrowthArea
- FirstObserved, LastObserved: timestamps

### GoalProfile
- Goal, Category, Status (active/achieved/paused/abandoned)
- StartedAt, TargetDate, ProgressNotes
- Confidence: float64

### LearningProfile
- LearningType (about_me/pattern/contact/goal)
- LearningKey, LearningValue
- Source (extraction/user_input)
- Confidence, IsConfirmed, IsRejected
- CreatedAt: timestamp

### ReflectionEntry
- Content (user-written text)
- Tags: []string
- EntryType (reflection/learning/breakthrough/struggle)
- AboutContactID: *int (optional)
- Created: timestamp

### ConfidenceStats
- Average, Min, Max: confidence distribution
- Count: number of items

## Data Flow

### Read Profile
```
GET /api/v2.1/profile?userID=user123
    ↓
GetUserProfile(user123)
    ├→ Query about_me_profile
    ├→ Query contacts
    ├→ Query contact_communication_patterns
    ├→ Query communication_patterns
    ├→ Query communication_goals
    ├→ Query implicit_learning
    ├→ Query reflection_journal
    └→ Aggregate + calculate confidence stats
    ↓
Return: UserProfile{
  aboutMe: {...},
  contacts: [...],
  patterns: [...],
  goals: [...],
  learnings: [...],
  reflections: [...],
  confidenceScores: {
    patterns: {average: 0.75, count: 5, min: 0.6, max: 0.9},
    goals: {average: 0.82, count: 2, min: 0.8, max: 0.85}
  }
}
```

### Add Reflection
```
POST /api/v2.1/reflections
{
  "content": "Realized I'm too passive in meetings",
  "entryType": "breakthrough",
  "tags": ["work", "assertiveness"],
  "aboutContactId": null
}
    ↓
AddReflection(userID, content, type, tags, contactID)
    └→ INSERT reflection_journal
    ↓
Return: {reflectionId: 42}
```

### Confirm Learning
```
POST /api/v2.1/learnings/15/confirm
    ↓
ConfirmLearning(userID, 15)
    └→ UPDATE implicit_learning SET is_confirmed=1
    ↓
Return: {learningId: 15, status: "confirmed"}
```

## Database Tables Read

- `about_me_profile` - Communication profile
- `contacts` - People in user's life
- `contact_communication_patterns` - How user talks to each person
- `communication_patterns` - Observed patterns
- `communication_goals` - Communication goals
- `implicit_learning` - System-learned insights
- `reflection_journal` - User's journal entries

## Error Handling

### Query Methods
- Return nil if not found (not an error)
- Log internal errors
- Propagate DB errors

### Write Methods
- Validate inputs (userID, IDs required)
- Return error if target not found
- Log errors for debugging

### Validation
- UserID cannot be empty
- Content/EntryType required for reflections
- Status required for goal updates

## Confidence Score Calculation

```
For each profile component with confidence values:
  average = sum(confidences) / count
  min = minimum confidence
  max = maximum confidence
  count = number of items

Score by category:
  - patterns: average of all pattern confidence scores
  - goals: average of all goal confidence scores
  - learnings: average of all learning confidence scores
  - aboutMe: single confidence value
```

## Performance Considerations

### Query Complexity
- GetUserProfile performs 7 queries (can be parallelized)
- GetContacts + getContactPatterns = N+1 pattern
- Suitable for initial load, consider caching for frequent queries

### Optimization Opportunities
1. Batch contact pattern queries (SELECT WHERE contact_id IN (...))
2. Cache confidence stats (only changes on profile update)
3. Add pagination for large result sets (many contacts/reflections)
4. Index queries by user_id and timestamp

## Tests (16 test cases)

| Test | Verifies |
|------|----------|
| GetUserProfileEmpty | Returns empty profile for new user |
| GetAboutMe | Retrieves AboutMe or nil |
| GetContacts | Retrieves contact list |
| GetPatterns | Retrieves patterns ordered by observation count |
| GetGoals | Retrieves goals ordered by date |
| GetLearnings | Retrieves learnings with confirmation status |
| GetReflections | Retrieves journal entries ordered by date |
| AddReflection | Creates new reflection entry |
| AddReflectionValidation | Validates required fields |
| ConfirmLearning | Marks learning as confirmed |
| RejectLearning | Marks learning as rejected |
| UpdateGoalProgress | Updates goal progress and status |
| CalculateConfidenceStats | Correctly calculates min/max/average |
| GetUserProfileMultiUser | Handles multiple users independently |
| ProfileStructure | Verifies structure and initialization |
| ValidateUserID | Rejects empty userID |

**Run tests**:
```bash
go test -v ./services -run ProfileServiceGate
```

## Integration Points

### Receives From
- HTTP Handlers (userID from requests)
- External callers (browser extension, CLI, etc.)

### Sends To
- HTTP Handlers (returns UserProfile JSON)
- Database (queries)

### Dependencies
- Database with Phase 1.2 schema
- No external dependencies

## Usage Example

```go
// Initialize
service := NewProfileService(db)

// Get complete profile
profile, err := service.GetUserProfile("user123")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User has %d patterns, %d goals\n",
    len(profile.Patterns), len(profile.Goals))

// Get specific component
patterns, _ := service.GetPatterns("user123")
for _, p := range patterns {
    fmt.Printf("%s (observed %d times, confidence %.1f%%)\n",
        p.Pattern, p.ObservationCount, p.Confidence*100)
}

// Add user reflection
id, _ := service.AddReflection(
    "user123",
    "Realized I tend to avoid conflict",
    "breakthrough",
    []string{"communication", "growth"},
    nil)
fmt.Printf("Added reflection %d\n", id)

// Confirm learning
service.ConfirmLearning("user123", 42)

// Update goal progress
service.UpdateGoalProgress("user123", 10, "Had assertive conversation with boss", "active")
```

## Confidence Scores

Confidence values are normalized (0-1 range):
- **0.0 - 0.3**: Low confidence (uncertain)
- **0.3 - 0.6**: Medium confidence (observed pattern)
- **0.6 - 0.9**: High confidence (repeated observation)
- **0.9 - 1.0**: Very high confidence (strongly reinforced)

Profile exposes statistics per category:
```json
{
  "confidenceScores": {
    "patterns": {
      "average": 0.75,
      "count": 12,
      "min": 0.6,
      "max": 0.95
    },
    "goals": {
      "average": 0.82,
      "count": 3,
      "min": 0.8,
      "max": 0.85
    }
  }
}
```

## Files

- `services/profile_service.go` - Implementation (390 lines)
- `services/profile_service_test.go` - Tests (360+ lines)
- `handlers/v2_1_profile_handlers.go` - HTTP handlers (350+ lines)
- `handlers/v2_1_profile_handlers_test.go` - Handler tests (420+ lines)

## Next: HTTP Integration

Handlers are ready and registered:
```
GET    /api/v2.1/profile
GET    /api/v2.1/profile/about-me
GET    /api/v2.1/profile/contacts
GET    /api/v2.1/profile/patterns
GET    /api/v2.1/profile/goals
GET    /api/v2.1/profile/learnings
GET    /api/v2.1/profile/reflections
POST   /api/v2.1/reflections
POST   /api/v2.1/learnings/{id}/confirm
POST   /api/v2.1/learnings/{id}/reject
PATCH  /api/v2.1/goals/{id}
```

All endpoints:
- Require `X-User-ID` header for auth
- Return JSON responses with data/message structure
- Handle missing data gracefully (nil → empty list)
- Validate input and return 400 on validation errors
