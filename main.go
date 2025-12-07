package main

import (
	"log"
	"os"
	"time"

	"timesheesh-backend/database"
	"timesheesh-backend/handlers"
	"timesheesh-backend/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found. Using environment variables.")
	}

	// Initialize database
	database.InitDatabase()

	// Setup Gin router
	r := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()

	// Get allowed origins from environment variable, default to allow all
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		// Split multiple origins by comma
		config.AllowOrigins = []string{allowedOrigins}
	} else {
		// Allow all origins by default (for development)
		config.AllowAllOrigins = true
	}

	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	r.Use(cors.New(config))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes - Authentication
	auth := r.Group("/api/auth")
	{
		auth.POST("/login", handlers.Login)
		auth.POST("/register", handlers.Register)
	}

	// Protected routes - require authentication
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", handlers.GetProfile)
		protected.GET("/dashboard", handlers.GetDashboard)

		admin := protected.Group("/admin")
		admin.Use(middleware.AdminOnly())
		{
			admin.POST("/register", handlers.Register)

			admin.GET("/users", handlers.GetAllUsers)
			admin.GET("/users/:id", handlers.GetUserByID)
			admin.POST("/users", handlers.CreateUser)
			admin.PUT("/users/:id", handlers.UpdateUser)
			admin.DELETE("/users/:id", handlers.DeleteUser)
		}

		// Project Management routes
		// Admin and Project Manager can create, update, delete
		// All authenticated users can view projects
		project := protected.Group("/project")
		project.Use(middleware.AdminOrProjectManager())
		{
			project.POST("/create", handlers.CreateProject)
			project.PUT("/update/:projectId", handlers.UpdateProject)
			project.DELETE("/delete/:id", handlers.DeleteProject)
		}

		// Get projects - accessible to all authenticated users
		protected.GET("/getproject", handlers.GetAllProjects)
		protected.GET("/getproject/:id", handlers.GetProjectByID)
	}

	// Start server
	port := ":8000"
	log.Printf("Server starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
