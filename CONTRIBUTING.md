# CONTRIBUTING TO MOLY

Welcome to Moly! This guide explains our development philosophy and how to contribute code.

## Core Philosophy

**Moly is a thinking partner, not a controller.**

- User maintains autonomy
- System asks, doesn't command
- Context helps but never constrains
- Privacy is non-negotiable

Before writing code, read MOLY_VISION.md to understand what Moly is and what it isn't.

## Development Setup

See DEVELOPMENT.md for:
- Prerequisites (Go, Node, SQLite)
- Backend setup and build
- Frontend setup and build
- How to run locally
- Debugging tips

## Code Guidelines

### Go (Backend)

**Style:**
- Follow Go conventions: `gofmt`, `goimports`
- Package main: handlers and initialization only
- Other packages: reusable, testable components
- No globals except `db` connection pool and logger

**Testing:**
- Unit tests in `*_test.go` files
- Test names: `TestFunctionName` or `TestFunctionName_Scenario`
- Use table-driven tests for multiple cases
- Mock database queries, not real database

**Errors:**
- Return errors, don't panic
- Wrap errors with context: `fmt.Errorf("context: %w", err)`
- Log at info for normal flow, error for problems
- Use log.Printf with [V2] prefix for debugging

**Database:**
- Queries in database/queries.go (organized by table)
- Always filter by user_id for security
- Use prepared statements to prevent SQL injection
- Transaction handling in handlers, not agents

**Architecture:**
- Handlers: HTTP ↔ Agent interface, no business logic
- Agents: Extract facts, reason, generate output, no DB access
- Database: Queries only, no business logic
- Schema: Types only, validators

### TypeScript/React (Frontend)

**Style:**
- Use `const` and `let`, never `var`
- Arrow functions preferred
- Component names: PascalCase
- Hook names: useCamelCase
- Folder structure: `src/components/`, `src/hooks/`, `src/stores/`

**State Management:**
- Zustand for global state (useAuthStore, useAboutMeStore, etc.)
- React hooks (useState, useEffect) for local component state
- Don't prop-drill; use stores for shared state

**Components:**
- Functional components only
- Hooks instead of class methods
- Keep components small (~200 lines max)
- One responsibility per component

**API Calls:**
- Centralize in custom hooks (useAuth, useApi, etc.)
- Always include Authorization header
- Handle 401 (unauthorized) → redirect to login
- Show loading state while fetching
- Show error message on failure

**Testing:**
- Test user interactions, not implementation details
- Use React Testing Library (not Enzyme)
- Test names should describe what user sees/does

## Git Workflow

### Branch naming

```
feature/what-you-built        # New capability
fix/what-you-fixed            # Bug fix
docs/what-you-documented      # Documentation only
refactor/what-you-improved    # No new features
```

### Commit messages

```
<type>: <subject (under 50 chars)>

<body: explain what and why, not how>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Example:
```
feat: add email/password authentication

Replaces code-based login with secure email/password registration.
Implements Bcrypt password hashing and 24-hour session tokens.
Adds multi-user isolation via user_id filtering on all queries.
```

### Pull requests

1. Create feature branch from `main`
2. Make changes, commit
3. Push and create PR
4. Describe what changed and why
5. Link any related issues
6. Wait for review
7. Merge when approved

PR description template:
```
## What changed
Brief summary (1-2 sentences)

## Why
The problem this solves

## Testing
How to verify it works

## Checklist
- [ ] Code follows guidelines
- [ ] Tests pass
- [ ] No new warnings
- [ ] Documentation updated
```

## Testing

### Manual testing checklist

- [ ] Feature works in isolation
- [ ] Doesn't break existing features
- [ ] Error cases handled gracefully
- [ ] Data is properly isolated by user
- [ ] No sensitive data in logs

### Automated testing

```bash
# Backend
cd moly-go && go test ./...

# Frontend
cd moly-extension && npm test
```

## Reporting Issues

Use GitHub Issues with:
- Clear title describing the problem
- Steps to reproduce (if applicable)
- Expected behavior
- Actual behavior
- Environment (Go version, Node version, OS)

## Code Review

When reviewing others' code:
- Is it clear and maintainable?
- Are edge cases handled?
- Is the design consistent?
- Are there security issues?
- Does it match the architecture?

Be kind. Code review is about making code better, not judging people.

## Documentation

When adding a feature:
- Update API.md if you added an endpoint
- Update ARCHITECTURE.md if you changed how data flows
- Add code comments only for non-obvious WHY (not WHAT)
- Update DEVELOPMENT.md with new setup steps

## Performance

Before optimizing:
1. Measure (log times, profile)
2. Identify the bottleneck
3. Make minimal change
4. Measure again to confirm improvement

Prefer clarity over performance. Optimize only when it matters.

## Security

**Never:**
- Store passwords in plain text (use Bcrypt)
- Trust user input (validate everything)
- Log sensitive data (tokens, passwords)
- Use SQL string concatenation (use prepared statements)
- Skip authentication checks

**Always:**
- Filter queries by user_id
- Return 401 on auth failure, 403 on permission failure
- Handle errors gracefully (don't leak internals)
- Review code for injection vulnerabilities

## Common Mistakes

- ❌ Not filtering by user_id in database queries → data leaks
- ❌ Logging session tokens or passwords → security issue
- ❌ Forgetting to commit schema changes → build breaks
- ❌ Prop-drilling instead of using stores → hard to maintain
- ❌ Ignoring error cases → poor UX

## Questions?

- Architecture questions: See ARCHITECTURE.md
- API questions: See API.md
- Implementation questions: Check existing code and follow patterns
- Design questions: See MOLY_VISION.md

## Recognition

Contributors are recognized in:
- Git commit history
- Project README (if substantial)
- Release notes (for major features)

Thank you for contributing to Moly!
