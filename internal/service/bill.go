package service

import (
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

// BillService 记账服务，负责账单的增删查和统计分析。
type BillService struct{}

// NewBillService 创建记账服务实例。
func NewBillService() *BillService {
	return &BillService{}
}

// BillStats 账单统计结果。
// 前端通常用这个数据来渲染饼图/柱状图等可视化图表。
type BillStats struct {
	TotalIncome  float64            `json:"total_income"`  // 总收入
	TotalExpense float64            `json:"total_expense"` // 总支出
	ByCategory   map[string]float64 `json:"by_category"`   // 按分类统计支出：{"餐饮": 520.5, "交通": 200.0}
}

// List 获取指定用户的账单列表。
//
// 支持按日期筛选：
//   - billDate 为空时：返回该用户的所有账单
//   - billDate 不为空时：只返回该日期的账单（如 "2025-06-11"）
//
// 参数：
//   - userID：用户 ID
//   - billDate：账单日期筛选（可选），格式 YYYY-MM-DD
//
// 返回：账单列表 和 可能的错误
func (s *BillService) List(userID uint64, billDate string) ([]model.Bill, error) {
	var bills []model.Bill
	// 先构建基础查询条件
	q := database.DB.Where("user_id = ?", userID)
	// 如果传了日期参数，追加日期过滤条件
	if billDate != "" {
		q = q.Where("bill_date = ?", billDate)
	}
	// 按账单日期降序排列，最新的在最前面
	err := q.Order("bill_date DESC").Find(&bills).Error
	return bills, err
}

// Create 创建一条新的账单记录。
//
// 参数：
//   - bill：账单数据指针
//
// 返回：可能的错误
func (s *BillService) Create(bill *model.Bill) error {
	return database.DB.Create(bill).Error
}

// Delete 删除指定账单。
//
// 安全性：同时检查 id 和 user_id
//
// 参数：
//   - id：账单 ID
//   - userID：用户 ID
//
// 返回：可能的错误
func (s *BillService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Bill{}).Error
}

// Stats 统计指定月份的账单数据。
//
// 通过 SQL 的 LIKE 操作符实现月份匹配：
//
//	yearMonth = "2025-06"  →  WHERE bill_date LIKE "2025-06%"
//	这会匹配所有 "2025-06-01"、"2025-06-15"、"2025-06-30" 等日期
//
// 统计过程：
//  1. 查询该月的所有账单
//  2. 遍历每条账单，累加收入和支出
//  3. 对支出按分类汇总（收入不按分类统计）
//
// 参数：
//   - userID：用户 ID
//   - yearMonth：年月，格式 YYYY-MM（如 "2025-06"）
//
// 返回：统计结果 和 可能的错误
func (s *BillService) Stats(userID uint64, yearMonth string) (*BillStats, error) {
	var bills []model.Bill
	// LIKE 查询：bill_date LIKE "2025-06%" → 匹配该月所有日期
	err := database.DB.Where("user_id = ? AND bill_date LIKE ?", userID, yearMonth+"%").Find(&bills).Error
	if err != nil {
		return nil, err
	}

	// 初始化统计结构体，map 需要用 make 创建，否则是 nil 无法写入
	stats := &BillStats{
		ByCategory: make(map[string]float64),
	}
	// 遍历账单，累加统计
	for _, b := range bills {
		if b.Type == "income" {
			// 收入：累加到总收入
			stats.TotalIncome += b.Amount
		} else {
			// 支出：累加到总支出，同时按分类汇总
			stats.TotalExpense += b.Amount
			stats.ByCategory[b.Category] += b.Amount
		}
	}
	return stats, nil
}
