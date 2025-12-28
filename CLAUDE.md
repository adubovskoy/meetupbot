# CLAUDE.md - Meetupbot Project Guide

## Project Overview

Meetupbot is a Telegram bot for event registration management. It handles user registrations, capacity tracking, QR code check-ins, and CSV exports. Built in Go with SQLite storage.

## Quick Commands

```bash
# Build
go build

# Run (requires .env or environment variables)
./meetupbot

# Run with inline config
BOT_TOKEN=xxx ADMIN_USERS=admin1 ./meetupbot
```

## Project Structure

```
meetupbot/
├── main.go          # Entry point, event loop, update routing
├── handlers.go      # Command handlers and business logic
├── repository.go    # SQLite database abstraction layer
├── config.go        # Configuration loading (.env and env vars)
├── dialog.go        # Dialog state machine for user input
├── models.go        # Data structures (User, Event, Registration)
├── middleware.go    # Admin authentication middleware
├── bot.db           # SQLite database (created at runtime)
├── .env.example     # Configuration template
└── README.md        # User documentation
```

## Architecture

- **Entry Point**: `main.go` initializes config, DB, and runs the Telegram update loop
- **Routing**: Updates are routed to handlers based on type (callback, command, message)
- **Handlers**: `handlers.go` contains all command implementations
- **Data Layer**: `repository.go` provides `Repository` interface for all DB operations
- **State Machine**: `dialog.go` manages multi-step input collection (name, email)
- **Auth**: `middleware.go` wraps admin-only handlers with permission checks

## Key Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `main.go:45-91` | Main event loop | Routes updates to handlers |
| `handlers.go:35-90` | Command router | `handleCommand()` |
| `handlers.go:92-150` | QR check-in | `handleImhere()` |
| `handlers.go:200-300` | Registration logic | `handleCallbackQuery()` |
| `handlers.go:400-500` | Dialog handling | `handleDialog()` |
| `repository.go:50-100` | User registration | `RegisterUser()`, `RemoveRegistration()` |
| `repository.go:200-250` | Event queries | `GetLatestEvent()`, `AddEvent()` |
| `config.go:30-80` | Config loading | `LoadConfig()` |
| `dialog.go:40-80` | State management | `SetState()`, `GetState()` |

## Bot Commands

**User Commands:**
- `/start` - Welcome message with registration button
- `/register` - Show registration button
- `/state` - Show available spots for current event

**Admin Commands** (requires username in `ADMIN_USERS`):
- `/addevent Name;YYYY-MM-DD;capacity` - Create new event
- `/qrcode` - Generate QR code for check-in
- `/export` - Export registrations as CSV

## Configuration

Environment variables (or `.env` file):

| Variable | Required | Description |
|----------|----------|-------------|
| `BOT_TOKEN` | Yes | Telegram Bot API token from @BotFather |
| `ADMIN_USERS` | No | Comma-separated admin usernames |
| `MANDATORY_FIELDS` | No | Fields to collect: `name`, `email`, or `name,email` |

## Database Schema

**SQLite tables** (auto-created on first run):

```sql
-- users table
telegram_id INTEGER, username TEXT, name TEXT, email TEXT,
event_id INTEGER, registration_date DATETIME,
registred INTEGER DEFAULT 0, visited INTEGER DEFAULT 0

-- events table
id INTEGER, name TEXT, date DATETIME, capacity INTEGER,
registration_count INTEGER DEFAULT 0, state TEXT DEFAULT 'active'
```

## Code Patterns

- **Repository Pattern**: All DB access through `Repository` interface
- **Middleware Pattern**: `AdminCheckMiddleware()` wraps protected handlers
- **State Machine**: `DialogManager` tracks user input state with mutex protection
- **Handler Pattern**: Each command has dedicated handler function

## Important Behaviors

1. **Single Active Event**: `/addevent` marks all existing events as "past"
2. **Capacity Enforcement**: Registration blocked when `registration_count >= capacity`
3. **Data Reuse**: Previous name/email values are reused for returning users
4. **QR Check-in**: Deep link `t.me/BotName?start=imhere` marks attendance
5. **Dialog States**: `NoDialog` → `WaitingForName` → `WaitingForEmail` → complete

## Dependencies

- `github.com/go-telegram-bot-api/telegram-bot-api` - Telegram API wrapper
- `github.com/mattn/go-sqlite3` - SQLite driver (requires CGO)
- `github.com/skip2/go-qrcode` - QR code generation

## Development Notes

- Go 1.24+ required
- No test files exist - manual testing required
- Dialog state is in-memory (lost on restart)
- CSV exports include UTF-8 BOM for Excel compatibility
- Temp files created in `/tmp/` for exports and QR codes
