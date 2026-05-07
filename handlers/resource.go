package handlers

import (
	"net/http"
	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
)

type CreateResourceInput struct {
	ProjectID uint               `json:"project_id" binding:"required"`
	Type      models.ResourceType `json:"type" binding:"required"` 
	Details   string             `json:"details" binding:"required"`
}

type UpdateResourceStatusInput struct {
	Status models.RequestStatus `json:"status" binding:"required"` 
}

// 1. Create Resource Request (Function Handler)
func CreateResourceRequest(c *gin.Context) {
	// Ambil user dari token (Pointer)
	user := c.MustGet("user").(*models.User)

	// Gunakan struct input yang baru
	var req CreateResourceInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi project
	var project models.Project
	if err := database.DB.First(&project, req.ProjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	resourceReq := models.ResourceRequest{
		ProjectID:   req.ProjectID,
		UserID:      user.ID,
		Type:        req.Type,
		Details:     req.Details,
		Status:      models.RequestStatusPending,
	}

	if err := database.DB.Create(&resourceReq).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	c.JSON(http.StatusCreated, resourceReq)
}

// 2. Get Resource Requests (Inbox)
func GetResourceRequests(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	var requests []models.ResourceRequest
	query := database.DB.Preload("Project").Preload("Requester")

	// Logic Filter: Admin lihat semua, PM lihat punya sendiri
	if user.Role != "admin" {
		query = query.Where("requester_id = ?", user.ID)
	}

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	if err := query.Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch requests"})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// 3. Update Status (Admin Approval)
func UpdateResourceStatus(c *gin.Context) {
	id := c.Param("id")
	
	// Gunakan struct input yang baru
	var req UpdateResourceStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var resourceReq models.ResourceRequest
	if err := database.DB.First(&resourceReq, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
		return
	}

	resourceReq.Status = req.Status
	
	if err := database.DB.Save(&resourceReq).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, resourceReq)
}