# AGENTS.md - Best Habit Backend Guide

This document provides context, conventions, and patterns for AI agents working on the "Best Habit" backend project.

## 1. Project Context
"Best Habit" is a Golang-based backend service for a mobile habit-tracking application. It features habit management, tasks, challenges, and gamification elements.
- **Primary Language**: Go (Golang)
- **Framework**: Gin Web Framework
- **Database**: MySQL (using `sqlx` and standard `sql/driver` patterns)
- **Architecture**: Modular (Transport - Business - Storage)

## 2. Key Architecture Patterns

### Module Structure
Code is organized by domain in `modules/`. Each module typically contains:
- **`...model`**: Structs for DB entities (`SQLModel`) and JSON requests/responses. Use `json` tags.
- **`...storage`**: DB operations. Methods should accept `context.Context` and `*sqlx.DB`.
- **`...biz`**: Business logic. Validates input and orchestrates storage calls.
- **`...transport/gin...`**: HTTP Handlers. Decodes requests and calls business logic.

### Database & Models
- **Base Model**: All entities should embed `common.SQLModel`.
  - Includes: `Id`, `CreatedAt`, `UpdatedAt`.
  - **UIDs**: We use `FakeID` (UID) to obscure real integer IDs in JSON responses.
    - DB Struct: `Id int`
    - JSON Struct: `FakeID *common.UID `
  - Use `ProcessUID()` helpers (if available) or manual `GenUID()` calls when returning data.

### Request/Response
- **Success**: Use `common.SimpleSuccessResponse(data)` or `common.NewSuccessResponse(...)`.
- **Errors**: Return `*common.AppError`.
  - Use helpers: `common.ErrDB(err)`, `common.ErrInvalidRequest(err)`, `common.ErrEntityNotFound(...)`.
  - Do not verify plain `error` values; verify against `common.AppError` when handling returns.

### Configuration
- **Env Vars**: Loaded via `godotenv`.
- **AppContext**: `component.AppContext` is passed to all handlers/biz logic to access DB, Auth, Uploaders, etc.

## 3. Coding Conventions

- **Variable Naming**: camelCase.
- **Functions**: PascalCase for exported, camelCase for internal.
- **Context**: Always propagate `context.Context` (usually `ctx`).
- **Imports**: modifying `go.mod` is allowed but check specific version requirements if any.
- **JSON**: heavily used for complex data (`avatar`, `settings`, `days`). Ensure structs have proper `json:"..."` tags.

## 4. Common Tasks & Workflows

### Adding a New API Endpoint
1.  **Define Model**: Create structs in `modules/[name]/[name]model/`. Embed `common.SQLModel`.
2.  **Storage Layer**: Implement CRUD in `modules/[name]/[name]storage/`.
3.  **Business Layer**: Implement logic in `modules/[name]/[name]biz/`.
4.  **Transport Layer**: Create Handler in `modules/[name]/[name]transport/gin[name]/`.
5.  **Router**: Register the route in `main.go` under the appropriate group (`/api/...`).

### Notifications
- Use `pubsub` for side effects (e.g., `common.TopicUserCreateNewTask`).
- See `subscriber/` for examples of handling these events.

## 5. Deployment
- **Docker**: `Dockerfile` is present.
- **Run Locally**: `go run main.go`.

---
**Note**: When generating code, always check `common/` for reusable utilities before writing from scratch.
