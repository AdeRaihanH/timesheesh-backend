package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"timesheesh-backend/database"
	"timesheesh-backend/models"
	"timesheesh-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateUserRequest represents update user request
type UpdateUserRequest struct {
	Email        *string              `json:"email,omitempty"`
	Password     *string              `json:"password,omitempty"`
	FullName     *string              `json:"full_name,omitempty"`
	Role         *models.Role         `json:"role,omitempty"`
	EmployeeType *models.EmployeeType `json:"employee_type,omitempty"`
}

// CreateUserRequest represents create user request (admin only)
type CreateUserRequest struct {
	Email        string               `json:"email" binding:"required,email"`
	Password     string               `json:"password" binding:"required,min=6"`
	FullName     string               `json:"full_name" binding:"required"`
	Role         models.Role          `json:"role" binding:"required"`
	EmployeeType *models.EmployeeType `json:"employee_type" binding:"omitempty"`
}

// GetAllUsers returns all users (admin only)
func GetAllUsers(c *gin.Context) {
	var users []models.User

	// Get query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Query users with pagination
	query := database.DB.Model(&models.User{})

	// Get total count
	var total int64
	query.Count(&total)

	// Get users
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetUserByID returns a user by ID (admin only)
func GetUserByID(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user (admin only)
func CreateUser(c *gin.Context) {
	var req CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role. Must be one of: admin, projectmanager, employee, finance",
		})
		return
	}

	// Validate employee type for employee role
	if req.Role == models.RoleEmployee {
		if req.EmployeeType == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "employee_type is required for employee role",
			})
			return
		}
		if !isValidEmployeeType(*req.EmployeeType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid employee_type. Must be one of: fulltime, parttime, freelance",
			})
			return
		}
	} else {
		// Clear employee type for non-employee roles
		req.EmployeeType = nil
	}

	// Check if email already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		Password:     hashedPassword,
		FullName:     req.FullName,
		Role:         req.Role,
		EmployeeType: req.EmployeeType,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates a user (admin only)
func UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields if provided
	if req.Email != nil {
		// Check if email already exists (excluding current user)
		var existingUser models.User
		if err := database.DB.Where("email = ? AND id != ?", *req.Email, id).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			return
		} else if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		user.Email = *req.Email
	}

	if req.Password != nil {
		hashedPassword, err := utils.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = hashedPassword
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}

	if req.Role != nil {
		// Validate role
		if !isValidRole(*req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid role. Must be one of: admin, projectmanager, employee, finance",
			})
			return
		}
		user.Role = *req.Role

		// Handle employee type based on new role
		if *req.Role == models.RoleEmployee {
			if req.EmployeeType == nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "employee_type is required when role is employee",
				})
				return
			}
			if !isValidEmployeeType(*req.EmployeeType) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid employee_type. Must be one of: fulltime, parttime, freelance",
				})
				return
			}
			user.EmployeeType = req.EmployeeType
		} else {
			// Clear employee type for non-employee roles
			user.EmployeeType = nil
		}
	} else if req.EmployeeType != nil {
		// If role is not being updated but employee_type is, validate current role
		if user.Role != models.RoleEmployee {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "employee_type can only be set for employee role",
			})
			return
		}
		if !isValidEmployeeType(*req.EmployeeType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid employee_type. Must be one of: fulltime, parttime, freelance",
			})
			return
		}
		user.EmployeeType = req.EmployeeType
	}

	// Save updated user
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	var user models.User

	// Find user including soft-deleted rows
	if err := database.DB.Unscoped().First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Prevent admin from deleting themselves
	currentUser, exists := c.Get("user")
	if exists {
		currentUserModel := currentUser.(*models.User)
		if currentUserModel.ID == user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete your own account"})
			return
		}
	}

	// HARD DELETE
	if err := database.DB.Unscoped().Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User permanently deleted"})
}
