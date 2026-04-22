package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
	redisRepo "github.com/DoMinhHHung/go-app/identity-service/internal/repository/redis"
	"github.com/DoMinhHHung/go-app/identity-service/internal/usecase"
)

type mockUserRepo struct {
	createFn          func(ctx context.Context, u *entity.User) error
	findByEmailFn     func(ctx context.Context, email string) (*entity.User, error)
	findByIDFn        func(ctx context.Context, id string) (*entity.User, error)
	existsActiveEmail func(ctx context.Context, email string) (bool, error)
	deleteByIDFn      func(ctx context.Context, id string) error
	activateByEmailFn func(ctx context.Context, email string) error
}

func (m *mockUserRepo) Create(ctx context.Context, u *entity.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, domainRepo.ErrUserNotFound
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*entity.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, domainRepo.ErrUserNotFound
}
func (m *mockUserRepo) ExistsActiveEmail(ctx context.Context, email string) (bool, error) {
	if m.existsActiveEmail != nil {
		return m.existsActiveEmail(ctx, email)
	}
	return false, nil
}
func (m *mockUserRepo) DeleteByID(ctx context.Context, id string) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}
func (m *mockUserRepo) ActivateByEmail(ctx context.Context, email string) error {
	if m.activateByEmailFn != nil {
		return m.activateByEmailFn(ctx, email)
	}
	return nil
}

type mockOTPRepo struct {
	saveFn          func(ctx context.Context, email, code string, ttl time.Duration) error
	getFn           func(ctx context.Context, email string) (string, error)
	deleteFn        func(ctx context.Context, email string) error
	incrResendFn    func(ctx context.Context, email string, ttl time.Duration) (int64, error)
	getResendFn     func(ctx context.Context, email string) (int64, error)
	incrAttemptFn   func(ctx context.Context, email string, ttl time.Duration) (int64, error)
	deleteAttemptFn func(ctx context.Context, email string) error
}

func (m *mockOTPRepo) Save(ctx context.Context, email, code string, ttl time.Duration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, email, code, ttl)
	}
	return nil
}
func (m *mockOTPRepo) Get(ctx context.Context, email string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, email)
	}
	return "", redisRepo.ErrOTPNotFound
}
func (m *mockOTPRepo) Delete(ctx context.Context, email string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, email)
	}
	return nil
}
func (m *mockOTPRepo) IncrResendCount(ctx context.Context, email string, ttl time.Duration) (int64, error) {
	if m.incrResendFn != nil {
		return m.incrResendFn(ctx, email, ttl)
	}
	return 1, nil
}
func (m *mockOTPRepo) GetResendCount(ctx context.Context, email string) (int64, error) {
	if m.getResendFn != nil {
		return m.getResendFn(ctx, email)
	}
	return 0, nil
}
func (m *mockOTPRepo) IncrAttemptCount(ctx context.Context, email string, ttl time.Duration) (int64, error) {
	if m.incrAttemptFn != nil {
		return m.incrAttemptFn(ctx, email, ttl)
	}
	return 1, nil
}
func (m *mockOTPRepo) DeleteAttemptCount(ctx context.Context, email string) error {
	if m.deleteAttemptFn != nil {
		return m.deleteAttemptFn(ctx, email)
	}
	return nil
}

type mockTokenRepo struct {
	saveFn   func(ctx context.Context, userID, token string, ttl time.Duration) error
	getFn    func(ctx context.Context, userID string) (string, error)
	deleteFn func(ctx context.Context, userID string) error
}

func (m *mockTokenRepo) SaveRefreshToken(ctx context.Context, userID, token string, ttl time.Duration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, userID, token, ttl)
	}
	return nil
}
func (m *mockTokenRepo) GetRefreshToken(ctx context.Context, userID string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID)
	}
	return "", redisRepo.ErrTokenNotFound
}
func (m *mockTokenRepo) DeleteRefreshToken(ctx context.Context, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID)
	}
	return nil
}

type mockPublisher struct {
	publishFn func(ctx context.Context, eventID, recipient, otpCode string, expireSeconds int) error
}

