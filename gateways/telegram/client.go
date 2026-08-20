package telegram

import (
	"context"
	"fmt"

	tele "gopkg.in/telebot.v3"
)

// Client sends outbound messages to a single, pre-configured Telegram chat.
// It never starts the bot's update poller — it is send-only.
type Client struct {
	bot    *tele.Bot
	chatID int64
}

// NewClient creates a Telegram client for the given bot token and destination chat ID.
func NewClient(botToken string, chatID int64) (*Client, error) {
	bot, err := tele.NewBot(tele.Settings{Token: botToken})
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return &Client{bot: bot, chatID: chatID}, nil
}

// SendMessage sends text to the configured chat. parseMode is "" (plain text),
// "Markdown", "MarkdownV2", or "HTML".
func (c *Client) SendMessage(_ context.Context, text string, parseMode string) (int64, error) {
	msg, err := c.bot.Send(tele.ChatID(c.chatID), text, &tele.SendOptions{ParseMode: parseMode})
	if err != nil {
		return 0, fmt.Errorf("send telegram message: %w", err)
	}

	return int64(msg.ID), nil
}
