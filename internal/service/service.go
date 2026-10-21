package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/andrey/pool-tracker/internal/domain"
	"gorm.io/gorm"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("session already exists")
	ErrInvalid   = errors.New("invalid input")
)

type Service struct {
	db         *gorm.DB
	sessionTTL time.Duration
}

func New(db *gorm.DB, sessionTTL time.Duration) *Service {
	return &Service{db: db, sessionTTL: sessionTTL}
}
func id() string               { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func token() string            { b := make([]byte, 32); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func hash(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }
func (s *Service) FindOrCreateUser(telegramID int64) (domain.User, error) {
	var u domain.User
	err := s.db.Where("telegram_id = ?", telegramID).First(&u).Error
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return u, err
	}
	u = domain.User{ID: id(), TelegramID: telegramID}
	if err = s.db.Create(&u).Error; err != nil {
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}
func (s *Service) CreateAuthSession(userID string) (string, error) {
	raw := token()
	session := domain.AuthSession{ID: id(), UserID: userID, TokenHash: hash(raw), ExpiresAt: time.Now().Add(s.sessionTTL)}
	return raw, s.db.Create(&session).Error
}
func (s *Service) UserByToken(raw string) (domain.User, error) {
	var session domain.AuthSession
	err := s.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", hash(raw), time.Now()).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	var u domain.User
	if err = s.db.First(&u, "id = ?", session.UserID).Error; err != nil {
		return u, err
	}
	return u, nil
}
func (s *Service) Logout(raw string) error {
	return s.db.Model(&domain.AuthSession{}).Where("token_hash = ? AND revoked_at IS NULL", hash(raw)).Update("revoked_at", time.Now()).Error
}
func validDate(date string) bool { _, err := time.Parse("2006-01-02", date); return err == nil }
func validate(date string, minutes int) error {
	if !validDate(date) || minutes < 1 || minutes > 1440 {
		return ErrInvalid
	}
	return nil
}
func (s *Service) List(userID, from, to string) ([]domain.GameSession, error) {
	if !validDate(from) || !validDate(to) || from > to {
		return nil, ErrInvalid
	}
	var rows []domain.GameSession
	return rows, s.db.Where("user_id = ? AND game_date >= ? AND game_date <= ?", userID, from, to).Order("game_date").Find(&rows).Error
}
func (s *Service) Create(userID, date string, minutes int, note string) (domain.GameSession, error) {
	if err := validate(date, minutes); err != nil {
		return domain.GameSession{}, err
	}
	row := domain.GameSession{ID: id(), UserID: userID, GameDate: date, DurationMinutes: minutes, Note: note}
	err := s.db.Create(&row).Error
	if err != nil && stringsContains(err.Error(), "UNIQUE") {
		return row, ErrDuplicate
	}
	return row, err
}
func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
func (s *Service) Get(userID, sessionID string) (domain.GameSession, error) {
	var row domain.GameSession
	err := s.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, ErrNotFound
	}
	return row, err
}
func (s *Service) Update(userID, sessionID, date string, minutes int, note string) (domain.GameSession, error) {
	if err := validate(date, minutes); err != nil {
		return domain.GameSession{}, err
	}
	row, err := s.Get(userID, sessionID)
	if err != nil {
		return row, err
	}
	err = s.db.Model(&row).Updates(map[string]any{"game_date": date, "duration_minutes": minutes, "note": note}).Error
	if err != nil && stringsContains(err.Error(), "UNIQUE") {
		return row, ErrDuplicate
	}
	return s.Get(userID, sessionID)
}
func (s *Service) Delete(userID, sessionID string) error {
	result := s.db.Where("id = ? AND user_id = ?", sessionID, userID).Delete(&domain.GameSession{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
func (s *Service) Statistics(userID, from, to string) (domain.Statistics, error) {
	if !validDate(from) || !validDate(to) || from > to {
		return domain.Statistics{}, ErrInvalid
	}
	var stat domain.Statistics
	err := s.db.Model(&domain.GameSession{}).Select("COUNT(*) AS playing_days, COALESCE(SUM(duration_minutes),0) AS total_minutes, COALESCE(AVG(duration_minutes),0) AS average_minutes, COALESCE(MAX(duration_minutes),0) AS longest_minutes").Where("user_id = ? AND game_date >= ? AND game_date <= ?", userID, from, to).Scan(&stat).Error
	return stat, err
}
