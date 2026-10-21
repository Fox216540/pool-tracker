package domain

import "time"

type User struct {
	ID         string `gorm:"primaryKey"`
	TelegramID int64  `gorm:"uniqueIndex"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (User) TableName() string { return "users" }

type GameSession struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	UserID          string    `gorm:"index" json:"-"`
	GameDate        string    `gorm:"column:game_date" json:"game_date"`
	DurationMinutes int       `gorm:"column:duration_minutes" json:"duration_minutes"`
	Note            string    `json:"note"`
	CreatedAt       time.Time `json:"-"`
	UpdatedAt       time.Time `json:"-"`
}

func (GameSession) TableName() string { return "game_sessions" }

type AuthSession struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	TokenHash string `gorm:"uniqueIndex"`
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

func (AuthSession) TableName() string { return "auth_sessions" }

type Statistics struct {
	PlayingDays    int `json:"playingDays"`
	TotalMinutes   int `json:"totalMinutes"`
	AverageMinutes int `json:"averageMinutes"`
	LongestMinutes int `json:"longestMinutes"`
}
