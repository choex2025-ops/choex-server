package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

type MemoryService struct{}

func NewMemoryService() *MemoryService {
	return &MemoryService{}
}

func (s *MemoryService) List(userID uint64) ([]model.AgentMemory, error) {
	var memories []model.AgentMemory
	err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&memories).Error
	return memories, err
}

func (s *MemoryService) Create(m *model.AgentMemory) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		// Create default current version
		v := model.MemoryVersion{
			MemoryID:    m.ID,
			VersionType: "current",
			Content:     "",
		}
		return tx.Create(&v).Error
	})
}

func (s *MemoryService) Activate(id uint64, userID uint64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Deactivate all
		if err := tx.Model(&model.AgentMemory{}).Where("user_id = ?", userID).Update("is_active", false).Error; err != nil {
			return err
		}
		// Activate target
		return tx.Model(&model.AgentMemory{}).Where("id = ? AND user_id = ?", id, userID).Update("is_active", true).Error
	})
}

func (s *MemoryService) Delete(id uint64, userID uint64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("memory_id = ?", id).Delete(&model.MemoryVersion{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.AgentMemory{}).Error
	})
}

func (s *MemoryService) GetVersions(memoryID uint64) (map[string]string, error) {
	var versions []model.MemoryVersion
	if err := database.DB.Where("memory_id = ?", memoryID).Find(&versions).Error; err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, v := range versions {
		result[v.VersionType] = v.Content
	}
	return result, nil
}

func (s *MemoryService) SaveVersion(memoryID uint64, versionType string, content string) error {
	if versionType == "current" {
		var backup model.MemoryVersion
		if err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, "current").First(&backup).Error; err == nil {
			database.DB.Model(&model.MemoryVersion{}).Where("memory_id = ? AND version_type = ?", memoryID, "backup").Delete(&model.MemoryVersion{})
			database.DB.Create(&model.MemoryVersion{
				MemoryID:    memoryID,
				VersionType: "backup",
				Content:     backup.Content,
			})
		}
	}

	var existing model.MemoryVersion
	err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, versionType).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return database.DB.Create(&model.MemoryVersion{
			MemoryID:    memoryID,
			VersionType: versionType,
			Content:     content,
		}).Error
	}
	return database.DB.Model(&existing).Update("content", content).Error
}

func (s *MemoryService) Restore(memoryID uint64) error {
	var backup model.MemoryVersion
	if err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, "backup").First(&backup).Error; err != nil {
		return err
	}
	return s.SaveVersion(memoryID, "current", backup.Content)
}
