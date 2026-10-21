# Pool Tracker

Telegram Mini App для учёта времени игры в бильярд.

## Архитектура

- Go/Gin API, Telegram webhook и бот — один процесс.
- SQLite с goose-миграциями; доступ к записям изолирован по Telegram ID.
- Vue 3 Mini App передаёт Telegram `initData`; сервер проверяет подпись и создаёт HttpOnly-сессию.

## Локальный запуск

```bash
cp .env.example .env
# заполните BOT_TOKEN, MINI_APP_URL и SESSION_SECRET
GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/go-mod" go run ./cmd/api
cd frontend && npm install && npm run dev
```

API доступен на `http://localhost:8080`, Vite — на `http://localhost:5173`. Для реальной Telegram-авторизации Mini App должен быть открыт Telegram по HTTPS.

## Проверка

```bash
GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/go-mod" go test ./...
cd frontend && npm run build
docker compose config
```

## Миграции

При запуске API все миграции из `migrations/` применяются автоматически. Для чистой локальной БД удалите только локальный файл БД, затем запустите API снова.

## Telegram

1. Создайте бота через BotFather и укажите `BOT_TOKEN`.
2. Разместите приложение по HTTPS и задайте URL в `MINI_APP_URL`.
3. Настройте webhook `https://<домен>/telegram/webhook` с secret token из `BOT_WEBHOOK_SECRET`.
4. При необходимости задайте `ALLOWED_TELEGRAM_ID`, чтобы бот и API принимали только владельца.

Команды: `/start`, `/calendar`, `/stats`, `/help`.

Для локальной проверки без webhook задайте `BOT_UPDATE_MODE=polling` и перезапустите API. Polling сам отключает webhook и получает сообщения Telegram; перед VPS верните `BOT_UPDATE_MODE=webhook`.

## VPS: Caddy и Let's Encrypt

Перед запуском укажите домен и направьте его A/AAAA-запись на IP VPS. Порты `80` и `443` должны быть свободны и доступны из интернета — тогда Caddy сам выпустит и продлит сертификат Let's Encrypt.

```bash
cp .env.example .env
# заполните APP_DOMAIN, ACME_EMAIL, BOT_TOKEN, MINI_APP_URL,
# ALLOWED_ORIGIN, SESSION_SECRET и BOT_WEBHOOK_SECRET
docker compose -f docker-compose.yml up -d --build
```

После запуска задайте в BotFather URL Mini App `https://<APP_DOMAIN>` и настройте Telegram webhook на `https://<APP_DOMAIN>/telegram/webhook` с `BOT_WEBHOOK_SECRET`. SQLite хранится в Docker volume `pool_data` и не публикуется.

`docker-compose.yml` остаётся вариантом для локальной проверки или существующего reverse proxy; `docker-compose.caddy.yml` предназначен для отдельного VPS, где Caddy занимает порты 80/443.

## Резервное копирование

`deploy/scripts/backup.sh` использует SQLite `.backup`, поэтому запускайте его в окружении с `sqlite3` и смонтированным volume базы. Пример: `DATABASE_PATH=/data/pool-tracker.db BACKUP_DIR=/backups sh deploy/scripts/backup.sh`. Восстановление: остановите API, замените файл базы резервной копией, запустите API и проверьте `/health/ready`.

Ограничения MVP: один календарный сеанс на пользователя в день; webhook Telegram настраивается после появления домена.
