package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andrey/pool-tracker/internal/bot"
	"github.com/andrey/pool-tracker/internal/config"
	"github.com/andrey/pool-tracker/internal/database"
	"github.com/andrey/pool-tracker/internal/handler"
	"github.com/andrey/pool-tracker/internal/service"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(c.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close(db)
	if err := database.Migrate(db, filepath.Join("migrations")); err != nil {
		log.Fatal(err)
	}
	svc := service.New(db, c.SessionTTL)
	telegramBot := bot.New(bot.Config{Token: c.BotToken, MiniAppURL: c.MiniAppURL, AllowedTelegramID: c.AllowedTelegramID}, svc)
	api := handler.New(svc, c, telegramBot)
	server := &http.Server{Addr: c.Addr, Handler: handler.Router(api), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if c.BotUpdateMode == "polling" {
		go func() {
			if err := telegramBot.RunPolling(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("telegram polling stopped: %v", err)
			}
		}()
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
