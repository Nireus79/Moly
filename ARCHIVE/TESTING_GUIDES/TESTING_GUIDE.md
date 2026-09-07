# Moly Testing Guide

## Quick Start

### Run All Tests
```bash
cd ~/vs_projects/Moly/Moly/moly-go
go test -v ./...
node sidebar_test.js
```

### Run Specific Test Suite
```bash
# Configuration tests only
go test -v -run TestConfig

# Chat tests only
go test -v -run TestChat

# Handler tests only
go test -v -run TestHandle

# Ollama tests only
go test -v -run TestOllama
```

---

## Test Structure

### Go Tests (5 files, 35 tests)

#### config_test.go (5 tests)
Tests configuration management:
- Default values
- Save/load persistence
- First-run initialization
- Timestamp updates
- Runtime configuration changes

Run: `go test -v -run TestConfig`

#### chat_test.go (5 tests)
Tests chat functionality:
- Request validation
- Response structure
- Error responses
- Mode parameter handling (direct vs socratic)
- Provider routing logic

Run: `go test -v -run TestChat`

#### handlers_test.go (21 tests)
Tests all HTTP endpoints:
- GET /api/status
- GET /api/first-run-check
- GET/POST /api/settings
- GET /api/models/list
- POST /api/models/pull
- POST /api/models/remove
- POST /api/ollama/start
- POST /api/ollama/stop
- POST /api/chat
- GET /sidebar.html
- GET /

Tests error cases:
- Invalid JSON requests
- Missing required fields
- Wrong HTTP methods
- CORS headers

Run: `go test -v -run TestHandle`

#### ollama_test.go (4 tests)
Tests Ollama integration:
- Service availability checks
- Return type validation
- Function signatures
- Error handling
- Endpoint validation

Run: `go test -v -run TestOllama`

### JavaScript Tests (1 file, 70 tests)

#### sidebar_test.js (70 tests)
Tests UI and frontend:
- Configuration defaults
- Provider options
- Message handling
- API endpoints
- Settings persistence
- Error handling
- Chat data flow
- UI elements (6 elements verified)
- Ollama management
- Model management

Run: `node sidebar_test.js`

---

## Test Coverage Summary

| Category | Tests | Status |
|----------|-------|--------|
| Configuration | 5 | ✅ All Passing |
| Chat | 5 | ✅ All Passing |
| HTTP Handlers | 21 | ✅ All Passing |
| Ollama | 4 | ✅ All Passing |
| JavaScript | 70 | ✅ All Passing |
| **Total** | **105** | **✅ All Passing** |

---

## Features Verified

### Core Functionality
- ✅ Configuration management (create, read, update, save)
- ✅ HTTP request/response handling
- ✅ Chat message routing
- ✅ Mode parameter (direct vs socratic)
- ✅ Error handling
- ✅ JSON serialization/deserialization

### API Endpoints (11/11)
- ✅ /api/status
- ✅ /api/first-run-check
- ✅ /api/settings (GET/POST)
- ✅ /api/models/list
- ✅ /api/models/pull
- ✅ /api/models/remove
- ✅ /api/chat
- ✅ /api/ollama/start
- ✅ /api/ollama/stop
- ✅ /sidebar.html
- ✅ / (root)

### Provider Integration
- ✅ Local/Ollama provider
- ✅ Claude provider
- ✅ OpenAI provider
- ✅ Unknown provider fallback

### UI Components (6/6)
- ✅ Message input
- ✅ Model selector
- ✅ Mode selector
- ✅ Ollama status indicator
- ✅ Model list
- ✅ Message display area

### Settings & Data
- ✅ Model persistence
- ✅ Mode persistence
- ✅ API key handling
- ✅ First-run flow
- ✅ Configuration updates

---

## Running Tests During Development

### Watch Mode (for TDD)
```bash
# Linux/Mac - requires entr or similar
go test -v ./... -count=1 | entr -r go test -v ./...
```

### Coverage Analysis
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Benchmarks (if needed)
```bash
go test -bench=. -benchmem ./...
```

---

## Continuous Integration

The test suite is designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Run Go Tests
  run: cd moly-go && go test -v ./...

- name: Run JavaScript Tests
  run: node moly-go/sidebar_test.js
```

---

## Test Philosophy

### What We Test
1. **Input Validation** - Does the system reject invalid input?
2. **Happy Path** - Does the system work correctly with valid input?
3. **Error Cases** - How does the system handle errors?
4. **Data Flow** - Do parameters flow correctly through the system?
5. **Integration** - Do components work together correctly?

### What We Don't Test (by design)
- External service behavior (Ollama, Claude API, OpenAI API)
  - These would require mocking, which is out of scope for MVP
- Browser-specific behavior (handled by browser compatibility testing)
- Real chat interactions (would require actual LLM calls)

---

## Adding New Tests

### Adding a Go Test
1. Create a test file (e.g., `feature_test.go`)
2. Add test functions with `Test` prefix
3. Use `*testing.T` for assertions
4. Run: `go test -v -run TestFeatureName`

Example:
```go
func TestNewFeature(t *testing.T) {
    result := someFunction()
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Adding a JavaScript Test
1. Add a test suite in `sidebar_test.js`
2. Use `assert(condition, message)` for assertions
3. Run: `node sidebar_test.js`

Example:
```javascript
testSuite("My Feature", [
  () => {
    assert(condition, "Feature works correctly");
  },
]);
```

---

## Troubleshooting

### Go Tests Fail
```bash
# Clean build
go clean -testcache
go test -v ./...

# Check Go version
go version
```

### JavaScript Tests Fail
```bash
# Ensure Node is installed
node --version

# Run with debug output
node sidebar_test.js 2>&1
```

### Specific Test Fails
```bash
# Run just that test
go test -v -run TestName

# Get more verbose output
go test -v -run TestName -x
```

---

## Performance

Typical test execution times:
- Go tests: 10-15ms
- JavaScript tests: < 100ms
- Total: < 150ms

---

## Maintenance

### Keep Tests Updated When:
- Adding new API endpoints
- Changing configuration structure
- Adding new UI elements
- Modifying data flow
- Changing error handling

### Review Tests Quarterly For:
- Dead code (tests for removed features)
- Coverage gaps
- Performance regressions
- Clarity and maintainability

---

## Further Reading

- [Go Testing Package](https://golang.org/pkg/testing/)
- [Go Test Best Practices](https://golang.org/doc/effective_go#testing)
- [HTTP Testing in Go](https://golang.org/pkg/net/http/httptest/)

---

**Last Updated**: September 4, 2026  
**Status**: All tests passing ✅
