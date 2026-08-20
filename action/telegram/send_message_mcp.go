package telegram

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"personal/gateways"
)

var SendTelegramMessageMCPDefinition = mcp.Tool{
	Name: "send_telegram_message",
	Annotations: &mcp.ToolAnnotations{
		Title: "Send Telegram message",
	},
	Description: `Send a text message to the user's Telegram chat, outside of an active conversation turn.

Use this tool when you need to reach the user asynchronously — there is no reply channel back through this tool, so only use it for things that don't need an immediate answer:
- A background task finished and the result is ready
- A reminder or scheduled check-in is due
- Something needs the user's attention between sessions

Required input:
- text: the message body to send. The message always renders as Telegram Markdown, so if you want rich formatting (bold, italic, code, links) write standard Markdown syntax directly in text — do not send plain text expecting it to render literally, and escape any literal _*[]() characters that aren't meant as Markdown.

Effects:
- Delivers immediately to the single Telegram chat configured for this app (not per-user, not selectable per call)
- Not idempotent — calling it twice sends the message twice

Example conversation:
User: "Let me know when the import finishes, I'm stepping away."
[... background task completes later ...]
[Call send_telegram_message(text="*Import finished*: 42 transactions added.")]`,
}

// SendTelegramMessageInput is the MCP tool input.
type SendTelegramMessageInput struct {
	Text string `json:"text" jsonschema:"The message text to send, formatted as Telegram Markdown"`
}

// SendTelegramMessageOutput is the MCP tool output.
type SendTelegramMessageOutput struct {
	MessageID int64 `json:"message_id" jsonschema:"Telegram ID of the sent message"`
}

func SendTelegramMessage(ctx context.Context, _ *mcp.CallToolRequest, input SendTelegramMessageInput) (*mcp.CallToolResult, SendTelegramMessageOutput, error) {
	if input.Text == "" {
		return nil, SendTelegramMessageOutput{}, fmt.Errorf("text must not be empty")
	}

	tg := gateways.TelegramFromContext(ctx)
	if tg == nil {
		return nil, SendTelegramMessageOutput{}, fmt.Errorf("telegram gateway not available in context")
	}

	messageID, err := tg.SendMessage(ctx, input.Text, "Markdown")
	if err != nil {
		return nil, SendTelegramMessageOutput{}, fmt.Errorf("telegram error: %w", err)
	}

	return nil, SendTelegramMessageOutput{MessageID: messageID}, nil
}
