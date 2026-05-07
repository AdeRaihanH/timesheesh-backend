package handlers

import (
	"net/http"
	"time"
	"timesheesh-backend/database"
	"timesheesh-backend/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Request Structure
type PaymentTransactionRequest struct {
	Name        string  `json:"name" binding:"required"`   // e.g. "Termin 1"
	Amount      float64 `json:"amount" binding:"required"` // Uang yang dibayar
	PaidAt      string  `json:"paid_at"`                   // YYYY-MM-DD
	Description string  `json:"description"`
}


// HELPER: HITUNG ULANG STATUS KONTRAK
func RecalculateContractStatus(tx *gorm.DB, contractID uint) error {
	// 1. Ambil Data Kontrak (Untuk tau RateAmount/Target)
	var contract models.Contract
	if err := tx.First(&contract, contractID).Error; err != nil {
		return err
	}

	// 2. Hitung Total Uang Keluar (Sum Amount di tabel Payment)
	var totalPaid float64
	if err := tx.Model(&models.ContractPayment{}).
		Where("contract_id = ?", contractID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalPaid).Error; err != nil {
		return err
	}

	// 3. Tentukan Status Baru
	target := float64(contract.RateAmount)
	var newStatus models.PaymentStatus

	if totalPaid <= 0 {
		newStatus = models.StatusPending
	} else if totalPaid >= target {
		newStatus = models.StatusPaid
	} else {
		newStatus = models.StatusPartiallyPaid
	}

	// 4. Update Field Contract
	return tx.Model(&contract).Updates(map[string]interface{}{
		"total_paid":     totalPaid,
		"payment_status": newStatus,
	}).Error
}


// ENDPOINT HANDLERS
// 1. ADD PAYMENT (Input Transaksi Realisasi)
// Endpoint: POST /api/contracts/:contractId/payment
func AddContractPayment(c *gin.Context) {
	contractId := c.Param("contractId")
	
	var req PaymentTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek Kontrak & Skema
	var contract models.Contract
	if err := database.DB.First(&contract, contractId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contract not found"})
		return
	}
	if contract.PaymentScheme != models.SchemeTermin && contract.PaymentScheme != models.SchemeBackToBack {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only available for Termin or Back-to-Back schemes"})
		return
	}

	// Parse Tanggal Bayar
	paidAt, _ := time.Parse("2006-01-02", req.PaidAt)
	if paidAt.IsZero() { paidAt = time.Now() }

	// Start Transaction
	tx := database.DB.Begin()

	// A. Create Payment Record
	payment := models.ContractPayment{
		ContractID:  contract.ID,
		Name:        req.Name,
		Amount:      req.Amount,
		PaidAt:      paidAt,
		Description: req.Description,
	}

	if err := tx.Create(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
		return
	}

	// B. Recalculate Contract Status
	if err := RecalculateContractStatus(tx, contract.ID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contract status"})
		return
	}

	tx.Commit()

	// Ambil data terbaru untuk response
	database.DB.First(&contract, contractId)
	remaining := float64(contract.RateAmount) - contract.TotalPaid

	c.JSON(http.StatusCreated, gin.H{
		"message": "Payment recorded",
		"data":    payment,
		"contract_summary": gin.H{
			"contract_target": contract.RateAmount,
			"total_paid":      contract.TotalPaid,
			"remaining":       remaining,
			"status":          contract.PaymentStatus,
		},
	})
}

// 2. EDIT PAYMENT (Revisi Transaksi)
// Endpoint: PUT /api/contract-payments/:paymentId
func EditContractPayment(c *gin.Context) {
	paymentId := c.Param("paymentId")
	
	var req PaymentTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payment models.ContractPayment
	if err := database.DB.First(&payment, paymentId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Payment record not found"})
		return
	}

	// Update Data Payment
	payment.Name = req.Name
	payment.Amount = req.Amount // Nominal Baru
	payment.Description = req.Description
	if req.PaidAt != "" {
		d, _ := time.Parse("2006-01-02", req.PaidAt)
		payment.PaidAt = d
	}

	tx := database.DB.Begin()

	if err := tx.Save(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	// Recalculate Contract Status
	if err := RecalculateContractStatus(tx, payment.ContractID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recalculate contract"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Payment updated and contract recalculated", "data": payment})
}

// 3. DELETE PAYMENT (Hapus Transaksi)
// Endpoint: DELETE /api/contract-payments/:paymentId
func DeleteContractPayment(c *gin.Context) {
	paymentId := c.Param("paymentId")
	
	var payment models.ContractPayment
	if err := database.DB.First(&payment, paymentId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	
	contractID := payment.ContractID // Simpan ID sebelum dihapus

	tx := database.DB.Begin()

	// Delete
	if err := tx.Delete(&payment).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment"})
		return
	}

	// Recalculate (TotalPaid akan berkurang)
	if err := RecalculateContractStatus(tx, contractID); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recalculate contract"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Payment deleted and contract balance updated"})
}

// 4. GET HISTORY (Lihat Transaksi)
// Endpoint: GET /api/contracts/:contractId/payments
func GetContractPayments(c *gin.Context) {
	contractId := c.Param("contractId")
	
	var payments []models.ContractPayment
	if err := database.DB.Where("contract_id = ?", contractId).Order("paid_at desc").Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, payments)
}