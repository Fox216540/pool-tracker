package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func signedInitData(t *testing.T, token string, values url.Values) string {
	t.Helper()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))
	return values.Encode()
}

func TestVerifyTelegramInitData(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	raw := signedInitData(t, "secret", url.Values{"auth_date": {"1792584000"}, "query_id": {"q"}, "user": {`{"id":42}`}})
	verified, err := VerifyTelegramInitData(raw, "secret", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if verified.User.ID != 42 {
		t.Fatalf("id = %d", verified.User.ID)
	}
}

func TestVerifyTelegramInitDataRejectsTamperingAndExpiry(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	raw := signedInitData(t, "secret", url.Values{"auth_date": {"1792584000"}, "user": {`{"id":42}`}})
	if _, err := VerifyTelegramInitData(raw+"x", "secret", time.Hour, now); err == nil {
		t.Fatal("tampered data accepted")
	}
	old := signedInitData(t, "secret", url.Values{"auth_date": {"1792497600"}, "user": {`{"id":42}`}})
	if _, err := VerifyTelegramInitData(old, "secret", time.Hour, now); err == nil {
		t.Fatal("expired data accepted")
	}
}
