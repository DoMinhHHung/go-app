package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"

	"github.com/DoMinhHHung/go-app/identity-service/internal/domain/entity"
	domainRepo "github.com/DoMinhHHung/go-app/identity-service/internal/domain/repository"
	"github.com/DoMinhHHung/go-app/identity-service/internal/event/publisher"
	redisRepo "github.com/DoMinhHHung/go-app/identity-service/internal/repository/redis"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrOTPExpiredOrInvalid = errors.New("otp is expired or invalid")
	ErrOTPMaxResend        = errors.New("otp resend limit reached")
)

type authUsecaseImpl struct {
	userRepo  domainRepo.UserRepository
	otpRepo   domainRepo.OTPRepository
	publisher *publisher.NotificationPublisher
	logger    *zap.Logger
	otpTTL    time.Duration
	maxResend int
}

func NewAuthUsecase(
	userRepo domainRepo.UserRepository,
	otpRepo domainRepo.OTPRepository,
	pub *publisher.NotificationPublisher,
	logger *zap.Logger,
	otpTTL time.Duration,
	maxResend int,
) AuthUsecase {
	return &authUsecaseImpl{
		userRepo:  userRepo,
		otpRepo:   otpRepo,
		publisher: pub,
		logger:    logger,
		otpTTL:    otpTTL,
		maxResend: maxResend,
	}
}

func (uc *authUsecaseImpl) Register(ctx context.Context, input RegisterInput) error {
	exists, err := uc.userRepo.ExistsActiveEmail(ctx, input.EmailAddress)
	if err != nil {
		return fmt.Errorf("register: check email: %w", err)
	}
	if exists {
		return ErrEmailAlreadyExists
	}

	hashedPw, err := hashArgon2id(input.Password)
	if err != nil {
		return fmt.Errorf("register: hash password: %w", err)
	}

	userID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("register: gen uuid: %w", err)
	}

	user := &entity.User{
		ID:           userID,
		EmailAddress: input.EmailAddress,
		FullName:     input.FullName,
		Password:     hashedPw,
		PhoneNumber:  input.PhoneNumber,
		Role:         entity.RoleUser,
		Status:       entity.StatusActive,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("register: create user: %w", err)
	}

	otpCode, err := generateOTP()
	if err != nil {
		return fmt.Errorf("register: gen otp: %w", err)
	}

	if err := uc.otpRepo.Save(ctx, input.EmailAddress, otpCode, uc.otpTTL); err != nil {
		return fmt.Errorf("register: save otp: %w", err)
	}

	eventID, _ := uuid.NewV7()
	if err := uc.publisher.PublishOTPEmail(ctx,
		eventID.String(),
		input.EmailAddress,
		otpCode,
		int(uc.otpTTL.Seconds()),
	); err != nil {
		uc.logger.Warn("register: publish otp email failed", zap.Error(err))
	}

	uc.logger.Info("user registered",
		zap.String("email", input.EmailAddress),
		zap.String("user_id", userID.String()),
	)
	return nil
}

func (uc *authUsecaseImpl) VerifyOTP(ctx context.Context, email, code string) error {
	saved, err := uc.otpRepo.Get(ctx, email)
	if errors.Is(err, redisRepo.ErrOTPNotFound) {
		return ErrOTPExpiredOrInvalid
	}
	if err != nil {
		return fmt.Errorf("verify otp: get: %w", err)
	}

	if saved != code {
		return ErrOTPExpiredOrInvalid
	}

	if err := uc.otpRepo.Delete(ctx, email); err != nil {
		uc.logger.Warn("verify otp: delete failed", zap.Error(err))
	}

	return nil
}

func (uc *authUsecaseImpl) ResendOTP(ctx context.Context, email string) error {
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

	eventID, _ := uuid.NewV7()
	if err := uc.publisher.PublishOTPEmail(ctx,
		eventID.String(),
		email,
		otpCode,
		int(uc.otpTTL.Seconds()),
	); err != nil {
		uc.logger.Warn("resend otp: publish failed", zap.Error(err))
	}

	return nil
}

func generateOTP() (string, error) {
	const digits = "0123456789"
	result := make([]byte, 6)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}

func hashArgon2id(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%x$%x", salt, hash), nil
}
