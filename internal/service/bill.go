package service

import (
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

type BillService struct{}

func NewBillService() *BillService {
	return &BillService{}
}

type BillStats struct {
	TotalIncome  float64            `json:"total_income"`
	TotalExpense float64            `json:"total_expense"`
	ByCategory   map[string]float64 `json:"by_category"`
}

func (s *BillService) List(userID uint64, billDate string) ([]model.Bill, error) {
	var bills []model.Bill
	q := database.DB.Where("user_id = ?", userID)
	if billDate != "" {
		q = q.Where("bill_date = ?", billDate)
	}
	err := q.Order("bill_date DESC").Find(&bills).Error
	return bills, err
}

func (s *BillService) Create(bill *model.Bill) error {
	return database.DB.Create(bill).Error
}

func (s *BillService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Bill{}).Error
}

func (s *BillService) Stats(userID uint64, yearMonth string) (*BillStats, error) {
	var bills []model.Bill
	err := database.DB.Where("user_id = ? AND bill_date LIKE ?", userID, yearMonth+"%").Find(&bills).Error
	if err != nil {
		return nil, err
	}

	stats := &BillStats{
		ByCategory: make(map[string]float64),
	}
	for _, b := range bills {
		if b.Type == "income" {
			stats.TotalIncome += b.Amount
		} else {
			stats.TotalExpense += b.Amount
			stats.ByCategory[b.Category] += b.Amount
		}
	}
	return stats, nil
}
