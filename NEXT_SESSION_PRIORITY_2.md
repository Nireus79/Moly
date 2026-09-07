# Priority 2: Load Context from Database (Next Session)

**Status**: Database now wired to agents ✅  
**Time**: 1.5-2 hours

---

## What to Implement

### File: `moly-go/agents/context_manager.go`

Replace all 10 TODOs with real database calls:

#### 1. GetAboutMe (Line 32)
```go
// BEFORE: TODO: Load from database
// AFTER:
func (cm *contextManager) GetAboutMe(userID string) (*models.AboutMe, error) {
    if userID == "" || cm.db == nil {
        return &models.AboutMe{UserID: userID}, nil // Return empty if no db
    }
    db := cm.db.(*sql.DB)
    // Query: SELECT * FROM about_me WHERE user_id = ?
    // Parse and return
}
```

#### 2. SetAboutMe (Line 50)
```go
// Store About Me profile to database
// SQL: INSERT OR REPLACE INTO about_me (user_id, communication_style, values, ...)
```

#### 3. GetContact (Line 60)
```go
// Load specific contact from database
// SQL: SELECT * FROM contacts WHERE user_id = ? AND id = ?
```

#### 4. GetContacts (Line 75)
```go
// Load all contacts for user from database
// SQL: SELECT * FROM contacts WHERE user_id = ?
// Return []models.Contact
```

#### 5. CreateContact (Line 93)
```go
// Save new contact to database
// SQL: INSERT INTO contacts (user_id, name, relationship, ...)
```

#### 6. UpdateContact (Line 103)
```go
// Update existing contact
// SQL: UPDATE contacts SET ... WHERE user_id = ? AND id = ?
```

#### 7. GetRelevantContext (Line 113)
```go
// MOST IMPORTANT - Load everything needed for suggestion generation:
// 1. Query AboutMe
// 2. Query Contacts (filter by conversation if provided)
// 3. Query conversation history
// 4. Query user behavioral profile
// 5. Query relevant reflections
// Return populated models.Context
```

#### 8-10. Reflection methods (Lines 142, 159, 177)
```go
// SaveReflection: INSERT INTO reflections
// ApproveReflection: UPDATE reflections SET status='approved'
// AppendMessage: INSERT INTO conversation_messages
```

---

## Database Access Pattern

Use type assertion (database already passed as interface{}):

```go
db := cm.db.(*sql.DB)
row := db.QueryRow("SELECT ... FROM about_me WHERE user_id = ?", cm.userID)
// Parse row into model
```

Reference existing database.go for:
- Table names: `about_me`, `contacts`, `conversations`, `messages`, `reflections`
- Column names and types
- Query examples from other handlers

---

## Testing After Implementation

1. Add test user data via extension UI
2. Send message through sidebar
3. Check logs show database queries executing
4. Verify suggestions reference loaded AboutMe/Contact
5. Verify reflection extraction captures user data

---

## After Priority 2

Move to Priority 3: Learning loop (record suggestion choices)
- File: `moly-go/agents/learning_agent.go`
- Implement RecordSuggestionChoice() to save to database
- Implement BuildUserProfile() to analyze patterns

---

**Build**: Will compile after adding methods (stubs OK for now)

