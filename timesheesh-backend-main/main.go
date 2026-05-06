package main

import (
	"log"
	"os"
	"strings"
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

	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		origins := strings.Split(allowedOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		config.AllowOrigins = origins
		log.Printf("CORS allowed origins: %v", config.AllowOrigins)
	} else {
		config.AllowAllOrigins = true
		log.Println("CORS allowing all origins")
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
		auth.POST("/login", handlers.Login) // Login 
		auth.POST("/register", handlers.Register) // Register (Initial setup)
	}

	// Protected routes - require authentication
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", handlers.GetProfile) // Profile
		protected.GET("/my-contracts", handlers.GetMyContracts) // Get contract user

		// blm di garap
		protected.GET("/dashboard", handlers.GetDashboard) 

		admin := protected.Group("/admin")
		admin.Use(middleware.AdminOnly())
		{

			admin.GET("/users", handlers.GetAllUsers) // Get all user
			admin.GET("/users/:id", handlers.GetUserByID) // Get Get Specific User Detail
			admin.POST("/users", handlers.CreateUser) // Create New User (Admin, PM, Finance, Employee)
			admin.PUT("/users/:id", handlers.UpdateUser) // Update User Data (Role, Active Status, etc)
			admin.DELETE("/users/:id", handlers.DeleteUser) // Hard Delete User

			// Route Contract
			admin.POST("/contracts", handlers.CreateContract)        // Buat kontrak baru
			admin.GET("/users/:id/contracts", handlers.GetUserContracts) // Lihat kontrak user
			admin.PUT("/contracts/:id", handlers.UpdateContract)     // Edit kontrak
			admin.DELETE("/contracts/:id", handlers.DeleteContract)
		}

		// Project Management routes
		// 1. GENERAL READ ROUTES (Employee/PM/Admin/Finance)
		// Tidak ada middleware khusus role di sini. 
		// Logic "siapa boleh lihat apa" ada di dalam codingan Handler masing-masing.
		protected.GET("/getproject", handlers.GetAllProjects)      // List Project
		protected.GET("/getproject/:id", handlers.GetProjectByID)  // Detail Project
		protected.GET("/project/:projectId/members", handlers.GetProjectMembers) // List Member

		// 2. ADMIN ONLY ROUTES (Strict Access)
		// Group ini untuk aksi yang sensitif: 
		// - Membuat Project (Create) 
		// - Menghapus Project (Delete) 
		// - Mengatur Tim (Assign/Remove) 
		projectAdmin := protected.Group("/project")
		projectAdmin.Use(middleware.AdminOnly())
		{
			projectAdmin.POST("/create", handlers.CreateProject)       // <--- ADMIN ONLY
			projectAdmin.DELETE("/delete/:id", handlers.DeleteProject) // <--- ADMIN ONLY
			
			// Team & Contract Management
			projectAdmin.POST("/assign", handlers.AssignMember)        // <--- ADMIN ONLY
			projectAdmin.DELETE("/member/:id", handlers.RemoveMember)  // <--- ADMIN ONLY
		}

		// 3. OPERATIONAL ROUTES (Admin OR Project Manager)
		// Group ini agar PM tetap bisa bekerja mengelola project yang sedang jalan.
		// Contoh: Update status dari 'Active' ke 'Completed', update nama, dll.
		projectOps := protected.Group("/project")
		projectOps.Use(middleware.AdminOrProjectManager())
		{
			projectOps.PUT("/update/:projectId", handlers.UpdateProject) // <--- ADMIN & PM
		}

		// CONTRACT PAYMENT
		paymentGroup := protected.Group("/contract-payments")
		paymentGroup.Use(middleware.AdminOnly())
		{
			// 1. Tambah Transaksi Pembayaran Baru
			// Akses via Contract ID
			paymentGroup.POST("/contracts/:contractId/payment", handlers.AddContractPayment)

			// 2. Lihat History Pembayaran
			paymentGroup.GET("/contracts/:contractId/payments", handlers.GetContractPayments)

			// 3. Edit & Delete Transaksi
			// Akses via Payment ID
			paymentGroup.PUT("/:paymentId", handlers.EditContractPayment)
			paymentGroup.DELETE("/:paymentId", handlers.DeleteContractPayment)
		}

		// Proxy Login
		// Wajib Admin Only
		admin.POST("/proxy-login", handlers.ProxyLogin)

		// Task Management Routes
		// Bisa diakses semua user (PM assign, Employee self-assign)
		tasks := protected.Group("/tasks")
		{
			tasks.POST("/", handlers.CreateTask)           // Buat Tugas
			tasks.PUT("/:id/status", handlers.UpdateTaskStatus) // Update Status
		}
		// Get Tasks juga butuh akses
		protected.GET("/project/:projectId/tasks", handlers.GetProjectTasks)

		// Timesheet Routes
		timesheets := protected.Group("/timesheets")
		{
			timesheets.POST("/clock-in", handlers.ClockIn)    // Mulai Kerja
			timesheets.POST("/clock-out", handlers.ClockOut)  // Selesai Kerja
			timesheets.POST("/log", handlers.CreateManualLog) // Baru: Manual Log dari Grid
			timesheets.DELETE("/log", handlers.DeleteWeekLogs) // Baru: Hapus Log (X button)
			timesheets.DELETE("/:id", handlers.DeleteTimesheet) // Baru: Hapus Log Individu
			timesheets.PUT("/:id", handlers.UpdateTimesheet) // Baru: Update Deskripsi/Data Log
			timesheets.GET("/my-logs", handlers.GetMyTimesheets) // History Saya
		}

		// Approval (PM Only)
		approvals := protected.Group("/approvals")
		approvals.Use(middleware.AdminOrProjectManager())
		{
			approvals.GET("/inbox", handlers.GetPendingTimesheets)       // Lihat Pending
			approvals.PUT("/timesheet/:id", handlers.ApproveRejectTimesheet) // Action
			approvals.PUT("/bulk-action", handlers.BulkApproveTimesheets)    // Bulk
		}

		// Resource Request (Manpower/Tools)
		resources := protected.Group("/resources")
		
		// 1. Create & Read (Bisa PM, Bisa Admin)
		resources.Use(middleware.AdminOrProjectManager()) 
		{
			resources.POST("/", handlers.CreateResourceRequest)
			resources.GET("/", handlers.GetResourceRequests)
		}

		// 2. Update Status (HANYA ADMIN yang boleh approve budget/hiring)
		// Kita buat group baru atau rute spesifik pakai middleware AdminOnly
		protected.PUT("/resources/:id/status", middleware.AdminOnly(), handlers.UpdateResourceStatus)

		// Dashboard
		dashboard := protected.Group("/dashboard")
		{
			// Dashboard Employee (Stats Gaji Sendiri)
			dashboard.GET("/my-stats", handlers.GetMyStats)

			// Dashboard PM (Monitoring Project Budget) 
			// Gunakan Middleware AdminOrProjectManager
			dashboard.GET("/project/:id/stats", middleware.AdminOrProjectManager(), handlers.GetProjectBudgetStats)
		}

		executive := protected.Group("/dashboard/executive")
		{
			executive.GET("/project/:id/pnl", handlers.GetProjectPnL)
		}
	}

	// Start server
	port := ":8000"
	log.Printf("Server starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
