package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

// BillHandler 记账相关的 HTTP 处理器。
type BillHandler struct {
	svc *service.BillService
}

// NewBillHandler 创建记账处理器实例。
func NewBillHandler(svc *service.BillService) *BillHandler {
	return &BillHandler{svc: svc}
}

// List 获取当前用户的账单列表。
//
//	GET /api/bills?date=2025-06-11   → 只查询指定日期的账单
//	GET /api/bills                    → 查询所有账单
func (h *BillHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	// c.Query("date") 获取 URL 查询参数（?date=xxx 部分）
	billDate := c.Query("date")
	bills, err := h.svc.List(userID, billDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 确保返回空数组而非 null
	if bills == nil {
		bills = []model.Bill{}
	}
	c.JSON(http.StatusOK, bills)
}

// Create 创建一条新账单。
//
//	POST /api/bills
//	请求体：{"amount": 35.5, "type": "expense", "category": "餐饮", "bill_date": "2025-06-11", "note": "午餐"}
func (h *BillHandler) Create(c *gin.Context) {
	var bill model.Bill
	if err := c.ShouldBindJSON(&bill); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 覆盖 UserID，确保只创建自己的账单
	bill.UserID = c.GetUint64("user_id")
	if err := h.svc.Create(&bill); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bill)
}

// Delete 删除指定账单。
//
//	DELETE /api/bills/:id
func (h *BillHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Stats 获取指定月份的账单统计。
//
//	GET /api/bills/stats?month=2025-06
//
// 返回数据示例：
//
//	{
//	  "total_income": 15000.00,
//	  "total_expense": 8500.50,
//	  "by_category": {
//	    "餐饮": 2500.00,
//	    "交通": 500.00,
//	    "购物": 3000.00
//	  }
//	}
func (h *BillHandler) Stats(c *gin.Context) {
	userID := c.GetUint64("user_id")
	// DefaultQuery：获取查询参数，不存在则返回默认值
	// 这里没有默认值——month 参数是必填的
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
