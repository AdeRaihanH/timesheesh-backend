package handlers

import (
	"net/http"
	"time"
	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
)

type DashboardResponse struct {
	Period           string  `json:"period"`
	ActiveContracts  int     `json:"active_contracts"`
	
	Earnings struct {
		FixedSalary     float64 `json:"fixed_salary"`     // Gaji Pasti (Bulanan)
		VariableIncome  float64 `json:"variable_income"`  // Gaji Estimasi (Jam-jaman)
		TotalEstimation float64 `json:"total_estimation"`
	} `json:"earnings"`

	ProjectBreakdown []ProjectEarning `json:"project_breakdown"`
	RecentHistory    []models.Timesheet `json:"recent_history"`
}

type ProjectEarning struct {
	ProjectName string  `json:"project_name"`
	HoursWorked float64 `json:"hours_worked"`
	RateUsed    int64   `json:"rate_used"` // Rate mana yang dipakai (Custom/Base)
	Subtotal    float64 `json:"subtotal"`
}

// GET /api/dashboard/my-stats
func GetMyStats(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	now := time.Now()
	
	// 1. Tentukan Range Awal & Akhir Bulan Ini
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	response := DashboardResponse{
		Period: startOfMonth.Format("January 2006"),
	}

	// ==========================================
	// STEP 1: LOAD CONTRACTS (Fixed Salary)
	// ==========================================
	var contracts []models.Contract
	if err := database.DB.Where("user_id = ? AND is_active = ?", user.ID, true).Find(&contracts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch contracts"})
		return
	}
	response.ActiveContracts = len(contracts)

	var baseHourlyRate int64 = 0

	for _, contract := range contracts {
		if contract.ContractType == models.ContractTypeMonthly {
			// Jika kontrak bulanan, langsung tambahkan ke Fixed Salary
			response.Earnings.FixedSalary += float64(contract.RateAmount)
		} else if contract.ContractType == models.ContractTypeTimesheet {
			// Simpan base rate hourly (jika punya multiple, ambil yang terakhir/terbesar logicnya)
			if contract.RateAmount > baseHourlyRate {
				baseHourlyRate = contract.RateAmount
			}
		}
	}

	// ==========================================
	// STEP 2: PREPARE CUSTOM RATES (Project Member)
	// ==========================================
	// Kita butuh tahu apakah user punya tarif khusus di project tertentu
	// Map: ProjectID -> CustomRate
	customRates := make(map[uint]int64)
	var members []models.ProjectMember
	database.DB.Where("user_id = ?", user.ID).Find(&members)

	for _, m := range members {
		if m.CustomRate != nil {
			customRates[m.ProjectID] = *m.CustomRate
		}
	}

	// ==========================================
	// STEP 3: CALCULATE TIMESHEETS (Variable Income)
	// ==========================================
	var timesheets []models.Timesheet
	// Ambil Timesheet APPROVED bulan ini
	database.DB.Preload("Project").
		Where("user_id = ? AND status = ? AND clock_in BETWEEN ? AND ?", 
		user.ID, models.TimesheetApproved, startOfMonth, endOfMonth).
		Find(&timesheets)

	// Map untuk breakdown per project
	projectStats := make(map[uint]*ProjectEarning)

	for _, ts := range timesheets {
		durationHours := float64(ts.DurationMinutes) / 60.0
		
		// LOGIC WATERFALL RATE:
		// 1. Cek apakah ada Custom Rate di project ini?
		// 2. Jika tidak, pakai Base Hourly Rate dari kontrak.
		var rateToUse int64 = baseHourlyRate
		if val, ok := customRates[ts.ProjectID]; ok {
			rateToUse = val
		}

		// Hitung Duit: Jam x Rate
		earnings := durationHours * float64(rateToUse)
		response.Earnings.VariableIncome += earnings

		// Masukkan ke Breakdown Project
		if _, exists := projectStats[ts.ProjectID]; !exists {
			projectStats[ts.ProjectID] = &ProjectEarning{
				ProjectName: ts.Project.Name,
				RateUsed:    rateToUse,
				HoursWorked: 0,
				Subtotal:    0,
			}
		}
		projectStats[ts.ProjectID].HoursWorked += durationHours
		projectStats[ts.ProjectID].Subtotal += earnings
	}

	// Convert Map breakdown ke Slice Array
	for _, stat := range projectStats {
		response.ProjectBreakdown = append(response.ProjectBreakdown, *stat)
	}

	// Total Akhir
	response.Earnings.TotalEstimation = response.Earnings.FixedSalary + response.Earnings.VariableIncome

	// ==========================================
	// STEP 4: RECENT HISTORY (5 Terakhir)
	// ==========================================
	var history []models.Timesheet
	database.DB.Preload("Task").Preload("Project").
		Where("user_id = ?", user.ID).
		Order("clock_in desc").
		Limit(5).
		Find(&history)
	
	response.RecentHistory = history

	c.JSON(http.StatusOK, response)
}

