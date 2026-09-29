package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// client is a minimal Telegram Bot API client for one chat.
type client struct {
	http   *http.Client
	token  string
	chatID string
}

// apiError is a Bot API failure response.
type apiError struct {
	Code        int    `json:"error_code"`
	Description string `json:"description"`
}

func (e *apiError) Error() string { return fmt.Sprintf("%d %s", e.Code, e.Description) }

// call POSTs form to method and decodes the result into result (may be nil).
func (c *client) call(method string, form url.Values, result any) error {
	form.Set("chat_id", c.chatID)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+c.token+"/"+method, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var body struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		apiError
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	if !body.OK {
		return &body.apiError
	}
	if result != nil {
		return json.Unmarshal(body.Result, result)
	}
	return nil
}

func htmlMessage(text string) url.Values {
	return url.Values{"text": {text}, "parse_mode": {"HTML"}, "disable_web_page_preview": {"true"}}
}

// send posts a message, optionally as a reply, and returns its id (0 on failure).
func (c *client) send(text string, replyTo int) int {
	form := htmlMessage(text)
	if replyTo != 0 {
		form.Set("reply_to_message_id", strconv.Itoa(replyTo))
	}
	var msg struct {
		MessageID int `json:"message_id"`
	}
	if err := c.call("sendMessage", form, &msg); err != nil {
		slog.Warn("telegram send failed", "err", err)
		return 0
	}
	return msg.MessageID
}

// editResult is the outcome of an edit.
type editResult int

const (
	editOK     editResult = iota
	editGone              // message deleted — safe to send a replacement
	editFailed            // transient error — retry next cycle
)

func (c *client) edit(msgID int, text string) editResult {
	form := htmlMessage(text)
	form.Set("message_id", strconv.Itoa(msgID))
	err := c.call("editMessageText", form, nil)
	if err == nil {
		return editOK
	}
	if e, ok := err.(*apiError); ok && e.Code == 400 {
		switch {
		case strings.Contains(e.Description, "not modified"):
			return editOK
		case strings.Contains(e.Description, "message to edit not found"), strings.Contains(e.Description, "MESSAGE_ID_INVALID"):
			return editGone
		}
	}
	slog.Warn("telegram edit failed", "msg", msgID, "err", err)
	return editFailed
}

func (c *client) pin(msgID int) {
	form := url.Values{"message_id": {strconv.Itoa(msgID)}, "disable_notification": {"true"}}
	if err := c.call("pinChatMessage", form, nil); err != nil {
		slog.Warn("telegram pin failed", "err", err)
	}
}
