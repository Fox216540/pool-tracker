package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andrey/pool-tracker/internal/auth"
	"github.com/andrey/pool-tracker/internal/bot"
	"github.com/andrey/pool-tracker/internal/config"
	"github.com/andrey/pool-tracker/internal/domain"
	"github.com/andrey/pool-tracker/internal/service"
	"github.com/gin-gonic/gin"
)

const cookieName = "pool_tracker_session"

type API struct {
	service *service.Service
	config  config.Config
	bot     *bot.Bot
}

type requestCounter struct {
	count   int
	resetAt time.Time
}
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]requestCounter
	now     func() time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{clients: make(map[string]requestCounter), now: time.Now}
}
func (l *rateLimiter) allow(client string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	item := l.clients[client]
	if now.After(item.resetAt) {
		item = requestCounter{resetAt: now.Add(time.Minute)}
	}
	item.count++
	l.clients[client] = item
	return item.count <= 120
}

func New(s *service.Service, c config.Config, b *bot.Bot) *API {
	return &API{service: s, config: c, bot: b}
}
func Router(api *API) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), bodyLimit(1<<20), cors(api.config.AllowedOrigin), newRateLimiter().middleware())
	r.GET("/health/live", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/health/ready", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/telegram/webhook", api.webhook)
	v1 := r.Group("/api/v1")
	v1.POST("/auth/telegram", api.telegramAuth)
	v1.POST("/auth/logout", api.logout)
	protected := v1.Group("")
	protected.Use(api.requireUser)
	protected.GET("/auth/me", api.me)
	protected.GET("/sessions", api.list)
	protected.POST("/sessions", api.create)
	protected.GET("/sessions/:id", api.get)
	protected.PATCH("/sessions/:id", api.update)
	protected.DELETE("/sessions/:id", api.delete)
	protected.GET("/statistics", api.statistics)
	return r
}
func (l *rateLimiter) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.allow(c.ClientIP()) {
			fail(c, http.StatusTooManyRequests, "too many requests")
			return
		}
		c.Next()
	}
}
func cors(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin != "" && c.GetHeader("Origin") == origin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *API) webhook(c *gin.Context) {
	if a.bot == nil || a.config.BotWebhookSecret == "" || c.GetHeader("X-Telegram-Bot-Api-Secret-Token") != a.config.BotWebhookSecret {
		fail(c, http.StatusUnauthorized, "invalid webhook")
		return
	}
	var update bot.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		fail(c, http.StatusBadRequest, "invalid update")
		return
	}
	if err := a.bot.Handle(c.Request.Context(), update); err != nil {
		log.Printf("telegram webhook update failed: %v", err)
		fail(c, http.StatusBadGateway, "could not process update")
		return
	}
	c.Status(http.StatusNoContent)
}
func bodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) { c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit); c.Next() }
}
func fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": http.StatusText(status), "message": message}})
}
func (a *API) telegramAuth(c *gin.Context) {
	var req struct {
		InitData string `json:"initData"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.InitData == "" {
		fail(c, 400, "initData is required")
		return
	}
	verified, err := auth.VerifyTelegramInitData(req.InitData, a.config.BotToken, a.config.TelegramAuthMaxAge, time.Now())
	if err != nil {
		fail(c, 401, "invalid Telegram authorization")
		return
	}
	if a.config.AllowedTelegramID != 0 && verified.User.ID != a.config.AllowedTelegramID {
		fail(c, 403, "access denied")
		return
	}
	u, err := a.service.FindOrCreateUser(verified.User.ID)
	if err != nil {
		fail(c, 500, "could not create user")
		return
	}
	raw, err := a.service.CreateAuthSession(u.ID)
	if err != nil {
		fail(c, 500, "could not create session")
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookieName, raw, int(a.config.SessionTTL.Seconds()), "/", "", strings.HasPrefix(a.config.MiniAppURL, "https://"), true)
	c.JSON(200, gin.H{"telegramId": u.TelegramID})
}
func (a *API) logout(c *gin.Context) {
	if raw, err := c.Cookie(cookieName); err == nil {
		_ = a.service.Logout(raw)
	}
	c.SetCookie(cookieName, "", -1, "/", "", strings.HasPrefix(a.config.MiniAppURL, "https://"), true)
	c.Status(204)
}
func (a *API) requireUser(c *gin.Context) {
	raw, err := c.Cookie(cookieName)
	if err != nil {
		fail(c, 401, "authentication required")
		return
	}
	u, err := a.service.UserByToken(raw)
	if err != nil {
		fail(c, 401, "authentication required")
		return
	}
	c.Set("user", u)
	c.Next()
}
func user(c *gin.Context) domain.User { return c.MustGet("user").(domain.User) }
func (a *API) me(c *gin.Context)      { u := user(c); c.JSON(200, gin.H{"telegramId": u.TelegramID}) }

type sessionInput struct {
	GameDate        string `json:"gameDate"`
	DurationMinutes int    `json:"durationMinutes"`
	Note            string `json:"note"`
}

func dates(c *gin.Context) (string, string, bool) {
	from, to := c.Query("from"), c.Query("to")
	if from == "" || to == "" {
		fail(c, 400, "from and to are required")
		return "", "", false
	}
	return from, to, true
}
func serviceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		fail(c, 404, "not found")
	case errors.Is(err, service.ErrDuplicate):
		fail(c, 409, "a session already exists for this date")
	case errors.Is(err, service.ErrInvalid):
		fail(c, 400, "invalid input")
	default:
		fail(c, 500, "internal server error")
	}
}
func (a *API) list(c *gin.Context) {
	from, to, ok := dates(c)
	if !ok {
		return
	}
	rows, err := a.service.List(user(c).ID, from, to)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(200, rows)
}
func (a *API) create(c *gin.Context) {
	var in sessionInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "invalid JSON")
		return
	}
	row, err := a.service.Create(user(c).ID, in.GameDate, in.DurationMinutes, in.Note)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(201, row)
}
func (a *API) get(c *gin.Context) {
	row, err := a.service.Get(user(c).ID, c.Param("id"))
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(200, row)
}
func (a *API) update(c *gin.Context) {
	var in sessionInput
	if c.ShouldBindJSON(&in) != nil {
		fail(c, 400, "invalid JSON")
		return
	}
	row, err := a.service.Update(user(c).ID, c.Param("id"), in.GameDate, in.DurationMinutes, in.Note)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(200, row)
}
func (a *API) delete(c *gin.Context) {
	if err := a.service.Delete(user(c).ID, c.Param("id")); err != nil {
		serviceError(c, err)
		return
	}
	c.Status(204)
}
func (a *API) statistics(c *gin.Context) {
	from, to, ok := dates(c)
	if !ok {
		return
	}
	stat, err := a.service.Statistics(user(c).ID, from, to)
	if err != nil {
		serviceError(c, err)
		return
	}
	c.JSON(200, stat)
}