func (m *mockPublisher) PublishOTPEmail(ctx context.Context, eventID, recipient, otpCode string, expireSeconds int) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, eventID, recipient, otpCode, expireSeconds)
	}
	return nil
}

func newTestUsecase(userRepo *mockUserRepo, otpRepo *mockOTPRepo, tokenRepo *mockTokenRepo, pub *mockPublisher) usecase.AuthUsecase {
	return usecase.NewAuthUsecase(
		userRepo, otpRepo, tokenRepo, pub,
		zap.NewNop(),
		5*time.Minute, 3,
		"access-secret", "refresh-secret",
		15*time.Minute, 7*24*time.Hour,
	)
}

// --- Register tests ---

func TestRegister_Success(t *testing.T) {
	uc := newTestUsecase(&mockUserRepo{}, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	err := uc.Register(context.Background(), usecase.RegisterInput{
		EmailAddress: "user@example.com",
		FullName:     "Test User",
		Password:     "Secret1234",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	uc := newTestUsecase(&mockUserRepo{}, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	err := uc.Register(context.Background(), usecase.RegisterInput{
		EmailAddress: "user@example.com",
		FullName:     "Test User",
		Password:     "weakpass",
	})
	if !errors.Is(err, usecase.ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestRegister_EmailConflict(t *testing.T) {
	userRepo := &mockUserRepo{
		createFn: func(_ context.Context, _ *entity.User) error {
			return domainRepo.ErrEmailConflict
		},
	}
	uc := newTestUsecase(userRepo, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	err := uc.Register(context.Background(), usecase.RegisterInput{
		EmailAddress: "dup@example.com",
		FullName:     "Test",
		Password:     "Valid1Pass",
	})
	if !errors.Is(err, usecase.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegister_OTPSaveFails_RollsBackUser(t *testing.T) {
	deleted := false
	userRepo := &mockUserRepo{
		deleteByIDFn: func(_ context.Context, _ string) error {
			deleted = true
			return nil
		},
	}
	otpRepo := &mockOTPRepo{
		saveFn: func(_ context.Context, _, _ string, _ time.Duration) error {
			return errors.New("redis down")
		},
	}
	uc := newTestUsecase(userRepo, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.Register(context.Background(), usecase.RegisterInput{
		EmailAddress: "user@example.com",
		FullName:     "Test",
		Password:     "Valid1Pass",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !deleted {
		t.Fatal("expected user rollback (DeleteByID), but it was not called")
	}
}

// --- VerifyOTP tests ---

func TestVerifyOTP_Success(t *testing.T) {
	activated := false
	userRepo := &mockUserRepo{
		activateByEmailFn: func(_ context.Context, _ string) error {
			activated = true
			return nil
		},
	}
	otpRepo := &mockOTPRepo{
		incrAttemptFn: func(_ context.Context, _ string, _ time.Duration) (int64, error) {
			return 1, nil
		},
		getFn: func(_ context.Context, _ string) (string, error) {
			return "123456", nil
		},
	}
	uc := newTestUsecase(userRepo, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.VerifyOTP(context.Background(), "user@example.com", "123456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !activated {
		t.Fatal("expected ActivateByEmail to be called")
	}
}

func TestVerifyOTP_InvalidCode(t *testing.T) {
	otpRepo := &mockOTPRepo{
		incrAttemptFn: func(_ context.Context, _ string, _ time.Duration) (int64, error) {
			return 1, nil
		},
		getFn: func(_ context.Context, _ string) (string, error) {
			return "654321", nil
		},
	}
	uc := newTestUsecase(&mockUserRepo{}, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.VerifyOTP(context.Background(), "user@example.com", "000000")
	if !errors.Is(err, usecase.ErrOTPExpiredOrInvalid) {
		t.Fatalf("expected ErrOTPExpiredOrInvalid, got %v", err)
	}
}

func TestVerifyOTP_TooManyAttempts(t *testing.T) {
	otpRepo := &mockOTPRepo{
		incrAttemptFn: func(_ context.Context, _ string, _ time.Duration) (int64, error) {
			return 6, nil
		},
	}
	uc := newTestUsecase(&mockUserRepo{}, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.VerifyOTP(context.Background(), "user@example.com", "123456")
	if !errors.Is(err, usecase.ErrOTPTooManyAttempts) {
		t.Fatalf("expected ErrOTPTooManyAttempts, got %v", err)
	}
}

func TestVerifyOTP_Expired(t *testing.T) {
	otpRepo := &mockOTPRepo{
		incrAttemptFn: func(_ context.Context, _ string, _ time.Duration) (int64, error) {
			return 1, nil
		},
		getFn: func(_ context.Context, _ string) (string, error) {
			return "", redisRepo.ErrOTPNotFound
		},
	}
	uc := newTestUsecase(&mockUserRepo{}, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.VerifyOTP(context.Background(), "user@example.com", "123456")
	if !errors.Is(err, usecase.ErrOTPExpiredOrInvalid) {
		t.Fatalf("expected ErrOTPExpiredOrInvalid, got %v", err)
	}
}

// --- Login tests ---

func activeUser() *entity.User {
	id, _ := uuid.NewV7()
	return &entity.User{
		ID:           id,
		EmailAddress: "user@example.com",
		Role:         entity.RoleUser,
		Status:       entity.StatusActive,
		// Use a known valid hash — "Valid1Pass" hashed at runtime won't match here,
		// so we test the wrong-password path; for correct-password path see integration tests.
	}
}

func TestLogin_UserNotFound_ReturnsInvalidCredentials(t *testing.T) {
	uc := newTestUsecase(&mockUserRepo{}, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		EmailAddress: "nobody@example.com",
		Password:     "Valid1Pass",
	})
	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_PendingUser_ReturnsNotVerified(t *testing.T) {
	id, _ := uuid.NewV7()
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{ID: id, Status: entity.StatusPending}, nil
		},
	}
	uc := newTestUsecase(userRepo, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		EmailAddress: "pending@example.com",
		Password:     "Valid1Pass",
	})
	if !errors.Is(err, usecase.ErrUserNotVerified) {
		t.Fatalf("expected ErrUserNotVerified, got %v", err)
	}
}

func TestLogin_BannedUser(t *testing.T) {
	id, _ := uuid.NewV7()
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{ID: id, Status: entity.StatusBanned}, nil
		},
	}
	uc := newTestUsecase(userRepo, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		EmailAddress: "banned@example.com",
		Password:     "Valid1Pass",
	})
	if !errors.Is(err, usecase.ErrUserBanned) {
		t.Fatalf("expected ErrUserBanned, got %v", err)
	}
}

// --- ResendOTP tests ---

func TestResendOTP_UserNotFound(t *testing.T) {
	uc := newTestUsecase(&mockUserRepo{}, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	err := uc.ResendOTP(context.Background(), "ghost@example.com")
	if !errors.Is(err, usecase.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestResendOTP_AlreadyVerified(t *testing.T) {
	id, _ := uuid.NewV7()
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{ID: id, Status: entity.StatusActive}, nil
		},
	}
	uc := newTestUsecase(userRepo, &mockOTPRepo{}, &mockTokenRepo{}, &mockPublisher{})
	err := uc.ResendOTP(context.Background(), "verified@example.com")
	if !errors.Is(err, usecase.ErrUserAlreadyVerified) {
		t.Fatalf("expected ErrUserAlreadyVerified, got %v", err)
	}
}

func TestResendOTP_MaxResendReached(t *testing.T) {
	id, _ := uuid.NewV7()
	userRepo := &mockUserRepo{
		findByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
			return &entity.User{ID: id, Status: entity.StatusPending}, nil
		},
	}
	otpRepo := &mockOTPRepo{
		incrResendFn: func(_ context.Context, _ string, _ time.Duration) (int64, error) {
			return 4, nil // exceeds maxResend=3
		},
	}
	uc := newTestUsecase(userRepo, otpRepo, &mockTokenRepo{}, &mockPublisher{})
	err := uc.ResendOTP(context.Background(), "user@example.com")
	if !errors.Is(err, usecase.ErrOTPMaxResend) {
		t.Fatalf("expected ErrOTPMaxResend, got %v", err)
	}
}
