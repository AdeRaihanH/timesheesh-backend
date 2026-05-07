package handlers

import (
	"net/http"
	"time"

	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Request Structures
type CreateContractRequest struct {
	UserID       uint                `json:"user_id" binding:"required"`
	ContractType models.ContractType `json:"contract_type" binding:"required"` // monthly, hourly, termin
	RateAmount   int64               `json:"rate_amount" binding:"required"`   // Gaji/Rate
	StartDate    string              `json:"start_date" binding:"required"`    // YYYY-MM-DD
	EndDate      *string             `json:"end_date,omitempty"`               // Optional
	IsActive     *bool               `json:"is_active,omitempty"`
}

type UpdateContractRequest struct {
	ContractType *models.ContractType `json:"contract_type,omitempty"`
	RateAmount   *int64               `json:"rate_amount,omitempty"`
	StartDate    *string              `json:"start_date,omitempty"`
	EndDate      *string              `json:"end_date,omitempty"`
	IsActive     *bool                `json:"is_active,omitempty"`
}

// 1. Create Contract (Admin Only)
// Endpoint: POST /api/admin/contracts
func CreateContract(c *gin.Context) {
	var req CreateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate User Exists
	var user models.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Parse Dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format (use YYYY-MM-DD)"})
		return
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsedEnd, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
			return
		}
		endDate = &parsedEnd
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	contract := models.Contract{
		UserID:       req.UserID,
		ContractType: req.ContractType,
		RateAmount:   req.RateAmount,
		StartDate:    startDate,
		EndDate:      endDate,
		IsActive:     isActive,
	}

	if err := database.DB.Create(&contract).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contract"})
		return
	}

	c.JSON(http.StatusCreated, contract)
}

// 2. Get User Contracts (Admin Only)
// Endpoint: GET /api/admin/users/:id/contracts
func GetUserContracts(c *gin.Context) {
	userId := c.Param("id")

	var contracts []models.Contract
	// Ambil semua kontrak milik user tertentu, urutkan dari yang terbaru
	if err := database.DB.Where("user_id = ?", userId).Order("created_at desc").Find(&contracts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contracts"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}

// 3. Update Contract (Admin Only)
// Endpoint: PUT /api/admin/contracts/:id
func UpdateContract(c *gin.Context) {
	id := c.Param("id")

	var contract models.Contract
	if err := database.DB.First(&contract, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var req UpdateContractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update Fields jika dikirim
	if req.ContractType != nil { contract.ContractType = *req.ContractType }
	if req.RateAmount != nil   { contract.RateAmount = *req.RateAmount }
	if req.IsActive != nil     { contract.IsActive = *req.IsActive }
	
	if req.StartDate != nil {
		date, _ := time.Parse("2006-01-02", *req.StartDate)
		contract.StartDate = date
	}
	if req.EndDate != nil {
		date, _ := time.Parse("2006-01-02", *req.EndDate)
		contract.EndDate = &date
	}

	if err := database.DB.Save(&contract).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contract"})
		return
	}

	c.JSON(http.StatusOK, contract)
}

// 4. Delete Contract (Admin Only)
// Endpoint: DELETE /api/admin/contracts/:id
func DeleteContract(c *gin.Context) {
	id := c.Param("id")
	
	// Hard Delete
	if err := database.DB.Delete(&models.Contract{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete contract"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Contract deleted"})
}

// 5. Get My Contracts (Logged In User)
// Endpoint: GET /api/my-contracts
func GetMyContracts(c *gin.Context) {
	// Ambil user yang sedang login dari token
	user := c.MustGet("user").(*models.User)

	var contracts []models.Contract
	// Cari kontrak milik user tersebut
	if err := database.DB.Where("user_id = ?", user.ID).Order("created_at desc").Find(&contracts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contracts"})
		return
	}

	c.JSON(http.StatusOK, contracts)
}