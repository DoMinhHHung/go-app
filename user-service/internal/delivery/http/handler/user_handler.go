package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/DoMinhHHung/go-app/user-service/internal/delivery/http/dto"
	"github.com/DoMinhHHung/go-app/user-service/internal/usecase"
)

type UserHandler struct {
	userUsecase usecase.AuthUsecase
	logger      *zap.Logger
}

func NewUserHandler(userUsecase usecase.AuthUsecase, logger *zap.Logger) *UserHandler {
	return &UserHandler{userUsecase: userUsecase, logger: logger}
}

// GetMe godoc
// @Summary      Lấy thông tin profile bản thân
// @Description  Trả về thông tin của user đang đăng nhập dựa trên JWT được API Gateway xác thực.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.Response{data=usecase.ProfileOutput}
// @Failure      401 {object} dto.ErrorResponse "Chưa đăng nhập"
// @Failure      404 {object} dto.ErrorResponse "User không tồn tại"
// @Failure      500 {object} dto.ErrorResponse "Lỗi server"
// @Router       /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.Fail("unauthorized", "UNAUTHORIZED"))
		return
	}

	profile, err := h.userUsecase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			c.JSON(http.StatusNotFound, dto.Fail("user not found", "USER_NOT_FOUND"))
		default:
			h.logger.Error("get profile failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("ok", profile))
}

// UpdateMe godoc
// @Summary      Cập nhật profile bản thân
// @Description  Cho phép user cập nhật full_name và phone_number của chính mình.
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body dto.UpdateProfileRequest true "Thông tin cập nhật"
// @Success      200 {object} dto.Response{data=usecase.ProfileOutput}
// @Failure      400 {object} dto.ErrorResponse "Dữ liệu không hợp lệ"
// @Failure      401 {object} dto.ErrorResponse "Chưa đăng nhập"
// @Failure      404 {object} dto.ErrorResponse "User không tồn tại"
// @Failure      500 {object} dto.ErrorResponse "Lỗi server"
// @Router       /api/v1/users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.Fail("unauthorized", "UNAUTHORIZED"))
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Fail(err.Error(), "VALIDATION_ERROR"))
		return
	}

	profile, err := h.userUsecase.UpdateProfile(c.Request.Context(), userID, usecase.UpdateProfileInput{
		FullName:    req.FullName,
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			c.JSON(http.StatusNotFound, dto.Fail("user not found", "USER_NOT_FOUND"))
		default:
			h.logger.Error("update profile failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("profile updated", profile))
}

// DeleteMe godoc
// @Summary      Xóa tài khoản bản thân (soft delete)
// @Description  Soft delete tài khoản của user đang đăng nhập. Sau khi xóa, email/phone có thể đăng ký lại.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.Response "Xóa thành công"
// @Failure      401 {object} dto.ErrorResponse "Chưa đăng nhập"
// @Failure      404 {object} dto.ErrorResponse "User không tồn tại"
// @Failure      500 {object} dto.ErrorResponse "Lỗi server"
// @Router       /api/v1/users/me [delete]
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.Fail("unauthorized", "UNAUTHORIZED"))
		return
	}

	if err := h.userUsecase.SoftDelete(c.Request.Context(), userID); err != nil {
		switch {
		case errors.Is(err, usecase.ErrUserNotFound):
			c.JSON(http.StatusNotFound, dto.Fail("user not found", "USER_NOT_FOUND"))
		default:
			h.logger.Error("soft delete failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		}
		return
	}

	c.JSON(http.StatusOK, dto.OK("account deleted", nil))
}

// ListUsers godoc
// @Summary      Danh sách users (Admin only)
// @Description  Trả về danh sách tất cả user đang active. Chỉ dành cho admin.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query int false "Số lượng kết quả (mặc định 20, tối đa 100)" default(20)
// @Param        offset query int false "Vị trí bắt đầu" default(0)
// @Success      200 {object} dto.Response{data=dto.ListUsersData}
// @Failure      401 {object} dto.ErrorResponse "Chưa đăng nhập"
// @Failure      403 {object} dto.ErrorResponse "Không phải admin"
// @Failure      500 {object} dto.ErrorResponse "Lỗi server"
// @Router       /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	role := c.GetHeader("X-User-Role")
	if role != "admin" {
		c.JSON(http.StatusForbidden, dto.Fail("admin only", "FORBIDDEN"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, err := h.userUsecase.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("list users failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.Fail("internal error", "INTERNAL_ERROR"))
		return
	}

	c.JSON(http.StatusOK, dto.OK("ok", dto.ListUsersData{
		Users:  users,
		Limit:  limit,
		Offset: offset,
	}))
}
