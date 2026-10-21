package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andrey/pool-tracker/internal/service"
)

type Config struct {
	Token, MiniAppURL string
	AllowedTelegramID int64
}
type Bot struct {
	config  Config
	service *service.Service
	client  *http.Client
}

func New(c Config, service *service.Service) *Bot {
	return &Bot{config: c, service: service, client: &http.Client{Timeout: 10 * time.Second}}
}

type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}
type Message struct {
	Chat Chat   `json:"chat"`
	From User   `json:"from"`
	Text string `json:"text"`
}
type Chat struct {
	ID int64 `json:"id"`
}
type User struct {
	ID int64 `json:"id"`
}

func (b *Bot) Handle(ctx context.Context, update Update) error {
	if update.Message == nil || (b.config.AllowedTelegramID != 0 && update.Message.From.ID != b.config.AllowedTelegramID) {
		return nil
	}
	msg := update.Message
	switch msg.Text {
	case "/start":
		return b.send(ctx, msg.Chat.ID, "Pool Tracker — учёт времени игры в бильярд.", true)
	case "/calendar":
		return b.send(ctx, msg.Chat.ID, "Откройте календарь:", true)
	case "/help":
		return b.send(ctx, msg.Chat.ID, "Используйте календарь Mini App, чтобы добавить или изменить игровую сессию. /stats покажет статистику за месяц.", false)
	case "/stats":
		u, err := b.service.FindOrCreateUser(msg.From.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		to := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		stat, err := b.service.Statistics(u.ID, from, to)
		if err != nil {
			return err
		}
		return b.send(ctx, msg.Chat.ID, fmt.Sprintf("За текущий месяц: %d игровых дней, %d мин., в среднем %d мин.", stat.PlayingDays, stat.TotalMinutes, stat.AverageMinutes), true)
	}
	return nil
}

func (b *Bot) RunPolling(ctx context.Context) error {
	if b.config.Token == "" {
		return fmt.Errorf("BOT_TOKEN is required for polling")
	}
	if err := b.call(ctx, "deleteWebhook", url.Values{"drop_pending_updates": {"false"}}, nil); err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	var offset int
	for ctx.Err() == nil {
		var updates []Update
		values := url.Values{"timeout": {"25"}, "allowed_updates": {`["message"]`}}
		if offset > 0 {
			values.Set("offset", fmt.Sprint(offset))
		}
		if err := b.call(ctx, "getUpdates", values, &updates); err != nil {
			if ctx.Err() != nil {
				break
			}
			return fmt.Errorf("get updates: %w", err)
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if err := b.Handle(ctx, update); err != nil {
				return fmt.Errorf("handle update %d: %w", update.UpdateID, err)
			}
		}
	}
	return ctx.Err()
}

func (b *Bot) call(ctx context.Context, method string, values url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+b.config.Token+"/"+method, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return err
	}
	if response.StatusCode/100 != 2 || !envelope.OK {
		return fmt.Errorf("telegram %s: %s", method, envelope.Description)
	}
	if result != nil {
		return json.Unmarshal(envelope.Result, result)
	}
	return nil
}
func (b *Bot) send(ctx context.Context, chatID int64, text string, appButton bool) error {
	if b.config.Token == "" {
		return nil
	}
	body := map[string]any{"chat_id": chatID, "text": text}
	if appButton && b.config.MiniAppURL != "" {
		body["reply_markup"] = map[string]any{"inline_keyboard": [][]any{{map[string]any{"text": "Открыть календарь", "web_app": map[string]string{"url": b.config.MiniAppURL}}}}}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+b.config.Token+"/sendMessage", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("telegram sendMessage: %s", resp.Status)
	}
	return nil
}
