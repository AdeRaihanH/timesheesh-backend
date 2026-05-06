package handlers

import (
	"log"
	"net/http"
	"time"
	"timesheesh-backend/database"
	"timesheesh-backend/models"
	"github.com/gin-gonic/gin"
)

// 1. Clock In Request 
type ClockInRequest struct {
	ProjectID   uint    `json:"project_id" binding:"required"`
	TaskID      *uint   `json:"task_id"` // Opsional
	FaceScore   float64 `json:"face_similarity_score"`
	Description string  `json:"description"` // Baru dari Tracker
}

type ClockOutRequest struct {
	Description string `json:"description"` // Made optional to support quick-restart/manual stop
}

type ApprovalRequest struct {
	Status        models.TimesheetStatus `json:"status" binding:"required"`
	RejectionNote string                 `json:"rejection_note"`
}

type CreateManualLogRequest struct {
	ProjectID       uint   `json:"project_id" binding:"required"`
	TaskID          *uint  `json:"task_id"`
	Date            string `json:"date" binding:"required"` // Format: YYYY-MM-DD
	DurationSeconds int    `json:"duration_seconds" binding:"required"`
	Description     string `json:"description"`
}

type DeleteWeekLogsRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	TaskID    *uint  `json:"task_id"`
	WeekStart string `json:"week_start" binding:"required"` // Mono: YYYY-MM-DD
}

type UpdateTimesheetRequest struct {
	Description     string `json:"description"`
	DurationSeconds *int   `json:"duration_seconds"`
}

// 1. Clock In (Hanya Mulai Waktu)
func ClockIn(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var req ClockInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek sesi aktif
	var activeSession models.Timesheet
	if err := database.DB.Where("user_id = ? AND clock_out IS NULL", user.ID).First(&activeSession).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have an active session. Please clock out first."})
		return
	}

	timesheet := models.Timesheet{
		UserID:              user.ID,
		ProjectID:           req.ProjectID,
		TaskID:              req.TaskID,
		Description:         req.Description,
		FaceSimilarityScore: req.FaceScore,
		ClockIn:             time.Now(),
		Status:              models.TimesheetPending,
	}

	if err := database.DB.Create(&timesheet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clock in failed"})
		return
	}

	// Update Status Task jadi In Progress
	if req.TaskID != nil {
		var task models.Task
		if err := database.DB.First(&task, *req.TaskID).Error; err == nil {
			if task.Status == models.TaskTodo {
				task.Status = models.TaskInProgress
				database.DB.Save(&task)
			}
		}
	}

	c.JSON(http.StatusCreated, timesheet)
}

// 2. Clock Out (Isi Deskripsi/Log)
func ClockOut(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	// Validasi Input: User WAJIB mengisi apa yang dikerjakan
	var req ClockOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid clock-out request: " + err.Error(),
		})
		return
	}

	// Cari Sesi Aktif
	var timesheet models.Timesheet
	if err := database.DB.Where("user_id = ? AND clock_out IS NULL", user.ID).First(&timesheet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active clock-in session found"})
		return
	}

	// Update Data
	now := time.Now()
	timesheet.ClockOut = &now
	
	// Hitung Durasi
	duration := now.Sub(timesheet.ClockIn)
	timesheet.DurationMinutes = int(duration.Minutes())
	timesheet.DurationSeconds = int(duration.Seconds())

	timesheet.Description = req.Description

	if err := database.DB.Save(&timesheet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Clock out failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Clock out successful. Work log saved.",
		"data":    timesheet,
	})
}

// 7. Manual Log Creation (From Grid)
func CreateManualLog(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var req CreateManualLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Timesheet] CreateManualLog: binding failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	log.Printf("[Timesheet] Received manual log request: %+v", req)

	// Parse date
	logDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		log.Printf("[Timesheet] CreateManualLog: date parse failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
		return
	}

	// Check if already exists for this day (Upsert)
	var timesheet models.Timesheet
	nextDay := logDate.AddDate(0, 0, 1)
	query := database.DB.Where("user_id = ? AND project_id = ? AND clock_in >= ? AND clock_in < ?", user.ID, req.ProjectID, logDate, nextDay)
	if req.TaskID != nil {
		query = query.Where("task_id = ?", *req.TaskID)
	} else {
		query = query.Where("task_id IS NULL")
	}

	result := query.First(&timesheet)

	// Calculate ClockOut
	clockIn := logDate
	clockOut := clockIn.Add(time.Duration(req.DurationSeconds) * time.Second)

	if result.Error == nil {
		// Update Existing
		timesheet.DurationSeconds = req.DurationSeconds
		timesheet.DurationMinutes = req.DurationSeconds / 60
		timesheet.ClockOut = &clockOut
		timesheet.Description = req.Description
		database.DB.Save(&timesheet)
		c.JSON(http.StatusOK, timesheet)
		return
	}

	// Create New
	timesheet = models.Timesheet{
		UserID:          user.ID,
		ProjectID:       req.ProjectID,
		TaskID:          req.TaskID,
		Description:     req.Description,
		ClockIn:         clockIn,
		ClockOut:        &clockOut,
		DurationSeconds: req.DurationSeconds,
		DurationMinutes: req.DurationSeconds / 60,
		Status:          models.TimesheetPending,
	}

	if err := database.DB.Create(&timesheet).Error; err != nil {
		log.Printf("[Timesheet] CreateManualLog: create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create manual log", "details": err.Error()})
		return
	}
	log.Printf("[Timesheet] Manual log created successfully: ID=%d", timesheet.ID)

	c.JSON(http.StatusCreated, timesheet)
}

