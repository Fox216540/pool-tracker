-- +goose Up
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    telegram_id INTEGER NOT NULL UNIQUE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
CREATE TABLE game_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_date TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL CHECK(duration_minutes > 0 AND duration_minutes <= 1440),
    note TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE(user_id, game_date)
);
CREATE INDEX idx_game_sessions_user_date ON game_sessions(user_id, game_date);
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    revoked_at DATETIME
);
CREATE INDEX idx_auth_sessions_token ON auth_sessions(token_hash);

-- +goose Down
DROP TABLE auth_sessions;
DROP TABLE game_sessions;
DROP TABLE users;
