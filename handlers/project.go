package handlers

import (
	"net/http"
	"strconv"
	"time"

	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Request Structures (Sangat Sederhana)
type CreateProjectRequest struct {
	Name                string   `json:"name" binding:"required"`
	ClientName          string   `json:"client_name" binding:"required"`
	ClientEmail         *string  `json:"client_email,omitempty"`
	BudgetRevenue       *float64 `json:"budget_revenue,omitempty"`       // Nilai Kontrak
	BudgetCost          *float64 `json:"budget_cost,omitempty"`          // Modal Project
	BudgetCostThreshold *float64 `json:"budget_cost_threshold,omitempty"`// Batas Warning
}

type UpdateProjectRequest struct {
	Name                *string  `json:"name,omitempty"`
	ClientName          *string  `json:"client_name,omitempty"`
	ClientEmail         *string  `json:"client_email,omitempty"`
	Status              *models.ProjectStatus `json:"status,omitempty"`
	BudgetRevenue       *float64 `json:"budget_revenue,omitempty"`
	BudgetCost          *float64 `json:"budget_cost,omitempty"`
	BudgetCostThreshold *float64 `json:"budget_cost_threshold,omitempty"`
}

// 1. CREATE PROJECT
// Endpoint: POST /api/project/create
func CreateProject(c *gin.Context) {
	var req CreateProjectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mapping ke Model Database
	project := models.Project{
		Name:                req.Name,
		ClientName:          req.ClientName,
		ClientEmail:         req.ClientEmail,
		BudgetRevenue:       req.BudgetRevenue,
		BudgetCost:          req.BudgetCost,
		BudgetCostThreshold: req.BudgetCostThreshold,
		Status:              models.ProjectStatusActive, // Default Active
	}

	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// 2. UPDATE PROJECT
// Endpoint: PUT /api/project/update/:projectId
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

	// Update Fields (Partial Update)
	if req.Name != nil { project.Name = *req.Name }
	if req.ClientName != nil { project.ClientName = *req.ClientName }
	if req.ClientEmail != nil { project.ClientEmail = req.ClientEmail }
	if req.Status != nil { project.Status = *req.Status }

	// Update Financial Fields
	if req.BudgetRevenue != nil { project.BudgetRevenue = req.BudgetRevenue }
	if req.BudgetCost != nil { project.BudgetCost = req.BudgetCost }
	if req.BudgetCostThreshold != nil { project.BudgetCostThreshold = req.BudgetCostThreshold }

	if err := database.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// 3. DELETE PROJECT
// Endpoint: DELETE /api/project/delete/:id
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

	// Soft Delete
	if err := database.DB.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// 4. GET ALL PROJECTS (Filtered by Role)
// Endpoint: GET /api/getproject
func GetAllProjects(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var projects []models.Project

	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Project{})

	// Admin DAN Finance bisa melihat semua project.
	// PM dan Employee hanya melihat project milik mereka.
	if user.Role != models.RoleAdmin && user.Role != models.RoleFinance {
		query = query.Joins("JOIN project_members ON project_members.project_id = projects.id").
			Where("project_members.user_id = ?", user.ID).
			Distinct("projects.*")
	}

	var total int64
	query.Count(&total)

	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&projects).Error; err != nil {
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

// 5. GET PROJECT BY ID
// Endpoint: GET /api/getproject/:id
func GetProjectByID(c *gin.Context) {
	id := c.Param("id")
	user := c.MustGet("user").(*models.User)

	var project models.Project

	// Preload Members untuk cek akses
	if err := database.DB.Preload("Members.User").First(&project, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Aturan: Admin & Finance BOLEH lihat semua.
	// PM & Employee HARUS member project.
	if user.Role != models.RoleAdmin && user.Role != models.RoleFinance {
		
		isMember := false
		// Cek apakah user ada di daftar member
		for _, member := range project.Members {
			if member.UserID == user.ID {
				isMember = true
				break
			}
		}

		// Jika bukan Admin/Finance DAN bukan Member -> Tolak
		if !isMember {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to view this project details"})
			return
		}
	}

	// Sanitasi Password Member
	for i := range project.Members {
		project.Members[i].User.Password = ""
	}

	c.JSON(http.StatusOK, project)
}

// Request Structure
type AssignMemberRequest struct {
	ProjectID     uint   `json:"project_id" binding:"required"`
	UserID        uint   `json:"user_id" binding:"required"`
	RoleInProject string `json:"role_in_project" binding:"required"`

	// === OPSIONAL: Custom Rate (Membuat Contract Baru) ===
	CustomRate    *int64                `json:"custom_rate,omitempty"`
	ContractType  *models.ContractType  `json:"contract_type,omitempty"`
	PaymentScheme *models.PaymentScheme `json:"payment_scheme,omitempty"`
}

// 1. ASSIGN MEMBER (Admin Only)
// Endpoint: POST /api/project/assign
func AssignMember(c *gin.Context) {
	// A. Security Check: Admin Only
	user := c.MustGet("user").(*models.User)
	if user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Admin can assign members with contracts"})
		return
	}

	var req AssignMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mulai Transaksi Database
	tx := database.DB.Begin()

	// B. Validasi Data Dasar (Project & User Ada)
	var project models.Project
	if err := tx.First(&project, req.ProjectID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var targetUser models.User
	if err := tx.First(&targetUser, req.UserID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// C. Cek Duplikasi Member
	var existingMember models.ProjectMember
	if err := tx.Where("project_id = ? AND user_id = ?", req.ProjectID, req.UserID).First(&existingMember).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "User is already assigned to this project"})
		return
	}

	// D. LOGIC CONTRACT (Custom vs Global)
	// Jika custom_rate diisi, kita buatkan kontrak baru khusus project ini.
	if req.CustomRate != nil {
		// Validasi kelengkapan data kontrak
		if req.ContractType == nil || req.PaymentScheme == nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "contract_type and payment_scheme are required for custom rate"})
			return
		}

		newContract := models.Contract{
			UserID:        req.UserID,
			ProjectID:     &req.ProjectID, // Link ke Project (Temporary)
			RateAmount:    *req.CustomRate,
			ContractType:  *req.ContractType,
			PaymentScheme: *req.PaymentScheme,
			StartDate:     time.Now(), // Efektif mulai hari ini/saat assign
			IsActive:      true,
		}

		if err := tx.Create(&newContract).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create custom contract"})
			return
		}
	} 
	// Jika custom_rate == nil, sistem tidak melakukan apa-apa terhadap tabel Contract.
	// Artinya user akan otomatis menggunakan Contract Global yang sudah dia punya (jika ada).

	// E. Simpan Member
	member := models.ProjectMember{
		ProjectID:     req.ProjectID,
		UserID:        req.UserID,
		RoleInProject: req.RoleInProject,
	}

	if err := tx.Create(&member).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign member"})
		return
	}

	// Commit Transaksi
	tx.Commit()

	c.JSON(http.StatusCreated, gin.H{
		"message": "Member assigned successfully",
		"data":    member,
		"custom_contract": req.CustomRate != nil, // Info flag apakah pakai custom rate
	})
}

