package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr, DatabasePath, BotToken, MiniAppURL, SessionSecret, BotWebhookSecret, AllowedOrigin string
	BotUpdateMode                                                                            string
	AllowedTelegramID                                                                        int64
	SessionTTL, TelegramAuthMaxAge                                                           time.Duration
}

func Load() (Config, error) {
	c := Config{Addr: env("APP_ADDR", ":8080"), DatabasePath: env("DATABASE_PATH", "./data/pool-tracker.db"), BotToken: os.Getenv("BOT_TOKEN"), MiniAppURL: os.Getenv("MINI_APP_URL"), SessionSecret: os.Getenv("SESSION_SECRET"), SessionTTL: duration("SESSION_TTL", 7*24*time.Hour), TelegramAuthMaxAge: duration("TELEGRAM_AUTH_MAX_AGE", 24*time.Hour)}
	c.BotWebhookSecret = os.Getenv("BOT_WEBHOOK_SECRET")
	c.AllowedOrigin = os.Getenv("ALLOWED_ORIGIN")
	c.BotUpdateMode = env("BOT_UPDATE_MODE", "webhook")
	if c.BotUpdateMode != "webhook" && c.BotUpdateMode != "polling" {
		return c, fmt.Errorf("BOT_UPDATE_MODE must be webhook or polling")
	}
	if raw := os.Getenv("ALLOWED_TELEGRAM_ID"); raw != "" {
		var err error
		c.AllowedTelegramID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c, fmt.Errorf("parse ALLOWED_TELEGRAM_ID: %w", err)
		}
	}
	return c, nil
}
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(key)); err == nil && v != 0 {
		return v
	}
	return fallback
}
