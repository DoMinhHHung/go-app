package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

// GetMe godoc
// @Summary Get current user profile
// @Security BearerAuth
// @Router /api/v1/users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	email := c.GetHeader("X-User-Email")
	role := c.GetHeader("X-User-Role")

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "unauthorized",
			"code":    "UNAUTHORIZED",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "ok",
		"data": gin.H{
			"user_id": userID,
			"email":   email,
			"role":    role,
		},
	})
}
