package handlers

import (
	"net/http"
	"strconv"

	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateProjectRequest represents create project request
type CreateProjectRequest struct {
	Name        string            `json:"name" binding:"required"`
	ClientName  string            `json:"client_name" binding:"required"`
	ClientEmail *string           `json:"client_email,omitempty"`
	BudgetType  models.BudgetType `json:"budget_type" binding:"required"`

	// Berdasarkan Proyek (Project Based)
	BudgetedHours       *int64 `json:"budgeted_hours,omitempty"`
	HourThreshold       *int64 `json:"hour_threshold,omitempty"`
	BudgetCost          *int64 `json:"budget_cost,omitempty"`
	BudgetCostThreshold *int64 `json:"budget_cost_threshold,omitempty"`
	BudgetRevenue       *int64 `json:"budget_revenue,omitempty"`

	// Berdasarkan Pengguna (User Based)
	CostPerHour *int64 `json:"cost_per_hour,omitempty"`
	RatePerHour *int64 `json:"rate_per_hour,omitempty"`
}

// UpdateProjectRequest represents update project request
type UpdateProjectRequest struct {
	Name        *string            `json:"name,omitempty"`
	ClientName  *string            `json:"client_name,omitempty"`
	ClientEmail *string            `json:"client_email,omitempty"`
	BudgetType  *models.BudgetType `json:"budget_type,omitempty"`

	// Berdasarkan Proyek (Project Based)
	BudgetedHours       *int64 `json:"budgeted_hours,omitempty"`
	HourThreshold       *int64 `json:"hour_threshold,omitempty"`
	BudgetCost          *int64 `json:"budget_cost,omitempty"`
	BudgetCostThreshold *int64 `json:"budget_cost_threshold,omitempty"`
	BudgetRevenue       *int64 `json:"budget_revenue,omitempty"`

	// Berdasarkan Pengguna (User Based)
	CostPerHour *int64 `json:"cost_per_hour,omitempty"`
	RatePerHour *int64 `json:"rate_per_hour,omitempty"`
}

// isValidBudgetType checks if budget type is valid
func isValidBudgetType(budgetType models.BudgetType) bool {
	validTypes := []models.BudgetType{
		models.BudgetTypeProject,
		models.BudgetTypeUser,
	}
	for _, validType := range validTypes {
		if budgetType == validType {
			return true
		}
	}
	return false
}

// validateProjectFields validates project fields based on budget type
func validateProjectFields(req *CreateProjectRequest) string {
	if !isValidBudgetType(req.BudgetType) {
		return "Invalid budget_type. Must be one of: project_based, user_based"
	}

	if req.BudgetType == models.BudgetTypeProject {
		// For project-based, validate project-based fields
		if req.BudgetedHours == nil {
			return "budgeted_hours is required for project_based budget type"
		}
		// Clear user-based fields
		req.CostPerHour = nil
		req.RatePerHour = nil
	} else if req.BudgetType == models.BudgetTypeUser {
		// For user-based, validate user-based fields
		if req.CostPerHour == nil || req.RatePerHour == nil {
			return "cost_per_hour and rate_per_hour are required for user_based budget type"
		}
		// Clear project-based fields
		req.BudgetedHours = nil
		req.HourThreshold = nil
		req.BudgetCost = nil
		req.BudgetCostThreshold = nil
		req.BudgetRevenue = nil
	}

	return ""
}

// validateUpdateProjectFields validates update project fields based on budget type
func validateUpdateProjectFields(req *UpdateProjectRequest, project *models.Project) string {
	budgetType := project.BudgetType
	if req.BudgetType != nil {
		if !isValidBudgetType(*req.BudgetType) {
			return "Invalid budget_type. Must be one of: project_based, user_based"
		}
		budgetType = *req.BudgetType
	}

	if budgetType == models.BudgetTypeProject {
		// Clear user-based fields if switching to project-based
		if req.BudgetType != nil {
			req.CostPerHour = nil
			req.RatePerHour = nil
		}
	} else if budgetType == models.BudgetTypeUser {
		// Clear project-based fields if switching to user-based
		if req.BudgetType != nil {
			req.BudgetedHours = nil
			req.HourThreshold = nil
			req.BudgetCost = nil
			req.BudgetCostThreshold = nil
			req.BudgetRevenue = nil
		}
	}

	return ""
}

// CreateProject creates a new project
func CreateProject(c *gin.Context) {
	var req CreateProjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate project fields
	if errMsg := validateProjectFields(&req); errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// Create project
	project := models.Project{
		Name:                req.Name,
		ClientName:          req.ClientName,
		ClientEmail:         req.ClientEmail,
		BudgetType:          req.BudgetType,
		BudgetedHours:       req.BudgetedHours,
		HourThreshold:       req.HourThreshold,
		BudgetCost:          req.BudgetCost,
		BudgetCostThreshold: req.BudgetCostThreshold,
		BudgetRevenue:       req.BudgetRevenue,
		CostPerHour:         req.CostPerHour,
		RatePerHour:         req.RatePerHour,
	}

	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetAllProjects returns all projects
func GetAllProjects(c *gin.Context) {
	var projects []models.Project

	// Get query parameters for pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	// Query projects with pagination
	query := database.DB.Model(&models.Project{})

	// Get total count
	var total int64
	query.Count(&total)

	// Get projects
	if err := query.Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": projects,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

// GetProjectByID returns a project by ID
func GetProjectByID(c *gin.Context) {
	id := c.Param("id")

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates a project
func UpdateProject(c *gin.Context) {
	projectId := c.Param("projectId")

	var project models.Project
	if err := database.DB.First(&project, projectId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var req UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate update fields
	if errMsg := validateUpdateProjectFields(&req, &project); errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// Update fields
	if req.Name != nil {
		project.Name = *req.Name
	}

	if req.ClientName != nil {
		project.ClientName = *req.ClientName
	}

	if req.ClientEmail != nil {
		project.ClientEmail = req.ClientEmail
	}

	if req.BudgetType != nil {
		project.BudgetType = *req.BudgetType
	}

	// Update project-based fields
	if req.BudgetedHours != nil {
		project.BudgetedHours = req.BudgetedHours
	}
	if req.HourThreshold != nil {
		project.HourThreshold = req.HourThreshold
	}
	if req.BudgetCost != nil {
		project.BudgetCost = req.BudgetCost
	}
	if req.BudgetCostThreshold != nil {
		project.BudgetCostThreshold = req.BudgetCostThreshold
	}
	if req.BudgetRevenue != nil {
		project.BudgetRevenue = req.BudgetRevenue
	}

	// Update user-based fields
	if req.CostPerHour != nil {
		project.CostPerHour = req.CostPerHour
	}
	if req.RatePerHour != nil {
		project.RatePerHour = req.RatePerHour
	}

	// Save updated project
	if err := database.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project
func DeleteProject(c *gin.Context) {
	id := c.Param("id")

	var project models.Project
	if err := database.DB.First(&project, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Soft delete
	if err := database.DB.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}
