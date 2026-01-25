package handlers

import (
	"net/http"

	"timesheesh-backend/database"
	"timesheesh-backend/models"
	"timesheesh-backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email        string               `json:"email" binding:"required,email"`
	Password     string               `json:"password" binding:"required,min=6"`
	FullName     string               `json:"full_name" binding:"required"`
	Role         models.Role          `json:"role" binding:"required"`
	EmployeeType *models.EmployeeType `json:"employee_type" binding:"omitempty"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// isValidRole checks if role is valid
func isValidRole(role models.Role) bool {
	validRoles := []models.Role{
		models.RoleAdmin,
		models.RoleProjectManager,
		models.RoleEmployee,
		models.RoleFinance,
	}
	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

// isValidEmployeeType checks if employee type is valid
func isValidEmployeeType(empType models.EmployeeType) bool {
	validTypes := []models.EmployeeType{
		models.EmployeeTypeFulltime,
		models.EmployeeTypeParttime,
		models.EmployeeTypeFreelance,
	}
	for _, validType := range validTypes {
		if empType == validType {
			return true
		}
	}
	return false
}

// hasAdmin checks if there is at least one admin in the database
func hasAdmin() bool {
	var count int64
	database.DB.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&count)
	return count > 0
}

// Register handles user registration
// It allows public registration ONLY for the very first user (who must be an Admin).
// Once an admin exists, this public endpoint is locked.
// POST /api/auth/register
func Register(c *gin.Context) {
	var req RegisterRequest

	// Bind JSON input to struct
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SECURITY CHECK: INITIAL SETUP ONLY 
	// Check if any admin already exists in the database
	if hasAdmin() {
		// If an admin exists, public registration is completely closed.
		// New users must be created by an Admin via the User Management module (POST /api/admin/users).
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Public registration is closed. Please contact the Administrator to create an account.",
		})
		return
	}

	// For the first user ever, enforce the Role to be Admin.
	// We cannot allow the first user to be a regular employee, or the system will be inaccessible.
	if req.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "System not initialized. The first user MUST be an 'admin'.",
		})
		return
	}
	
	// Validate user role enum
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role. Must be one of: admin, projectmanager, employee, finance",
		})
		return
	}

	// Aturan: Admin = NULL, Role Lain = WAJIB ISI
	if req.Role == models.RoleAdmin {
		// Admin tidak memiliki tipe kepegawaian
		req.EmployeeType = nil 
	} else {
		// Untuk ProjectManager, Employee, & Finance -> Wajib isi Employee Type
		if req.EmployeeType == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "employee_type is required for role " + string(req.Role),
			})
			return
		}
		// Validasi Enum EmployeeType
		if !isValidEmployeeType(*req.EmployeeType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid employee_type. Must be one of: fulltime, parttime, freelance",
			})
			return
		}
	}

	// Check for duplicate email
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking email"})
		return
	}

	// Hash the password securely
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create the user object
	user := models.User{
		Email:        req.Email,
		Password:     hashedPassword,
		FullName:     req.FullName,
		Role:         req.Role,
		EmployeeType: req.EmployeeType,
	}

	// Save to database
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Auto-login: Generate JWT token for the new user immediately
	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	// Prepare response (hide password)
	user.Password = ""

	c.JSON(http.StatusCreated, AuthResponse{
		Token: token,
		User:  user,
	})
}

// Login handles user login
// POST /api/auth/login
func Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user by email
	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check password
	if !utils.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate token
	token, err := utils.GenerateToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Remove password from response
	user.Password = ""

	c.JSON(http.StatusOK, AuthResponse{
		Token: token,
		User:  user,
	})
}

// GetProfile returns current user profile
// GET /api/profile
func GetProfile(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userModel := user.(*models.User)
	userModel.Password = ""

	c.JSON(http.StatusOK, userModel)
}

// GetDashboard returns dashboard data based on user role
func GetDashboard(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	userModel := user.(*models.User)
	role := userModel.Role

	// Dashboard response based on role
	var dashboardData gin.H

	switch role {
	case models.RoleAdmin:
		dashboardData = gin.H{
			"role":    "admin",
			"message": "Admin Dashboard",
			"features": []string{
				"User Management",
				"System Configuration",
				"All Projects Access",
				"Reports & Analytics",
			},
		}
	case models.RoleProjectManager:
		dashboardData = gin.H{
			"role":    "projectmanager",
			"message": "Project Manager Dashboard",
			"features": []string{
				"Project Management",
				"Team Management",
				"Task Assignment",
				"Project Reports",
			},
		}
	case models.RoleEmployee:
		employeeType := "N/A"
		if userModel.EmployeeType != nil {
			employeeType = string(*userModel.EmployeeType)
		}
		dashboardData = gin.H{
			"role":          "employee",
			"message":       "Employee Dashboard",
			"employee_type": employeeType,
			"features": []string{
				"Time Tracking",
				"Task View",
				"Personal Reports",
				"Profile Management",
			},
		}
	case models.RoleFinance:
		dashboardData = gin.H{
			"role":    "finance",
			"message": "Finance/Management Dashboard",
			"features": []string{
				"Company Overview",
				"Financial Reports",
				"Budget Management",
				"Revenue Analysis",
				"Team Performance",
				"Strategic Planning",
			},
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role"})
		return
	}

	c.JSON(http.StatusOK, dashboardData)
}

// Struct Request
type ProxyLoginRequest struct {
	TargetUserID uint `json:"target_user_id" binding:"required"`
}

// ProxyLogin allows Admin to login as another user
func ProxyLogin(c *gin.Context) {
	var req ProxyLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Cari User Target
	var user models.User
	if err := database.DB.First(&user, req.TargetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Target user not found"})
		return
	}

	// 2. Generate Token (Milik User Tersebut)
	token, err := utils.GenerateToken(&user) // Asumsi fungsi ini ada di folder utils
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 3. Return Token seolah-olah user yang login
	c.JSON(http.StatusOK, gin.H{
		"message": "Proxy login successful",
		"token":   token,
		"user":    user,
	})
}