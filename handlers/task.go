package handlers

import (
	"net/http"
	"timesheesh-backend/database"
	"timesheesh-backend/models"
	"github.com/gin-gonic/gin"
)

type CreateTaskRequest struct {
	ProjectID    uint        `json:"project_id" binding:"required"`
	AssignedToID *uint       `json:"assigned_to_id"` // Sekarang opsional jika ada Role
	Role         models.Role `json:"role"`           // Opsional
	Title        string      `json:"title" binding:"required"`
	Description  string      `json:"description"`
}

type UpdateTaskStatusRequest struct {
	Status models.TaskStatus `json:"status" binding:"required"`
}

// 1. Create Task (Bisa PM assign orang lain, atau User assign diri sendiri)
func CreateTask(c *gin.Context) {
	// Ambil ID User yang sedang login dari Token
	currentUser := c.MustGet("user").(*models.User)

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Logic Bulk Assign by Role
	if req.Role != "" {
		// Cari semua member project tersebut yang punya Role ini
		var memberUsers []models.User
		err := database.DB.Table("users").
			Select("users.*").
			Joins("JOIN project_members ON project_members.user_id = users.id").
			Where("project_members.project_id = ? AND users.role = ?", req.ProjectID, req.Role).
			Find(&memberUsers).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project members by role"})
			return
		}

		if len(memberUsers) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No project members found with the specified role"})
			return
		}

		// Create multiple tasks
		var tasks []models.Task
		for _, u := range memberUsers {
			tasks = append(tasks, models.Task{
				ProjectID:    req.ProjectID,
				CreatedByID:  currentUser.ID,
				AssignedToID: u.ID,
				Title:        req.Title,
				Description:  req.Description,
				Status:       models.TaskTodo,
			})
		}

		if err := database.DB.Create(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bulk tasks"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Tasks created for role", "count": len(tasks)})
		return
	}

	// 2. Logic Single Assign by ID (Existing)
	if req.AssignedToID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assigned_to_id or role is required"})
		return
	}

	task := models.Task{
		ProjectID:    req.ProjectID,
		CreatedByID:  currentUser.ID,
		AssignedToID: *req.AssignedToID,
		Title:        req.Title,
		Description:  req.Description,
		Status:       models.TaskTodo,
	}

	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// 2. Get Tasks (Filter by Project & My Tasks)
func GetProjectTasks(c *gin.Context) {
	projectId := c.Param("projectId")
	currentUser := c.MustGet("user").(*models.User)
	
	// Query param ?my_tasks=true
	onlyMyTasks := c.Query("my_tasks") == "true"

	var tasks []models.Task
	query := database.DB.Where("project_id = ?", projectId)

	if onlyMyTasks {
		query = query.Where("assigned_to_id = ?", currentUser.ID)
	}

	if err := query.Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// 3. Update Task Status (Manual update, misal Done)
func UpdateTaskStatus(c *gin.Context) {
	id := c.Param("id")
	var req UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task models.Task
	if err := database.DB.First(&task, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	task.Status = req.Status
	database.DB.Save(&task)

	c.JSON(http.StatusOK, task)
}