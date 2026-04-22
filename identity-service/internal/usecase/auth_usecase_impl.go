package usecase

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
	redisRepo "github.com/DoMinhHHung/go-app/identity-service/internal/repository/redis"
	jwtpkg "github.com/DoMinhHHung/go-app/identity-service/pkg/jwt"
)

// Argon2id parameters
const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 2
	argon2KeyLen  = 32
	argon2SaltLen = 16

	maxOTPAttempts = 5
	otpAttemptTTL  = 30 * time.Minute
)

var (
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrPhoneAlreadyExists  = errors.New("phone number already registered")
	ErrPendingVerification = errors.New("account pending verification, please check your email for OTP")
	ErrOTPExpiredOrInvalid = errors.New("otp is expired or invalid")
	ErrOTPTooManyAttempts  = errors.New("too many otp verification attempts, try again in 30 minutes")
	ErrOTPMaxResend        = errors.New("otp resend limit reached")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserNotVerified     = errors.New("email not verified, please check your inbox")
	ErrUserAlreadyVerified = errors.New("email already verified")
	ErrUserBanned          = errors.New("account is banned")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrWeakPassword        = errors.New("password must be at least 8 characters and contain uppercase, lowercase, and a digit")
	ErrUserNotFound        = errors.New("user not found")
)

type EventPublisher interface {
	PublishOTPEmail(ctx context.Context, eventID, recipient, otpCode string, expireSeconds int) error
}

type authUsecaseImpl struct {
	userRepo         domainRepo.UserRepository
	otpRepo          domainRepo.OTPRepository
	tokenRepo        domainRepo.TokenRepository
	publisher        EventPublisher
	logger           *zap.Logger
	otpTTL           time.Duration
	maxResend        int
	jwtAccessSecret  string
	jwtRefreshSecret string
	jwtAccessTTL     time.Duration
	jwtRefreshTTL    time.Duration
}

func NewAuthUsecase(
	userRepo domainRepo.UserRepository,
	otpRepo domainRepo.OTPRepository,
	tokenRepo domainRepo.TokenRepository,
	pub EventPublisher,
	logger *zap.Logger,
	otpTTL time.Duration,
	maxResend int,
	jwtAccessSecret string,
	jwtRefreshSecret string,
	jwtAccessTTL time.Duration,
	jwtRefreshTTL time.Duration,
) AuthUsecase {
	return &authUsecaseImpl{
		userRepo:         userRepo,
		otpRepo:          otpRepo,
		tokenRepo:        tokenRepo,
		publisher:        pub,
		logger:           logger,
		otpTTL:           otpTTL,
		maxResend:        maxResend,
		jwtAccessSecret:  jwtAccessSecret,
		jwtRefreshSecret: jwtRefreshSecret,
		jwtAccessTTL:     jwtAccessTTL,
		jwtRefreshTTL:    jwtRefreshTTL,
	}
}

func (uc *authUsecaseImpl) Register(ctx context.Context, input RegisterInput) error {
	if err := validatePassword(input.Password); err != nil {
		return err
	}

	hashedPw, err := hashArgon2id(input.Password)
	if err != nil {
		return fmt.Errorf("register: hash password: %w", err)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("register: gen uuid: %w", err)
	}

	otpCode, err := generateOTP()
	if err != nil {
		return fmt.Errorf("register: gen otp: %w", err)
	}

	user := &entity.User{
		ID:           userID,
		EmailAddress: input.EmailAddress,
		FullName:     input.FullName,
		Password:     hashedPw,
		PhoneNumber:  input.PhoneNumber,
		Role:         entity.RoleUser,
		Status:       entity.StatusPending,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		switch {
		case errors.Is(err, domainRepo.ErrEmailConflict):
			return uc.handleEmailConflict(ctx, input.EmailAddress, user)
		case errors.Is(err, domainRepo.ErrPhoneConflict):
			return ErrPhoneAlreadyExists
		}
		return fmt.Errorf("register: create user: %w", err)
	}

	if err := uc.otpRepo.Save(ctx, input.EmailAddress, otpCode, uc.otpTTL); err != nil {
		uc.logger.Error("register: save otp failed, rolling back user creation",
			zap.String("user_id", userID.String()), zap.Error(err))
		if rbErr := uc.userRepo.DeleteByID(ctx, userID.String()); rbErr != nil {
			uc.logger.Error("register: rollback failed", zap.Error(rbErr))
		}
		return fmt.Errorf("register: save otp: %w", err)
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		uc.logger.Warn("register: gen event uuid failed, using user id as fallback", zap.Error(err))
		eventID = userID
	}

	if err := uc.publisher.PublishOTPEmail(ctx, eventID.String(), input.EmailAddress, otpCode, int(uc.otpTTL.Seconds())); err != nil {
		uc.logger.Warn("register: publish otp email failed", zap.Error(err))
	}

	uc.logger.Info("user registered, pending verification",
		zap.String("email", input.EmailAddress),
		zap.String("user_id", userID.String()),
	)
	return nil
}

