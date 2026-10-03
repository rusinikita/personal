# Telegram Notifications - Complete Specification

## Overview

MCP tool that lets the agent proactively send a text message to the user's Telegram chat, outside of an active conversation turn (e.g. reminders, background-task completion notices, questions that need attention between sessions). Outbound-only: the app calls the Telegram Bot API's `sendMessage` endpoint over plain HTTP. There is no inbound bot (no long polling / webhook, no bot commands) — receiving messages from Telegram is a separate, future backlog item.

## Architecture Diagrams

### Entity Relation Diagram

No new entities. No DB table is introduced — chat ID and bot token are configuration values (env vars), not stored data.

### C4 Context Diagram

```mermaid
graph TB
    Agent[AI Agent / MCP Client]

    subgraph "Personal App"
        MCP[MCP Server]
        TG[Telegram Gateway]
    end

    TGAPI[Telegram Bot API]
    User[User's Telegram App]

    Agent -->|send_telegram_message| MCP
    MCP -->|SendMessage| TG
    TG -->|POST sendMessage| TGAPI
    TGAPI -->|deliver| User

    style Agent fill:#e1f5ff
    style MCP fill:#ffe1e1
    style TG fill:#ffe1e1
    style TGAPI fill:#e1ffe1
```

## Database Schema

### SQL DDL

None. No migration file is added for this feature.

## Go Code Structure

### Domain Models

No new domain package needed — the MCP input/output structs (below) are sufficient, following the pattern of simple tools like `get_balance`.

### Repository Interface

New gateway interface, separate from `gateways.DB`, added to `gateways/interfaces.go`:

```go
// Telegram wraps the Telegram Bot API for outbound notifications.
type Telegram interface {
    // SendMessage sends text to the configured chat. parseMode is "" (plain text),
    // "Markdown", or "HTML" per the Telegram Bot API sendMessage parse_mode param.
    SendMessage(ctx context.Context, text string, parseMode string) (messageID int64, err error)
}
```

Context wiring added to `gateways/context.go`, mirroring `WithDB`/`DBFromContext`:

```go
func WithTelegram(ctx context.Context, tg Telegram) context.Context
func TelegramFromContext(ctx context.Context) Telegram
```

Implementation: `gateways/telegram/client.go` — a small struct wrapping a `*telebot.Bot` (from `gopkg.in/telebot.v3`) and the destination `chatID` (from env vars, passed in at construction in `main.go`). `SendMessage` calls `bot.Send(telebot.ChatID(chatID), text, opts)`, mapping `parseMode` to `telebot.SendOptions.ParseMode`. The bot is created via `telebot.NewBot(telebot.Settings{Token: botToken})` with no `Poller` configured and `bot.Start()` is never called, since this feature only sends.

## MCP Tools

### send_telegram_message

Sends a text message to the user's configured Telegram chat. Input is just `text` — `parse_mode` is not exposed to the agent; every message is sent with Telegram Markdown parse mode hard-coded in the handler. The tool description tells the agent to write Markdown syntax directly in `text` when it wants rich formatting, instead of offering a separate parameter. Returns the Telegram `message_id` of the sent message. Returns an error if the Telegram API call fails (e.g. invalid token, chat not found, malformed Markdown) — no automatic retry.

## Configuration

- **`TELEGRAM_BOT_TOKEN`**: bot token from BotFather, required — app fails to start if missing (same pattern as `DATABASE_URL`).
- **`TELEGRAM_CHAT_ID`**: destination chat ID, required — same fail-fast pattern.

## E2E Tests

In `tests/telegram_send_message_test.go`:

- `TestSendTelegramMessage_*`: success; markdown text; empty text; API error; telegram not configured

## Changelog

- **03-10-26** — migrated to the new spec template: removed Best Practices Applied and the sequence diagrams together with every reference to them; Configuration narrowed to environment variables; added E2E Tests; added Changelog (Architecture Diagrams, Configuration, E2E Tests, Changelog)
- **20-08-26** — initial version: `send_telegram_message` MCP tool
