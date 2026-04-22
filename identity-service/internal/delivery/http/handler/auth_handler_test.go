package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/handler"
	"github.com/DoMinhHHung/go-app/identity-service/internal/usecase"
)

// mockAuthUsecase implements usecase.AuthUsecase for handler tests.
type mockAuthUsecase struct {
	registerFn     func(ctx context.Context, input usecase.RegisterInput) error
	verifyOTPFn    func(ctx context.Context, email, code string) error
	resendOTPFn    func(ctx context.Context, email string) error
	loginFn        func(ctx context.Context, input usecase.LoginInput) (usecase.LoginOutput, error)
	refreshTokenFn func(ctx context.Context, token string) (usecase.RefreshOutput, error)
	logoutFn       func(ctx context.Context, token string) error
}

func (m *mockAuthUsecase) Register(ctx context.Context, input usecase.RegisterInput) error {
	if m.registerFn != nil {
		return m.registerFn(ctx, input)
	}
	return nil
}
func (m *mockAuthUsecase) VerifyOTP(ctx context.Context, email, code string) error {
	if m.verifyOTPFn != nil {
		return m.verifyOTPFn(ctx, email, code)
	}
	return nil
}
func (m *mockAuthUsecase) ResendOTP(ctx context.Context, email string) error {
	if m.resendOTPFn != nil {
		return m.resendOTPFn(ctx, email)
	}
	return nil
}
func (m *mockAuthUsecase) Login(ctx context.Context, input usecase.LoginInput) (usecase.LoginOutput, error) {
	if m.loginFn != nil {
		return m.loginFn(ctx, input)
	}
	return usecase.LoginOutput{}, nil
}
func (m *mockAuthUsecase) RefreshToken(ctx context.Context, token string) (usecase.RefreshOutput, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(ctx, token)
	}
	return usecase.RefreshOutput{}, nil
}
func (m *mockAuthUsecase) Logout(ctx context.Context, token string) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, token)
	}
	return nil
}

func setupRouter(uc usecase.AuthUsecase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewAuthHandler(uc, zap.NewNop())
	r.POST("/register", h.Register)
	r.POST("/verify-otp", h.VerifyOTP)
	r.POST("/resend-otp", h.ResendOTP)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.RefreshToken)
	r.POST("/logout", h.Logout)
	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Register ---

func TestRegister_Returns201(t *testing.T) {
	r := setupRouter(&mockAuthUsecase{})
	w := postJSON(r, "/register", map[string]string{
		"email_address": "user@example.com",
		"full_name":     "Test User",
		"password":      "Secret1234",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegister_Returns409OnEmailConflict(t *testing.T) {
	uc := &mockAuthUsecase{
		registerFn: func(_ context.Context, _ usecase.RegisterInput) error {
			return usecase.ErrEmailAlreadyExists
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/register", map[string]string{
		"email_address": "dup@example.com",
		"full_name":     "Test User",
		"password":      "Secret1234",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestRegister_Returns400OnWeakPassword(t *testing.T) {
	uc := &mockAuthUsecase{
		registerFn: func(_ context.Context, _ usecase.RegisterInput) error {
			return usecase.ErrWeakPassword
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/register", map[string]string{
		"email_address": "user@example.com",
		"full_name":     "Test",
		"password":      "Secret1234",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRegister_Returns400OnMissingFields(t *testing.T) {
	r := setupRouter(&mockAuthUsecase{})
	w := postJSON(r, "/register", map[string]string{"email_address": "bad"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- VerifyOTP ---

func TestVerifyOTP_Returns200(t *testing.T) {
	r := setupRouter(&mockAuthUsecase{})
	w := postJSON(r, "/verify-otp", map[string]string{
		"email_address": "user@example.com",
		"otp_code":      "123456",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestVerifyOTP_Returns401OnInvalidOTP(t *testing.T) {
	uc := &mockAuthUsecase{
		verifyOTPFn: func(_ context.Context, _, _ string) error {
			return usecase.ErrOTPExpiredOrInvalid
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/verify-otp", map[string]string{
		"email_address": "user@example.com",
		"otp_code":      "000000",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyOTP_Returns429OnTooManyAttempts(t *testing.T) {
	uc := &mockAuthUsecase{
		verifyOTPFn: func(_ context.Context, _, _ string) error {
			return usecase.ErrOTPTooManyAttempts
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/verify-otp", map[string]string{
		"email_address": "user@example.com",
		"otp_code":      "123456",
	})
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
}

// --- Login ---

func TestLogin_Returns200WithTokens(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, _ usecase.LoginInput) (usecase.LoginOutput, error) {
			return usecase.LoginOutput{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
				ExpiresIn:    900,
			}, nil
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/login", map[string]string{
		"email_address": "user@example.com",
		"password":      "Secret1234",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	if !ok || data["access_token"] != "access-token" {
		t.Fatalf("expected access_token in response, got %v", resp)
	}
}

func TestLogin_Returns401OnInvalidCredentials(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, _ usecase.LoginInput) (usecase.LoginOutput, error) {
			return usecase.LoginOutput{}, usecase.ErrInvalidCredentials
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/login", map[string]string{
		"email_address": "user@example.com",
		"password":      "WrongPass1",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_Returns403OnUnverifiedEmail(t *testing.T) {
	uc := &mockAuthUsecase{
		loginFn: func(_ context.Context, _ usecase.LoginInput) (usecase.LoginOutput, error) {
			return usecase.LoginOutput{}, usecase.ErrUserNotVerified
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/login", map[string]string{
		"email_address": "unverified@example.com",
		"password":      "Secret1234",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// --- Logout ---

func TestLogout_Returns200(t *testing.T) {
	r := setupRouter(&mockAuthUsecase{})
	w := postJSON(r, "/logout", map[string]string{"refresh_token": "some-token"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestLogout_Returns401OnInvalidToken(t *testing.T) {
	uc := &mockAuthUsecase{
		logoutFn: func(_ context.Context, _ string) error {
			return usecase.ErrInvalidToken
		},
	}
	r := setupRouter(uc)
	w := postJSON(r, "/logout", map[string]string{"refresh_token": "bad-token"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
