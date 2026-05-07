package middleware

import (
	"net/http"
	"strings"

	"timesheesh-backend/database"
	"timesheesh-backend/models"
	"timesheesh-backend/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT token and sets user in context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from database
		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", &user)
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)

		c.Next()
	}
}

// RoleMiddleware checks if user has required role
func RoleMiddleware(allowedRoles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		role := userRole.(models.Role)

		// Check if user role is in allowed roles
		allowed := false
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// AdminOnly middleware - only allows admin role
func AdminOnly() gin.HandlerFunc {
	return RoleMiddleware(models.RoleAdmin)
}

// ProjectManagerOnly middleware - only allows project manager role
func ProjectManagerOnly() gin.HandlerFunc {
	return RoleMiddleware(models.RoleProjectManager)
}

// EmployeeOnly middleware - only allows employee role
func EmployeeOnly() gin.HandlerFunc {
	return RoleMiddleware(models.RoleEmployee)
}

// FinanceOnly middleware - only allows finance/management role
func FinanceOnly() gin.HandlerFunc {
	return RoleMiddleware(models.RoleFinance)
}

// AdminOrProjectManager middleware - allows admin or project manager
func AdminOrProjectManager() gin.HandlerFunc {
	return RoleMiddleware(models.RoleAdmin, models.RoleProjectManager)
}

