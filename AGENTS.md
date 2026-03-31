# Agent Behavior Guidelines

This document outlines the expected behavior and operational guidelines for agents working in this project.

## Core Principles

### 1. Skill Utilization

- **Always leverage available skills** when they match the task at hand
- Skills provide specialized capabilities and domain knowledge
- Check available skills before implementing solutions manually
- Prefer skill-based approaches over manual implementation when applicable

### 2. Step-by-Step Planning

- **Break down complex tasks** into clear, manageable steps
- Create structured implementation plans for non-trivial work
- Document the approach before executing
- Use the session workspace (`plan.md`) for planning when needed
- Track progress using the SQL todo system for multi-step tasks

### 3. User Confirmation

- **Request user confirmation** before executing non-trivial tasks
- Non-trivial tasks include:
  - Making significant code changes that affect core functionality
  - Deleting or modifying multiple files
  - Implementing new features or major refactors
  - Changes that could impact production behavior
  - Installing new dependencies or tools
  - Making architectural decisions
- Provide a clear summary of what will be done and ask for approval
- For trivial tasks (typos, minor fixes, single-line changes), proceed without confirmation

### 4. Self-Documentation

- **Expand this AGENTS.md file** when:
  - New patterns or best practices emerge
  - Project-specific conventions are established
  - Common pitfalls or gotchas are discovered
  - New workflows or processes are introduced
- Keep this document up-to-date as the living source of truth for agent behavior
- Document lessons learned and refinements to the development process

## Workflow Best Practices

### Before Starting Work

1. Understand the full scope of the request
2. Check for available skills that could help
3. For non-trivial tasks: create a plan and get user confirmation
4. Identify dependencies and potential impacts

### During Execution

1. Follow the planned steps systematically
2. Update todo status as work progresses
3. Validate changes incrementally
4. Report progress clearly and concisely

### After Completion

1. Verify the expected outcome
2. Run relevant tests or validations
3. Clean up temporary files or artifacts
4. Summarize what was done

## Project-Specific Guidelines

### Tech Stack

- **Language:** Go 1.22+
- **Web framework:** Gin (HTTP routes + HTML template rendering)
- **Database:** SQLite via `mattn/go-sqlite3` (WAL mode)
- **SQL code generation:** sqlc — schema in `internal/db/migrations/`, queries in `internal/db/queries/`, generated code in `internal/db/sqlc/`
- **WebSocket:** `gorilla/websocket` for live result broadcasting
- **Frontend:** Server-side rendered Go HTML templates + vanilla JS
- **Deployment:** Docker Compose on Hetzner VPS

### Project Structure

```
cmd/server/main.go           — Entry point (run() pattern to avoid exitAfterDefer)
internal/config/              — YAML config loading
internal/db/migrations/       — SQL schema (CREATE TABLE statements)
internal/db/queries/          — sqlc query definitions (annotated SQL)
internal/db/sqlc/             — Generated code (DO NOT EDIT manually)
internal/handlers/            — Gin HTTP handlers (vote, results, admin)
internal/websocket/           — WebSocket hub for broadcasting updates
templates/                    — Go HTML templates
static/css/                   — Stylesheets
static/js/                    — Client-side JavaScript
photos/                       — Exhibition photo thumbnails (not committed)
config.yaml                   — Exhibition configuration
```

### Development

#### Building

```bash
go build ./cmd/server/
```

#### Linting

Two linters are configured and must pass before committing:

```bash
# Go linting (golangci-lint v2)
golangci-lint run ./...

# Frontend/config formatting (prettier)
prettier --check "templates/**/*.html" "static/**/*.{css,js}" "*.{yaml,yml,json,md}"

# Auto-fix prettier issues
prettier --write "templates/**/*.html" "static/**/*.{css,js}" "*.{yaml,yml,json,md}"
```

#### Regenerating sqlc Code

After modifying `internal/db/migrations/*.sql` or `internal/db/queries/*.sql`:

```bash
sqlc generate
```

The generated files in `internal/db/sqlc/` are excluded from golangci-lint.

### Conventions

- **main.go pattern:** Use a `run() error` function called from `main()` to allow proper `defer` cleanup and avoid `exitAfterDefer` lint warnings
- **Error handling:** Return errors up the stack; only `log.Fatal` in `main()`
- **Random numbers:** Use `crypto/rand` (not `math/rand`) for security-sensitive operations like the prize draw
- **File reads:** Unchecked `os.ReadFile` on config paths is acceptable (G304 suppressed in golangci-lint) since paths come from trusted config
- **WebSocket:** Always discard error returns from `conn.Close()` with `_ =` to satisfy errcheck
- **Cookie sessions:** HttpOnly + SameSite=Strict, 30-day expiry
- **CSS:** Dark theme with CSS custom properties (variables), mobile-first responsive design

### Gotchas

- `golangci-lint` v2 uses `linters.settings` (not `linters-settings`) for per-linter config
- `gosimple` and `typecheck` are not standalone linters in v2 — they're part of `staticcheck`
- SQLite requires CGo (`CGO_ENABLED=1`) — the Dockerfile uses `gcc` + `musl-dev` in the build stage
- The `internal/db/sqlc/` directory is excluded from linting since it contains generated code

---

_This document is maintained by agents and should be updated whenever new behavioral patterns or guidelines are established._
