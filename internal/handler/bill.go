package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

type BillHandler struct {
	svc *service.BillService
}

func NewBillHandler(svc *service.BillService) *BillHandler {
	return &BillHandler{svc: svc}
}

func (h *BillHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	billDate := c.Query("date")
	bills, err := h.svc.List(userID, billDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bills == nil {
		bills = []model.Bill{}
	}
	c.JSON(http.StatusOK, bills)
}

func (h *BillHandler) Create(c *gin.Context) {
	var bill model.Bill
	if err := c.ShouldBindJSON(&bill); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bill.UserID = c.GetUint64("user_id")
	if err := h.svc.Create(&bill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bill)
}

func (h *BillHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *BillHandler) Stats(c *gin.Context) {
	userID := c.GetUint64("user_id")
	yearMonth := c.DefaultQuery("month", "")
	if yearMonth == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month parameter required (format: YYYY-MM)"})
		return
	}
	stats, err := h.svc.Stats(userID, yearMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
