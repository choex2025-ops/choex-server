package service

import (
	"errors"

	"gorm.io/gorm"

	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

// MemoryService 智能体记忆服务，负责记忆和版本的增删改查。
//
// 核心设计：
//   1. 每个用户可以创建多个记忆（AgentMemory）
//   2. 同一时间只有一个记忆处于激活状态（IsActive = true）
//   3. 每个记忆可以有多个版本（MemoryVersion）：current、backup、custom
//   4. 修改 current 版本时，旧内容自动备份为 backup 版本
//   5. backup 版本只有一个（新的会覆盖旧的）
//
// 记忆版本的生命周期示例：
//
//	用户创建记忆 "理财助手" → current 版本为空
//	用户保存 prompt → current = "你是理财顾问..."，无 backup
//	用户修改 prompt → 旧内容移到 backup = "你是理财顾问..."
//	                  current = "你是专业的理财顾问，擅长..."
//	用户点击恢复   → current 被 backup 的内容覆盖
//	                  current = "你是理财顾问..."（恢复了）
//
// 为什么要用数据库事务（Transaction）？
//
//	以激活记忆为例：
//	  Step 1: 把所有记忆设为 IsActive = false
//	  Step 2: 把目标记忆设为 IsActive = true
//	如果不加事务，Step 1 执行成功但 Step 2 失败，结果所有记忆都是未激活状态。
//	加了事务后：要么两步都成功，要么都回滚（Rollback），不会出现中间状态。
//	这就是事务的 ACID 特性中的 A（Atomicity，原子性）。
type MemoryService struct{}

// NewMemoryService 创建记忆服务实例。
func NewMemoryService() *MemoryService {
	return &MemoryService{}
}

// List 获取指定用户的所有记忆列表。
//
// 参数：
//   - userID：用户 ID
//
// 返回：记忆列表 和 可能的错误
func (s *MemoryService) List(userID uint64) ([]model.AgentMemory, error) {
	var memories []model.AgentMemory
	err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&memories).Error
	return memories, err
}

// Create 创建一条新记忆，同时创建默认的 current 版本。
//
// 使用事务确保：
//   记忆记录创建成功 → 默认版本记录也创建成功
//   任一步失败 → 全部回滚
//
// 参数：
//   - m：记忆数据指针（UserID 应该在调用前设置好）
//
// 返回：可能的错误
func (s *MemoryService) Create(m *model.AgentMemory) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: 创建记忆记录
		if err := tx.Create(m).Error; err != nil {
			return err // 返回错误，事务自动回滚
		}
		// Step 2: 创建默认的 current 版本（内容为空）
		v := model.MemoryVersion{
			MemoryID:    m.ID,          // 关联刚创建的记忆
			VersionType: "current",     // 默认版本类型
			Content:     "",            // 空内容，等待用户编辑
		}
		return tx.Create(&v).Error
	})
}

// Activate 激活指定记忆，同时停用该用户的其他所有记忆。
//
// 使用事务确保激活操作的原子性。
//
// 参数：
//   - id：要激活的记忆 ID
//   - userID：用户 ID
//
// 返回：可能的错误
func (s *MemoryService) Activate(id uint64, userID uint64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: 将该用户的所有记忆设为未激活
		// Update + Where 表示批量更新符合条件的记录
		if err := tx.Model(&model.AgentMemory{}).Where("user_id = ?", userID).Update("is_active", false).Error; err != nil {
			return err
		}
		// Step 2: 激活目标记忆
		// 同时检查 id 和 user_id，确保只激活自己的记忆
		return tx.Model(&model.AgentMemory{}).Where("id = ? AND user_id = ?", id, userID).Update("is_active", true).Error
	})
}

// Delete 删除记忆及其所有版本。
//
// 使用事务确保：删除记忆记录 → 删除关联的版本记录，要么都成功要么都失败。
//
// 参数：
//   - id：要删除的记忆 ID
//   - userID：用户 ID
//
// 返回：可能的错误
func (s *MemoryService) Delete(id uint64, userID uint64) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Step 1: 先删除该记忆的所有版本（外键级联的手动实现）
		if err := tx.Where("memory_id = ?", id).Delete(&model.MemoryVersion{}).Error; err != nil {
			return err
		}
		// Step 2: 删除记忆记录本身
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.AgentMemory{}).Error
	})
}

// GetVersions 获取指定记忆的所有版本。
//
// 返回一个 map，key 是版本类型（"current"/"backup"/"custom"），
// value 是该版本的 content。
//
// 参数：
//   - memoryID：记忆 ID
//
// 返回：版本 map 和 可能的错误
func (s *MemoryService) GetVersions(memoryID uint64) (map[string]string, error) {
	var versions []model.MemoryVersion
	if err := database.DB.Where("memory_id = ?", memoryID).Find(&versions).Error; err != nil {
		return nil, err
	}
	// 把 slice 转为 map，方便前端使用
	result := make(map[string]string)
	for _, v := range versions {
		result[v.VersionType] = v.Content
	}
	return result, nil
}

// SaveVersion 保存指定记忆的某个版本。
//
// 特殊逻辑：当保存 "current" 版本时，
//  1. 先把旧的 current 内容备份为 "backup" 版本（旧的 backup 被覆盖）
//  2. 再更新 "current" 为新内容
//
// 对于 "custom" 或 "backup" 类型的版本：
//   - 如果该类型版本已存在 → 更新内容
//   - 如果不存在 → 创建新版本记录
//
// 参数：
//   - memoryID：记忆 ID
//   - versionType：版本类型
//   - content：版本内容
//
// 返回：可能的错误
func (s *MemoryService) SaveVersion(memoryID uint64, versionType string, content string) error {
	// ---- 处理 current 版本的自动备份 ----
	if versionType == "current" {
		var backup model.MemoryVersion
		// 查找当前的 current 版本
		if err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, "current").First(&backup).Error; err == nil {
			// 当前 current 版本存在，先备份旧内容到 backup 版本
			// 删除旧的 backup（如果有）
			database.DB.Model(&model.MemoryVersion{}).Where("memory_id = ? AND version_type = ?", memoryID, "backup").Delete(&model.MemoryVersion{})
			// 创建新的 backup（内容来自旧的 current）
			database.DB.Create(&model.MemoryVersion{
				MemoryID:    memoryID,
				VersionType: "backup",
				Content:     backup.Content,
			})
		}
	}

	// ---- 保存/更新版本 ----
	var existing model.MemoryVersion
	// 查找该类型版本是否已存在
	err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, versionType).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 不存在 → 创建新版本
		return database.DB.Create(&model.MemoryVersion{
			MemoryID:    memoryID,
			VersionType: versionType,
			Content:     content,
		}).Error
	}
	// 已存在 → 更新内容
	return database.DB.Model(&existing).Update("content", content).Error
}

// Restore 从 backup 版本恢复 current 版本。
//
// 其实就是把 backup 的内容复制到 current。
// 如果 backup 版本不存在，返回错误。
//
// 参数：
//   - memoryID：记忆 ID
//
// 返回：可能的错误
func (s *MemoryService) Restore(memoryID uint64) error {
	var backup model.MemoryVersion
	// 查找 backup 版本
	if err := database.DB.Where("memory_id = ? AND version_type = ?", memoryID, "backup").First(&backup).Error; err != nil {
		return err // backup 不存在，无法恢复
	}
	// 用 backup 的内容覆盖 current 版本
	return s.SaveVersion(memoryID, "current", backup.Content)
}
