package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TelegramUser struct {
	ID int64 `json:"id"`
}
type VerifiedTelegramData struct {
	User     TelegramUser
	AuthDate time.Time
}

func VerifyTelegramInitData(raw, botToken string, maxAge time.Duration, now time.Time) (VerifiedTelegramData, error) {
	if botToken == "" {
		return VerifiedTelegramData{}, fmt.Errorf("telegram authentication is not configured")
	}
	v, err := url.ParseQuery(raw)
	if err != nil {
		return VerifiedTelegramData{}, fmt.Errorf("parse init data: %w", err)
	}
	hash := v.Get("hash")
	if hash == "" {
		return VerifiedTelegramData{}, fmt.Errorf("missing init-data hash")
	}
	v.Del("hash")
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+v.Get(k))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	signature := hmac.New(sha256.New, secret.Sum(nil))
	signature.Write([]byte(strings.Join(parts, "\n")))
	expected, err := hex.DecodeString(hash)
	if err != nil || !hmac.Equal(expected, signature.Sum(nil)) {
		return VerifiedTelegramData{}, fmt.Errorf("invalid init-data signature")
	}
	authUnix, err := strconv.ParseInt(v.Get("auth_date"), 10, 64)
	if err != nil {
		return VerifiedTelegramData{}, fmt.Errorf("invalid auth_date")
	}
	authDate := time.Unix(authUnix, 0)
	if authDate.After(now.Add(5*time.Minute)) || now.Sub(authDate) > maxAge {
		return VerifiedTelegramData{}, fmt.Errorf("expired auth_date")
	}
	var user TelegramUser
	if err := json.Unmarshal([]byte(v.Get("user")), &user); err != nil || user.ID == 0 {
		return VerifiedTelegramData{}, fmt.Errorf("invalid telegram user")
	}
	return VerifiedTelegramData{User: user, AuthDate: authDate}, nil
}
