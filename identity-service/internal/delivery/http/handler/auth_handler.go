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

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(err.Error(), "VALIDATION_ERROR"))
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
		default:
			h.logger.Error("register failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusCreated, dto.OK("registration successful, please verify your OTP", nil))
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(err.Error(), "VALIDATION_ERROR"))
		return
	}

	if err := h.authUsecase.VerifyOTP(c.Request.Context(), req.EmailAddress, req.OTPCode); err != nil {
		switch {
		case errors.Is(err, usecase.ErrOTPExpiredOrInvalid):
			c.JSON(http.StatusUnauthorized, dto.Fail("OTP is invalid or expired", "OTP_INVALID"))
		default:
			h.logger.Error("verify otp failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("email verified successfully", nil))
}

func (h *AuthHandler) ResendOTP(c *gin.Context) {
	var req dto.ResendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(err.Error(), "VALIDATION_ERROR"))
		return
	}

	if err := h.authUsecase.ResendOTP(c.Request.Context(), req.EmailAddress); err != nil {
		switch {
		case errors.Is(err, usecase.ErrOTPMaxResend):
			c.JSON(http.StatusTooManyRequests, dto.Fail("OTP resend limit reached", "OTP_MAX_RESEND"))
		default:
			h.logger.Error("resend otp failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("OTP resent successfully", nil))
}
