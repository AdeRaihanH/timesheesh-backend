package database

import (
	"log"
	"timesheesh-backend/models"
	"timesheesh-backend/utils"
)

func Seed() {
	var admin models.User
	result := DB.Where("role = ?", models.RoleAdmin).First(&admin)

	if result.Error != nil {
		log.Println("Seeding: No admin user found. Creating initial admin...")
		hashedPassword, _ := utils.HashPassword("admin123")
		admin = models.User{
			Email:    "admin@gmail.com",
			Password: hashedPassword,
			FullName: "admin",
			Role:     models.RoleAdmin,
			IsActive: true,
		}
		
		if err := DB.Create(&admin).Error; err != nil {
			log.Printf("Seeding Error: Failed to create admin: %v", err)
			return 
		}
		log.Println("Seeding: Admin user created successfully (admin@gmail.com / admin123)")
	}

	var projectCount int64
	DB.Model(&models.Project{}).Count(&projectCount)

	if projectCount == 0 {
		log.Println("Seeding: No projects found. Creating sample projects...")
		projects := []models.Project{
			{Name: "Internal Development", ClientName: "Timesheesh", Status: models.ProjectStatusActive},
			{Name: "Client Alpha", ClientName: "Alpha Corp", Status: models.ProjectStatusActive},
			{Name: "Marketing Q1", ClientName: "Marketify", Status: models.ProjectStatusCompleted},
		}

		for i := range projects {
			if err := DB.Create(&projects[i]).Error; err != nil {
				log.Printf("Seeding Error: Failed to create project %s: %v", projects[i].Name, err)
				continue
			}

			log.Printf("Seeding: Creating tasks for project %s...", projects[i].Name)
			tasks := []models.Task{
				{ProjectID: projects[i].ID, CreatedByID: admin.ID, AssignedToID: admin.ID, Title: "Backend Setup", Status: models.TaskTodo, Description: "Initialize Go server"},
				{ProjectID: projects[i].ID, CreatedByID: admin.ID, AssignedToID: admin.ID, Title: "UI Design", Status: models.TaskInProgress, Description: "Design Figma mockups"},
				{ProjectID: projects[i].ID, CreatedByID: admin.ID, AssignedToID: admin.ID, Title: "Database Migration", Status: models.TaskDone, Description: "Configure PostgreSQL schema"},
			}

			for j := range tasks {
				DB.Create(&tasks[j])
			}
		}
		log.Println("Seeding: Sample projects and tasks created successfully")
	}
}
