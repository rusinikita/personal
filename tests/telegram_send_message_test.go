package tests

import (
	"context"
	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"personal/action/telegram"
	"personal/gateways"
)

// fakeTelegramClient is a test double for gateways.Telegram — it never calls
// the real Telegram Bot API, it just records what it was asked to send.
type fakeTelegramClient struct {
	sentText      string
	sentParseMode string
	nextMessageID int64
	err           error
}

func (f *fakeTelegramClient) SendMessage(_ context.Context, text string, parseMode string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.sentText = text
	f.sentParseMode = parseMode
	return f.nextMessageID, nil
}

func (s *IntegrationTestSuite) TestSendTelegramMessage_Success() {
	fake := &fakeTelegramClient{nextMessageID: 42}
	ctx := gateways.WithTelegram(s.Context(), fake)

	_, out, err := telegram.SendTelegramMessage(ctx, nil, telegram.SendTelegramMessageInput{
		Text: "Background task finished",
	})
	require.NoError(s.T(), err)

	assert.Equal(s.T(), int64(42), out.MessageID)
	assert.Equal(s.T(), "Background task finished", fake.sentText)
	assert.Equal(s.T(), "Markdown", fake.sentParseMode, "messages are always sent as Telegram Markdown")
}

func (s *IntegrationTestSuite) TestSendTelegramMessage_MarkdownText() {
	fake := &fakeTelegramClient{nextMessageID: 7}
	ctx := gateways.WithTelegram(s.Context(), fake)

	_, out, err := telegram.SendTelegramMessage(ctx, nil, telegram.SendTelegramMessageInput{
		Text: "*bold* reminder",
	})
	require.NoError(s.T(), err)

	assert.Equal(s.T(), int64(7), out.MessageID)
	assert.Equal(s.T(), "*bold* reminder", fake.sentText)
	assert.Equal(s.T(), "Markdown", fake.sentParseMode)
}

func (s *IntegrationTestSuite) TestSendTelegramMessage_EmptyText() {
	fake := &fakeTelegramClient{}
	ctx := gateways.WithTelegram(s.Context(), fake)

	_, _, err := telegram.SendTelegramMessage(ctx, nil, telegram.SendTelegramMessageInput{
		Text: "",
	})
	require.Error(s.T(), err)
	assert.Equal(s.T(), "", fake.sentText, "gateway must not be called for invalid input")
}

func (s *IntegrationTestSuite) TestSendTelegramMessage_APIError() {
	fake := &fakeTelegramClient{err: errors.New("telegram: chat not found")}
	ctx := gateways.WithTelegram(s.Context(), fake)

	_, _, err := telegram.SendTelegramMessage(ctx, nil, telegram.SendTelegramMessageInput{
		Text: "hello",
	})
	require.Error(s.T(), err)
}

func (s *IntegrationTestSuite) TestSendTelegramMessage_TelegramNotConfigured() {
	// No gateways.WithTelegram in context — handler must fail clearly rather
	// than panic, mirroring how GetBalance handles a missing DB.
	ctx := s.Context()

	_, _, err := telegram.SendTelegramMessage(ctx, nil, telegram.SendTelegramMessageInput{
		Text: "hello",
	})
	require.Error(s.T(), err)
}