// 3. Get User History (My Logs)
func GetMyTimesheets(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var logs []models.Timesheet
	
	database.DB.Preload("Task").Preload("Project").Where("user_id = ?", user.ID).Order("clock_in desc").Find(&logs)
	c.JSON(http.StatusOK, logs)
}

// 3.1. Delete Week Logs (X button in Grid)
func DeleteWeekLogs(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	var req DeleteWeekLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse week_start
	startDate, err := time.Parse("2006-01-02", req.WeekStart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid week_start format. Use YYYY-MM-DD"})
		return
	}
	endDate := startDate.AddDate(0, 0, 7) // Full week

	query := database.DB.Where("user_id = ? AND project_id = ? AND clock_in >= ? AND clock_in < ?", user.ID, req.ProjectID, startDate, endDate)
	if req.TaskID != nil {
		query = query.Where("task_id = ?", *req.TaskID)
	} else {
		query = query.Where("task_id IS NULL")
	}

	if err := query.Delete(&models.Timesheet{}).Error; err != nil {
		log.Printf("[Timesheet] DeleteWeekLogs failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete week logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Week logs deleted successfully"})
}

// 4. Approval Inbox (PM melihat pending request)
func GetPendingTimesheets(c *gin.Context) {
	// Filter: Harusnya hanya project milik PM, tapi untuk tahap ini kita tampilkan semua pending dulu
	var logs []models.Timesheet
	
	database.DB.Preload("User").Preload("Task").
		Where("status = ?", models.TimesheetPending).
		Order("clock_in asc").
		Find(&logs)
	
	// Sanitasi
	for i := range logs {
		logs[i].User.Password = ""
	}

	c.JSON(http.StatusOK, logs)
}

// 5. Approve/Reject Action
func ApproveRejectTimesheet(c *gin.Context) {
	id := c.Param("id")
	approver := c.MustGet("user").(*models.User)

	var req ApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var timesheet models.Timesheet
	if err := database.DB.First(&timesheet, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Timesheet not found"})
		return
	}

	// Validasi Reject Wajib Note
	if req.Status == models.TimesheetRejected && req.RejectionNote == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection note is required"})
		return
	}

	timesheet.Status = req.Status
	timesheet.RejectionNote = req.RejectionNote
	timesheet.ApprovedByID = &approver.ID

	if err := database.DB.Save(&timesheet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, timesheet)
}

// 5.1. Update Timesheet (Edit deskripsi atau durasi)
func UpdateTimesheet(c *gin.Context) {
	id := c.Param("id")
	user := c.MustGet("user").(*models.User)

	var req UpdateTimesheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var timesheet models.Timesheet
	if err := database.DB.Where("id = ? AND user_id = ?", id, user.ID).First(&timesheet).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Timesheet not found"})
		return
	}

	// Update Fields
	timesheet.Description = req.Description
	if req.DurationSeconds != nil {
		timesheet.DurationSeconds = *req.DurationSeconds
		timesheet.DurationMinutes = *req.DurationSeconds / 60
	}

	if err := database.DB.Save(&timesheet).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Update failed"})
		return
	}

	c.JSON(http.StatusOK, timesheet)
}

// 5.2. Delete Individual Timesheet
func DeleteTimesheet(c *gin.Context) {
	id := c.Param("id")
	user := c.MustGet("user").(*models.User)

	result := database.DB.Where("id = ? AND user_id = ?", id, user.ID).Delete(&models.Timesheet{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Timesheet not found or unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Timesheet deleted successfully"})
}

// Struct Request Baru untuk Bulk
type BulkApprovalRequest struct {
	TimesheetIDs  []uint                 `json:"timesheet_ids" binding:"required"` // Array ID
	Status        models.TimesheetStatus `json:"status" binding:"required"`
	RejectionNote string                 `json:"rejection_note"`
}

// 6. Bulk Approve/Reject (Bisa approve 1, 2, atau 100 sekaligus)
func BulkApproveTimesheets(c *gin.Context) {
	approver := c.MustGet("user").(*models.User)

	var req BulkApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi Rejection Note
	if req.Status == models.TimesheetRejected && req.RejectionNote == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection note is required for rejection"})
		return
	}

	// Update Massal Menggunakan Query SQL 'IN'
	result := database.DB.Model(&models.Timesheet{}).
		Where("id IN ?", req.TimesheetIDs). // WHERE id IN (1, 2)
		Updates(map[string]interface{}{
			"status":         req.Status,
			"rejection_note": req.RejectionNote,
			"approved_by_id": approver.ID,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bulk update"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Bulk update successful",
		"rows_affected": result.RowsAffected,
	})
}