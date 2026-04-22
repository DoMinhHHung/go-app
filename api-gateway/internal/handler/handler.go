package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	_ "github.com/DoMinhHHung/go-app/api-gateway/docs"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/dto"
	"github.com/DoMinhHHung/go-app/api-gateway/internal/proxy"
)

// GatewayHandler holds proxies to all upstream services and exposes annotated handlers.
type GatewayHandler struct {
	identityProxy *proxy.Proxy
	userProxy     *proxy.Proxy
}

func New(identityProxy, userProxy *proxy.Proxy) *GatewayHandler {
	return &GatewayHandler{
		identityProxy: identityProxy,
		userProxy:     userProxy,
	}
}

// Health godoc
// @Summary      Health check
// @Description  Returns gateway status
// @Tags         system
// @Produce      json
// @Success      200 {object} dto.HealthResponse
// @Router       /health [get]
func (h *GatewayHandler) Health(cfg dto.HealthResponse) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, cfg)
	}
}

// — Auth routes (proxied to identity-service) —————————————————————————————————

// Register godoc
// @Summary      Đăng ký tài khoản
// @Description  Tạo tài khoản mới và gửi mã OTP qua email. Tài khoản ở trạng thái pending cho đến khi xác thực OTP.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "Thông tin đăng ký"
// @Success      201 {object} dto.SuccessResponse
// @Failure      400 {object} dto.ErrorResponse "Validation error / weak password"
// @Failure      409 {object} dto.ErrorResponse "Email already registered"
// @Failure      429 {object} dto.ErrorResponse "Rate limit exceeded"
// @Failure      502 {object} dto.ErrorResponse "Upstream unavailable"
// @Router       /api/v1/auth/register [post]
func (h *GatewayHandler) Register(c *gin.Context) { h.identityProxy.Handler()(c) }

// VerifyOTP godoc
// @Summary      Xác thực OTP
// @Description  Xác thực mã OTP gửi qua email. Sau khi thành công tài khoản chuyển sang active.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.VerifyOTPRequest true "OTP verification"
// @Success      200 {object} dto.SuccessResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse "OTP invalid or expired"
// @Failure      429 {object} dto.ErrorResponse "Too many attempts (max 5 per 30 min)"
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/auth/verify-otp [post]
func (h *GatewayHandler) VerifyOTP(c *gin.Context) { h.identityProxy.Handler()(c) }

// ResendOTP godoc
// @Summary      Gửi lại OTP
// @Description  Gửi lại mã OTP mới qua email. Giới hạn 3 lần/giờ.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.ResendOTPRequest true "Email cần gửi lại OTP"
// @Success      200 {object} dto.SuccessResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse "User not found"
// @Failure      409 {object} dto.ErrorResponse "Email already verified"
// @Failure      429 {object} dto.ErrorResponse "Resend limit reached"
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/auth/resend-otp [post]
func (h *GatewayHandler) ResendOTP(c *gin.Context) { h.identityProxy.Handler()(c) }

// Login godoc
// @Summary      Đăng nhập
// @Description  Xác thực email và password. Trả về access token (15 phút) và refresh token (7 ngày).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "Thông tin đăng nhập"
// @Success      200 {object} dto.SuccessResponse{data=dto.LoginData}
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse "Invalid credentials"
// @Failure      403 {object} dto.ErrorResponse "Email not verified / account banned"
// @Failure      429 {object} dto.ErrorResponse
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *GatewayHandler) Login(c *gin.Context) { h.identityProxy.Handler()(c) }

// RefreshToken godoc
// @Summary      Làm mới access token
// @Description  Dùng refresh token để lấy access token mới mà không cần đăng nhập lại.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.RefreshTokenRequest true "Refresh token"
// @Success      200 {object} dto.SuccessResponse{data=dto.RefreshData}
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse "Invalid or expired refresh token"
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/auth/refresh [post]
func (h *GatewayHandler) RefreshToken(c *gin.Context) { h.identityProxy.Handler()(c) }

// Logout godoc
// @Summary      Đăng xuất
// @Description  Vô hiệu hóa refresh token. Access token sẽ hết hạn tự nhiên.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body dto.LogoutRequest true "Refresh token cần revoke"
// @Success      200 {object} dto.SuccessResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse "Invalid token"
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/auth/logout [post]
func (h *GatewayHandler) Logout(c *gin.Context) { h.identityProxy.Handler()(c) }

// — User routes (proxied to user-service, JWT required) ───────────────────────

// GetMe godoc
// @Summary      Lấy thông tin bản thân
// @Description  Trả về thông tin profile của user đang đăng nhập.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.SuccessResponse
// @Failure      401 {object} dto.ErrorResponse "Token invalid or missing"
// @Failure      502 {object} dto.ErrorResponse
// @Router       /api/v1/users/me [get]
func (h *GatewayHandler) GetMe(c *gin.Context) { h.userProxy.Handler()(c) }