// 2. GET PROJECT MEMBERS
// Endpoint: GET /api/project/:projectId/members
func GetProjectMembers(c *gin.Context) {
	projectId := c.Param("projectId")
	user := c.MustGet("user").(*models.User)

	// A. Security Check (Role Based Access)
	// Admin & Finance: Boleh lihat semua project.
	// PM & Employee: Hanya boleh lihat jika mereka anggota project tersebut.
	if user.Role != models.RoleAdmin && user.Role != models.RoleFinance {
		var count int64
		// Cek apakah user yang login terdaftar di project ini
		database.DB.Model(&models.ProjectMember{}).
			Where("project_id = ? AND user_id = ?", projectId, user.ID).
			Count(&count)

		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to view members of this project"})
			return
		}
	}

	var members []models.ProjectMember
	// B. Ambil Data
	// Preload "User" untuk menampilkan nama/email.
	if err := database.DB.Where("project_id = ?", projectId).Preload("User").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
		return
	}

	// C. Sanitasi Password
	for i := range members {
		members[i].User.Password = ""
	}

	c.JSON(http.StatusOK, members)
}

// 3. REMOVE MEMBER (Admin Only)
// Endpoint: DELETE /api/project/member/:id
// Note: :id di sini adalah ID dari tabel `project_members` (bukan UserID)
func RemoveMember(c *gin.Context) {
	id := c.Param("id")
	user := c.MustGet("user").(*models.User)

	// A. Security Check: Admin Only
	if user.Role != models.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only Admin can remove members"})
		return
	}

	// Mulai Transaksi Database
	tx := database.DB.Begin()

	// B. Cari Data Member Dulu
	var member models.ProjectMember
	if err := tx.First(&member, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Member record not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// C. LOGIC CONTRACT CLEANUP (PENTING)
	// Jika user ini punya Kontrak Custom (Temporary) di project ini, kontraknya harus dimatikan.
	// Kita cari kontrak yang UserID & ProjectID-nya cocok, dan masih aktif.
	if err := tx.Model(&models.Contract{}).
		Where("user_id = ? AND project_id = ? AND is_active = ?", member.UserID, member.ProjectID, true).
		Updates(map[string]interface{}{
			"is_active": false,
			"end_date":  time.Now(), // Set end date jadi hari ini
		}).Error; err != nil {
		
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate project contract"})
		return
	}
	// Note: Kontrak Global (yang project_id = NULL) tidak akan tersentuh logic ini.

	// D. Hapus Member dari Project
	if err := tx.Delete(&member).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Member removed and associated contract terminated (if any)"})
}
