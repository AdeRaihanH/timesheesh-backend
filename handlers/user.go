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
	IsActive     *bool                `json:"is_active,omitempty"`
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

// CREATE USER (Admin Only)
// POST /api/admin/users
func CreateUser(c *gin.Context) {
	var req CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Validate Role Enum
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role. Must be one of: admin, projectmanager, employee, finance",
		})
		return
	}

	// 2. Validate Employee Type (Revised Logic)
	// Admin = NULL. Role Lain = WAJIB ISI.
	if req.Role == models.RoleAdmin {
		req.EmployeeType = nil
	} else {
		// Jika ProjectManager, Finance, atau Employee
		if req.EmployeeType == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "employee_type is required for role " + string(req.Role),
			})
			return
		}
		if !isValidEmployeeType(*req.EmployeeType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid employee_type. Must be one of: fulltime, parttime, freelance",
			})
			return
		}
	}

	// 3. Check Email Duplication
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 4. Hash Password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// 5. Create User
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

	user.Password = ""
	c.JSON(http.StatusCreated, user)
}


// UPDATE USER (Admin Only)
// PUT /api/admin/users/:id
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

	// Update Basic Fields
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Email != nil {
		var existingUser models.User
		if err := database.DB.Where("email = ? AND id != ?", *req.Email, id).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
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

	// === LOGIC UPDATE ROLE & EMPLOYEE TYPE ===
	
	// Tentukan Role Target (Apakah berubah atau tetap?)
	targetRole := user.Role
	if req.Role != nil {
		if !isValidRole(*req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		targetRole = *req.Role
	}

	// Tentukan EmployeeType Target
	var targetEmpType *models.EmployeeType = user.EmployeeType
	if req.EmployeeType != nil {
		targetEmpType = req.EmployeeType
	}

	// Validasi Kombinasi Baru
	if targetRole == models.RoleAdmin {
		// Jika targetnya Admin -> EmployeeType harus NULL
		targetEmpType = nil
	} else {
		// Jika targetnya Non-Admin -> EmployeeType WAJIB ADA
		if targetEmpType == nil {
			// Case: Dulunya Admin (Type=nil) -> Diubah jadi PM, tapi lupa kirim employee_type baru
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "employee_type is required when role is set to " + string(targetRole),
			})
			return
		}
		// Validasi Enum
		if !isValidEmployeeType(*targetEmpType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee_type"})
			return
		}
	}

	// Apply Changes
	if req.Role != nil {
		user.Role = *req.Role
	}
	user.EmployeeType = targetEmpType

	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

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
