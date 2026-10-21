package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter()
	now := time.Now()
	l.now = func() time.Time { return now }
	for range 120 {
		if !l.allow("127.0.0.1") {
			t.Fatal("request rejected too early")
		}
	}
	if l.allow("127.0.0.1") {
		t.Fatal("rate limit was not enforced")
	}
	now = now.Add(time.Minute + time.Second)
	if !l.allow("127.0.0.1") {
		t.Fatal("rate limit did not reset")
	}
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cors("https://example.com"))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("origin = %q", got)
	}
}