func (uc *authUsecaseImpl) handleEmailConflict(ctx context.Context, email string, newUser *entity.User) error {
	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return ErrEmailAlreadyExists
	}

	if existing.Status != entity.StatusPending {
		return ErrEmailAlreadyExists
	}

	_, otpErr := uc.otpRepo.Get(ctx, email)
	if !errors.Is(otpErr, redisRepo.ErrOTPNotFound) {
		return ErrPendingVerification
	}

	if rbErr := uc.userRepo.DeleteByID(ctx, existing.ID.String()); rbErr != nil {
		uc.logger.Error("register: soft-delete stale pending failed", zap.Error(rbErr))
		return ErrEmailAlreadyExists
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		if errors.Is(err, domainRepo.ErrEmailConflict) {
			return ErrEmailAlreadyExists
		}
		return fmt.Errorf("register: retry create: %w", err)
	}
	return nil
}

func (uc *authUsecaseImpl) VerifyOTP(ctx context.Context, email, code string) error {
	attempts, err := uc.otpRepo.GetAttemptCount(ctx, email)
	if err != nil {
		return fmt.Errorf("verify otp: check attempts: %w", err)
	}
	if attempts >= maxOTPAttempts {
		return ErrOTPTooManyAttempts
	}

	saved, err := uc.otpRepo.Get(ctx, email)
	if errors.Is(err, redisRepo.ErrOTPNotFound) {
		return ErrOTPExpiredOrInvalid
	}
	if err != nil {
		return fmt.Errorf("verify otp: get: %w", err)
	}

	if subtle.ConstantTimeCompare([]byte(saved), []byte(code)) != 1 {
		_, _ = uc.otpRepo.IncrAttemptCount(ctx, email, otpAttemptTTL)
		return ErrOTPExpiredOrInvalid
	}

	_ = uc.otpRepo.Delete(ctx, email)
	_ = uc.otpRepo.DeleteAttemptCount(ctx, email)

	return uc.userRepo.ActivateByEmail(ctx, email)
}

func (uc *authUsecaseImpl) ResendOTP(ctx context.Context, email string) error {
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domainRepo.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("resend otp: find user: %w", err)
	}

	if user.Status == entity.StatusActive {
		return ErrUserAlreadyVerified
	}

	count, err := uc.otpRepo.IncrResendCount(ctx, email, time.Hour)
	if err != nil {
		return fmt.Errorf("resend otp: incr count: %w", err)
	}
	if count > int64(uc.maxResend) {
		return ErrOTPMaxResend
	}

	otpCode, err := generateOTP()
	if err != nil {
		return fmt.Errorf("resend otp: gen: %w", err)
	}

	if err := uc.otpRepo.Save(ctx, email, otpCode, uc.otpTTL); err != nil {
		return fmt.Errorf("resend otp: save: %w", err)
	}

	eventID, err := uuid.NewV7()
	if err != nil {
		uc.logger.Warn("resend otp: gen event uuid failed, using user id as fallback", zap.Error(err))
		eventID = user.ID
	}

	if err := uc.publisher.PublishOTPEmail(ctx, eventID.String(), email, otpCode, int(uc.otpTTL.Seconds())); err != nil {
		uc.logger.Warn("resend otp: publish failed", zap.Error(err))
	}

	return nil
}

