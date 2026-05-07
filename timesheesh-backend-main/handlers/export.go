package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
)

// ExportTasksCSV 
func ExportTasksCSV(c *gin.Context) {
	projectId := c.Param("projectId")
	var tasks []models.Task

	// Retrieve data from database based on Project ID
	if err := database.DB.Where("project_id = ?", projectId).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data untuk export"})
		return
	}

	// Setting the header so that the browser recognizes this as a download file
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=tasks-project-%s.csv", projectId))
	c.Header("Content-Type", "text/csv")

	// Initialize CSV writer to create simple Excel format
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header column
	writer.Write([]string{"ID", "Judul Task", "Status", "Assigned To ID"})

	// Write data contens
	for _, task := range tasks {
		row := []string{
			fmt.Sprintf("%d", task.ID),
			task.Title,
			string(task.Status),
			fmt.Sprintf("%d", task.AssignedToID),
		}
		writer.Write(row)
	}
}