// Struct Dashboard PM
type PMProjectDashboardResponse struct {
	ProjectName string `json:"project_name"`
	ClientName  string `json:"client_name"`

	// 1. Time Health (Mandays/Jam)
	TimeBudget struct {
		TotalAllocated int64   `json:"total_allocated"` // BudgetedHours
		Used           float64 `json:"used"`            // Dari Timesheet (Jam)
		Remaining      float64 `json:"remaining"`
		BurnPercentage float64 `json:"burn_percentage"`
		Status         string  `json:"status"` // safe / warning / critical
	} `json:"time_budget"`

	// 2. Financial Health (Uang)
	FinancialBudget struct {
		TotalBudget    float64 `json:"total_budget"` // BudgetCost
		LaborCost      float64 `json:"labor_cost"`   // Gaji Tim
		ExpenseCost    float64 `json:"expense_cost"` // Server/Tools
		TotalBurned    float64 `json:"total_burned"` // Labor + Expense
		Remaining      float64 `json:"remaining"`
		BurnPercentage float64 `json:"burn_percentage"`
		Status         string  `json:"status"`
	} `json:"financial_budget"`

	Alerts []string `json:"alerts"`
}

// Dashboard For PM (Budget Monitoring)
// GET /api/dashboard/project/:id/stats
func GetProjectBudgetStats(c *gin.Context) {
	projectID := c.Param("id")

	var project models.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	response := PMProjectDashboardResponse{
		ProjectName: project.Name,
		ClientName:  project.ClientName,
	}

	//  CALCULATE TIME BURN (Mandays)
	// Strategy: Hitung semua jam kerja (Pending + Approved) agar monitoring Real-time.
	var totalMinutes int64
	database.DB.Model(&models.Timesheet{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(sum(duration_minutes), 0)").
		Scan(&totalMinutes)

	hoursUsed := float64(totalMinutes) / 60.0

	// Handle Null Pointers for Project Budget
	var budgetedHours int64 = 0
	if project.BudgetedHours != nil {
		budgetedHours = *project.BudgetedHours
	}

	response.TimeBudget.TotalAllocated = budgetedHours
	response.TimeBudget.Used = hoursUsed
	response.TimeBudget.Remaining = float64(budgetedHours) - hoursUsed

	// Status Logic & Alert Waktu
	if budgetedHours > 0 {
		response.TimeBudget.BurnPercentage = (hoursUsed / float64(budgetedHours)) * 100

		var hourThreshold int64 = 0
		if project.HourThreshold != nil {
			hourThreshold = *project.HourThreshold
		}

		if hoursUsed >= float64(budgetedHours) {
			response.TimeBudget.Status = "critical"
			response.Alerts = append(response.Alerts, "CRITICAL: Time budget exceeded!")
		} else if hourThreshold > 0 && hoursUsed >= float64(hourThreshold) {
			response.TimeBudget.Status = "warning"
			response.Alerts = append(response.Alerts, "WARNING: Time budget is running low.")
		} else {
			response.TimeBudget.Status = "safe"
		}
	} else {
		response.TimeBudget.Status = "safe" // No budget set
	}

	// CALCULATE FINANCIAL BURN (Labor + Expenses)
	// A. PREPARE DATA (Agar tidak query N+1 dalam loop)
	// Ambil semua member project untuk cek Custom Rate
	customRates := make(map[uint]int64)
	var members []models.ProjectMember
	database.DB.Where("project_id = ?", projectID).Find(&members)
	var userIDs []uint
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		if m.CustomRate != nil {
			customRates[m.UserID] = *m.CustomRate
		}
	}

	// Ambil semua Contract aktif milik user-user tersebut untuk Base Rate
	// Map: UserID -> Rate Per Jam (Float karena hasil bagi 173)
	baseRates := make(map[uint]float64)
	var contracts []models.Contract
	if len(userIDs) > 0 {
		database.DB.Where("user_id IN ? AND is_active = ?", userIDs, true).Find(&contracts)
	}

	for _, contract := range contracts {
		// Logic: Jika user punya multiple contract, yang terakhir di-load yang dipakai
		if contract.ContractType == models.ContractTypeMonthly {
			// Standar Internasional: Gaji Sebulan / 173 Jam
			baseRates[contract.UserID] = float64(contract.RateAmount) / 173.0
		} else {
			baseRates[contract.UserID] = float64(contract.RateAmount)
		}
	}

	// B. HITUNG LABOR COST (Gaji Tim)
	// Hanya hitung Timesheet APPROVED (Uang keluar valid)
	var timesheets []models.Timesheet
	database.DB.Where("project_id = ? AND status = ?", projectID, models.TimesheetApproved).Find(&timesheets)

	var totalLaborCost float64 = 0
	for _, ts := range timesheets {
		var rate float64 = 0

		// 1. Cek Custom Rate Project dulu
		if val, ok := customRates[ts.UserID]; ok {
			rate = float64(val)
		} else {
			// 2. Cek Base Rate Contract
			if val, ok := baseRates[ts.UserID]; ok {
				rate = val
			}
		}

		hours := float64(ts.DurationMinutes) / 60.0
		totalLaborCost += hours * rate
	}
	response.FinancialBudget.LaborCost = totalLaborCost

	// C. HITUNG EXPENSE COST (Tools/Server)
	// Ambil dari ResourceRequest type='tool' dan status='approved'
	var totalExpenseCost float64
	database.DB.Model(&models.ResourceRequest{}).
		Where("project_id = ? AND type = ? AND status = ?", projectID, "tool", "approved").
		Select("COALESCE(sum(amount), 0)").
		Scan(&totalExpenseCost)

	response.FinancialBudget.ExpenseCost = totalExpenseCost

	// D. TOTAL FINANCIAL STATUS
	totalBurn := totalLaborCost + totalExpenseCost
	response.FinancialBudget.TotalBurned = totalBurn

	var budgetCost float64 = 0
	if project.BudgetCost != nil {
		budgetCost = *project.BudgetCost
	}
	response.FinancialBudget.TotalBudget = float64(budgetCost)
	response.FinancialBudget.Remaining = float64(budgetCost) - totalBurn

	if budgetCost > 0 {
		response.FinancialBudget.BurnPercentage = (totalBurn / float64(budgetCost)) * 100

		var costThreshold float64 = 0
		if project.BudgetCostThreshold != nil {
			costThreshold = *project.BudgetCostThreshold
		}

		if totalBurn >= float64(budgetCost) {
			response.FinancialBudget.Status = "critical"
			response.Alerts = append(response.Alerts, "CRITICAL: Financial budget exceeded!")
		} else if costThreshold > 0 && totalBurn >= float64(costThreshold) {
			response.FinancialBudget.Status = "warning"
			response.Alerts = append(response.Alerts, "WARNING: Financial budget is running low.")
		} else {
			response.FinancialBudget.Status = "safe"
		}
	} else {
		response.FinancialBudget.Status = "safe"
	}

	c.JSON(http.StatusOK, response)
}

// Struct Response Dashboard Executive
type PnLResponse struct {
	ProjectName string `json:"project_name"`
	Currency    string `json:"currency"`

	// Summary Utama (Kartu Atas)
	Statement struct {
		Revenue          float64 `json:"revenue"`           // BudgetRevenue
		TotalCost        float64 `json:"total_cost"`        // Labor + Expense
		NetProfit        float64 `json:"net_profit"`        // Revenue - Cost
		MarginPercentage float64 `json:"margin_percentage"` // (Profit/Revenue)*100
		Status           string  `json:"status"`            // healthy/warning/danger
	} `json:"pnl_statement"`

	// Breakdown untuk Pie Chart (4 Slices)
	CostBreakdown []BreakdownItem `json:"cost_breakdown"`
}

type BreakdownItem struct {
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
	ColorCode  string  `json:"color_code,omitempty"` // Opsional: Helper untuk Frontend
}

// GET /api/dashboard/executive/project/:id/pnl
func GetProjectPnL(c *gin.Context) {
	// 1. CEK ROLE (Manual Check selain Middleware)
	// Pastikan hanya Admin atau Finance
	user := c.MustGet("user").(*models.User)
	if user.Role != "admin" && user.Role != "finance" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied. Executive level only."})
		return
	}

	projectID := c.Param("id")
	var project models.Project
	if err := database.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// ==========================================
	// 1. HITUNG LABOR COST (Timesheet x Rates)
	// ==========================================
	
	// A. Siapkan Data Rate (Caching Strategy)
	// Ambil semua member & contract untuk menghindari query berulang
	customRates := make(map[uint]int64)
	baseRates := make(map[uint]float64)
	
	var members []models.ProjectMember
	database.DB.Where("project_id = ?", projectID).Find(&members)
	var userIDs []uint
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
		if m.CustomRate != nil {
			customRates[m.UserID] = *m.CustomRate
		}
	}

	if len(userIDs) > 0 {
		var contracts []models.Contract
		database.DB.Where("user_id IN ? AND is_active = ?", userIDs, true).Find(&contracts)
		for _, ct := range contracts {
			if ct.ContractType == "monthly" {
				baseRates[ct.UserID] = float64(ct.RateAmount) / 173.0 // Rumus: Gaji / 173 Jam
			} else {
				baseRates[ct.UserID] = float64(ct.RateAmount)
			}
		}
	}

	// B. Hitung Total Gaji (Hanya Timesheet Approved)
	var timesheets []models.Timesheet
	database.DB.Where("project_id = ? AND status = ?", projectID, models.TimesheetApproved).Find(&timesheets)

	var totalLaborCost float64 = 0
	for _, ts := range timesheets {
		var rate float64 = 0
		// Priority: Custom Rate > Base Contract
		if val, ok := customRates[ts.UserID]; ok {
			rate = float64(val)
		} else if val, ok := baseRates[ts.UserID]; ok {
			rate = val
		}
		
		hours := float64(ts.DurationMinutes) / 60.0
		totalLaborCost += hours * rate
	}

	// ==========================================
	// 2. HITUNG EXPENSE COST (Resource Requests)
	// ==========================================
	// Kita akan group by Type
	type ExpenseResult struct {
		Type   string
		Total  float64
	}
	var expenses []ExpenseResult
	
	// Query: Select type, sum(amount) from resource_requests ... group by type
	database.DB.Model(&models.ResourceRequest{}).
		Select("type, sum(amount) as total").
		Where("project_id = ? AND status = ?", projectID, "approved").
		Group("type").
		Scan(&expenses)

	// Mapping hasil query ke variabel terpisah
	var costTool, costInfra, costAccom, costManpowerReq float64
	for _, exp := range expenses {
		switch exp.Type {
		case string(models.ResTypeTool):
			costTool = exp.Total
		case string(models.ResTypeInfra):
			costInfra = exp.Total
		case string(models.ResTypeAccom):
			costAccom = exp.Total
		case string(models.ResTypeManpower):
			costManpowerReq = exp.Total
		}
	}

	// Note: costManpowerReq (misal fee agency) kita gabung ke Labor Cost agar Pie Chart rapi
	// atau bisa dibiarkan terpisah. Di sini saya gabungkan ke Labor Cost agar sesuai request "Cost SDM".
	totalLaborCost += costManpowerReq

	// ==========================================
	// 3. SUSUN DATA P&L & RESPONSE
	// ==========================================
	response := PnLResponse{
		ProjectName: project.Name,
		Currency:    "IDR",
	}

	// Hitung Totals
	totalExpense := costTool + costInfra + costAccom
	totalAllCost := totalLaborCost + totalExpense

	var revenue float64 = 0
	if project.BudgetRevenue != nil {
		revenue = *project.BudgetRevenue
	}

	netProfit := revenue - totalAllCost
	
	// Isi PnL Statement
	response.Statement.Revenue = revenue
	response.Statement.TotalCost = totalAllCost
	response.Statement.NetProfit = netProfit
	
	if revenue > 0 {
		response.Statement.MarginPercentage = (netProfit / revenue) * 100
	}

	// Tentukan Status Kesehatan Project
	margin := response.Statement.MarginPercentage
	if margin >= 30 {
		response.Statement.Status = "healthy"
	} else if margin >= 10 {
		response.Statement.Status = "warning"
	} else {
		response.Statement.Status = "danger"
	}

	// Isi Breakdown untuk Pie Chart (4 Kategori Utama)
	// Helper function untuk hitung %
	calcPercent := func(amount, total float64) float64 {
		if total == 0 { return 0 }
		return (amount / total) * 100
	}

	response.CostBreakdown = []BreakdownItem{
		{
			Category:   "Labor Cost (SDM)",
			Amount:     totalLaborCost,
			Percentage: calcPercent(totalLaborCost, totalAllCost),
			ColorCode:  "#3B82F6", // Biru
		},
		{
			Category:   "Infrastructure",
			Amount:     costInfra,
			Percentage: calcPercent(costInfra, totalAllCost),
			ColorCode:  "#F59E0B", // Orange
		},
		{
			Category:   "Tools & Assets",
			Amount:     costTool,
			Percentage: calcPercent(costTool, totalAllCost),
			ColorCode:  "#10B981", // Hijau
		},
		{
			Category:   "Accommodation",
			Amount:     costAccom,
			Percentage: calcPercent(costAccom, totalAllCost),
			ColorCode:  "#8B5CF6", // Ungu
		},
	}

	c.JSON(http.StatusOK, response)
}