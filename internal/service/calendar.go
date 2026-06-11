package service

import (
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

func (s *CalendarService) List(userID uint64) ([]model.Event, error) {
	var events []model.Event
	err := database.DB.Where("user_id = ?", userID).Order("start_time ASC").Find(&events).Error
	return events, err
}

func (s *CalendarService) Create(event *model.Event) error {
	return database.DB.Create(event).Error
}

func (s *CalendarService) Update(id uint64, userID uint64, updates map[string]any) error {
	return database.DB.Model(&model.Event{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

func (s *CalendarService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Event{}).Error
}
