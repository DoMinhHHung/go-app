package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	mw "github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/middleware"
)

type mockRateLimitRepo struct {
	count int64
	err   error
}

func (m *mockRateLimitRepo) IncrBy(_ context.Context, _ string, _ time.Duration) (int64, error) {
	m.count++
	return m.count, m.err
}

func newTestEngine(rl *mw.RateLimitMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/resend-otp",
		rl.ByEmail(2, time.Minute),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)
	return r
}

func postBody(r *gin.Engine, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/resend-otp", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestByEmail_AllowsUnderLimit(t *testing.T) {
	repo := &mockRateLimitRepo{}
	rl := mw.NewRateLimitMiddleware(repo, mw.RateLimitConfig{})
	r := newTestEngine(rl)

	w := postBody(r, map[string]string{"email_address": "user@example.com"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestByEmail_BlocksWhenOverLimit(t *testing.T) {
	// counter already at 3, exceeds max=2
	repo := &mockRateLimitRepo{count: 2}
	rl := mw.NewRateLimitMiddleware(repo, mw.RateLimitConfig{})
	r := newTestEngine(rl)

	w := postBody(r, map[string]string{"email_address": "user@example.com"})
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

func TestByEmail_MissingEmail_FallsBackToIPLimit_NotBypassed(t *testing.T) {
	// With an empty email, fallback IP limit should still be applied (not bypassed).
	// Here count stays under limit so request proceeds — but the key point is
	// that no panic or bypass occurs with malformed JSON.
	repo := &mockRateLimitRepo{}
	rl := mw.NewRateLimitMiddleware(repo, mw.RateLimitConfig{})
	r := newTestEngine(rl)

	// Send malformed / empty email body — should NOT bypass rate limiting
	req := httptest.NewRequest(http.MethodPost, "/resend-otp", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Under limit → request proceeds normally
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (under limit), got %d", w.Code)
	}
	// Verify the fallback counter was incremented (not a free pass)
	if repo.count == 0 {
		t.Fatal("expected rate limit counter to be incremented even for missing email")
	}
}

func TestByEmail_BodyReinjected_HandlerCanReadIt(t *testing.T) {
	// After ByEmail reads the body, the handler should still be able to read it.
	repo := &mockRateLimitRepo{}
	rl := mw.NewRateLimitMiddleware(repo, mw.RateLimitConfig{})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/test",
		rl.ByEmail(5, time.Minute),
		func(c *gin.Context) {
			var body struct {
				EmailAddress string `json:"email_address"`
			}
			if err := c.ShouldBindJSON(&body); err != nil || body.EmailAddress == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "body not re-injected"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"email": body.EmailAddress})
		},
	)

	b, _ := json.Marshal(map[string]string{"email_address": "test@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected handler to read body after middleware, got %d: %s", w.Code, w.Body.String())
	}
}