func (uc *authUsecaseImpl) Login(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.EmailAddress)
	if err != nil {
		if errors.Is(err, domainRepo.ErrUserNotFound) {
			return LoginOutput{}, ErrInvalidCredentials
		}
		return LoginOutput{}, fmt.Errorf("login: find user: %w", err)
	}

	switch user.Status {
	case entity.StatusPending:
		return LoginOutput{}, ErrUserNotVerified
	case entity.StatusBanned:
		return LoginOutput{}, ErrUserBanned
	}

	ok, err := verifyArgon2id(input.Password, user.Password)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: verify password: %w", err)
	}
	if !ok {
		return LoginOutput{}, ErrInvalidCredentials
	}

	accessToken, err := jwtpkg.GenerateAccessToken(
		user.ID.String(), user.EmailAddress, string(user.Role),
		uc.jwtAccessSecret, uc.jwtAccessTTL,
	)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: generate access token: %w", err)
	}

	refreshToken, err := jwtpkg.GenerateRefreshToken(user.ID.String(), uc.jwtRefreshSecret, uc.jwtRefreshTTL)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: generate refresh token: %w", err)
	}

	if err := uc.tokenRepo.SaveRefreshToken(ctx, user.ID.String(), refreshToken, uc.jwtRefreshTTL); err != nil {
		return LoginOutput{}, fmt.Errorf("login: save refresh token: %w", err)
	}

	uc.logger.Info("user logged in", zap.String("user_id", user.ID.String()))
	return LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtAccessTTL.Seconds()),
	}, nil
}

func (uc *authUsecaseImpl) RefreshToken(ctx context.Context, refreshToken string) (RefreshOutput, error) {
	userID, err := jwtpkg.ParseRefreshToken(refreshToken, uc.jwtRefreshSecret)
	if err != nil {
		return RefreshOutput{}, ErrInvalidToken
	}

	stored, err := uc.tokenRepo.GetRefreshToken(ctx, userID)
	if err != nil || stored != refreshToken {
		return RefreshOutput{}, ErrInvalidToken
	}

	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return RefreshOutput{}, fmt.Errorf("refresh token: find user: %w", err)
	}

	accessToken, err := jwtpkg.GenerateAccessToken(
		userID, user.EmailAddress, string(user.Role),
		uc.jwtAccessSecret, uc.jwtAccessTTL,
	)
	if err != nil {
		return RefreshOutput{}, fmt.Errorf("refresh token: generate: %w", err)
	}

	return RefreshOutput{
		AccessToken: accessToken,
		ExpiresIn:   int64(uc.jwtAccessTTL.Seconds()),
	}, nil
}

func (uc *authUsecaseImpl) Logout(ctx context.Context, refreshToken string) error {
	userID, err := jwtpkg.ParseRefreshToken(refreshToken, uc.jwtRefreshSecret)
	if err != nil {
		return ErrInvalidToken
	}

	stored, err := uc.tokenRepo.GetRefreshToken(ctx, userID)
	if err != nil || stored != refreshToken {
		return ErrInvalidToken
	}

	return uc.tokenRepo.DeleteRefreshToken(ctx, userID)
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashArgon2id(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%x$%x",
		argon2Memory, argon2Time, argon2Threads, salt, hash), nil
}

func verifyArgon2id(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("verify: invalid hash format")
	}

	var memory uint32
	var timeCost uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads)
	if err != nil {
		return false, fmt.Errorf("verify: parse params: %w", err)
	}

	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("verify: decode salt: %w", err)
	}

	expectedHash, err := hex.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("verify: decode hash: %w", err)
	}

	keyLen := uint32(len(expectedHash))
	computed := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLen)
	return subtle.ConstantTimeCompare(computed, expectedHash) == 1, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}
