package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func authError(c *gin.Context, msg, code string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"message": msg,
		"code":    code,
	})
}

// JWTAuth validates the Bearer token and injects user_id/email/role into context.
// Downstream proxy handlers forward these as X-User-* headers to upstream services.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			authError(c, "authorization token required", "UNAUTHORIZED")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		cl, err := parseAccessToken(tokenStr, secret)
		if err != nil {
			authError(c, "invalid or expired token", "TOKEN_INVALID")
			return
		}

		c.Set("user_id", cl.UserID)
		c.Set("email", cl.Email)
		c.Set("role", cl.Role)
		c.Next()
	}
}

func parseAccessToken(tokenStr, secret string) (*claims, error) {
	cl := &claims{}
	token, err := jwt.ParseWithClaims(tokenStr, cl, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if cl.ExpiresAt != nil && cl.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token expired")
	}
	return cl, nil
}
