package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/identity-service/internal/delivery/http/dto"
	"github.com/DoMinhHHung/go-app/identity-service/internal/usecase"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	logger      *zap.Logger
}

func NewAuthHandler(authUsecase usecase.AuthUsecase, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		logger:      logger,
	}
}

// Register godoc
// @Summary      Đăng ký user mới
// @Description  Tạo tài khoản và gửi OTP qua email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Thông tin đăng ký"
// @Success      201 {object} dto.Response
// @Failure      400 {object} dto.ErrorResponse
// @Failure      409 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	input := usecase.RegisterInput{
		EmailAddress: req.EmailAddress,
		FullName:     req.FullName,
		Password:     req.Password,
		PhoneNumber:  req.PhoneNumber,
	}

	if err := h.authUsecase.Register(c.Request.Context(), input); err != nil {
		switch {
		case errors.Is(err, usecase.ErrEmailAlreadyExists):
			c.JSON(http.StatusConflict, dto.Fail("email already registered", "EMAIL_CONFLICT"))
		case errors.Is(err, usecase.ErrPhoneAlreadyExists):
			c.JSON(http.StatusConflict, dto.Fail("phone number already registered", "PHONE_CONFLICT"))
		case errors.Is(err, usecase.ErrPendingVerification):
			c.JSON(http.StatusConflict, dto.Fail("account pending verification, please check your email for OTP", "PENDING_VERIFICATION"))
		case errors.Is(err, usecase.ErrWeakPassword):
			c.JSON(http.StatusBadRequest, dto.Fail(err.Error(), "WEAK_PASSWORD"))
		default:
			h.logger.Error("register failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.OK("registration successful, please check your email for OTP", nil))
}

// VerifyOTP godoc
// @Summary      Xác thực OTP
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.VerifyOTPRequest true "OTP verification"
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      429 {object} dto.ErrorResponse
// @Router       /api/v1/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	if err := h.authUsecase.VerifyOTP(c.Request.Context(), req.EmailAddress, req.OTPCode); err != nil {
		switch {
		case errors.Is(err, usecase.ErrOTPExpiredOrInvalid):
			c.JSON(http.StatusUnauthorized, dto.Fail("OTP is invalid or expired", "OTP_INVALID"))
		case errors.Is(err, usecase.ErrOTPTooManyAttempts):
			c.JSON(http.StatusTooManyRequests, dto.Fail(err.Error(), "OTP_TOO_MANY_ATTEMPTS"))
		default:
			h.logger.Error("verify otp failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("email verified successfully", nil))
}

// ResendOTP godoc
// @Summary      Gửi lại OTP
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.ResendOTPRequest true "Resend OTP"
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.ErrorResponse
// @Failure      429 {object} dto.ErrorResponse
// @Router       /api/v1/auth/resend-otp [post]
func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	if err := h.authUsecase.ResendOTP(c.Request.Context(), req.EmailAddress); err != nil {
		switch {
		case errors.Is(err, usecase.ErrOTPMaxResend):
			c.JSON(http.StatusTooManyRequests, dto.Fail("OTP resend limit reached", "OTP_MAX_RESEND"))
		case errors.Is(err, usecase.ErrUserNotFound):
			c.JSON(http.StatusNotFound, dto.Fail("user not found", "USER_NOT_FOUND"))
		case errors.Is(err, usecase.ErrUserAlreadyVerified):
			c.JSON(http.StatusConflict, dto.Fail("email already verified", "ALREADY_VERIFIED"))
		default:
			h.logger.Error("resend otp failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("OTP resent successfully", nil))
}

// Login godoc
// @Summary      Đăng nhập
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Login credentials"
// @Success      200 {object} dto.Response{data=dto.LoginResponseData}
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	output, err := h.authUsecase.Login(c.Request.Context(), usecase.LoginInput{
		EmailAddress: req.EmailAddress,
		Password:     req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, dto.Fail("invalid email or password", "INVALID_CREDENTIALS"))
		case errors.Is(err, usecase.ErrUserNotVerified):
			c.JSON(http.StatusForbidden, dto.Fail("please verify your email first", "EMAIL_NOT_VERIFIED"))
		case errors.Is(err, usecase.ErrUserBanned):
			c.JSON(http.StatusForbidden, dto.Fail("account is banned", "ACCOUNT_BANNED"))
		default:
			h.logger.Error("login failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("login successful", dto.LoginResponseData{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		ExpiresIn:    output.ExpiresIn,
	}))
}

// RefreshToken godoc
// @Summary      Làm mới access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshTokenRequest true "Refresh token"
// @Success      200 {object} dto.Response{data=dto.RefreshResponseData}
// @Failure      401 {object} dto.ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	output, err := h.authUsecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, dto.Fail("invalid or expired refresh token", "TOKEN_INVALID"))
		default:
			h.logger.Error("refresh token failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("token refreshed", dto.RefreshResponseData{
		AccessToken: output.AccessToken,
		ExpiresIn:   output.ExpiresIn,
	}))
}

// Logout godoc
// @Summary      Đăng xuất
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LogoutRequest true "Logout"
// @Success      200 {object} dto.Response
// @Failure      401 {object} dto.ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(dto.ParseValidationError(err), "VALIDATION_ERROR"))
		return
	}

	if err := h.authUsecase.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidToken):
			c.JSON(http.StatusUnauthorized, dto.Fail("invalid refresh token", "TOKEN_INVALID"))
		default:
			h.logger.Error("logout failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("logged out successfully", nil))
}
