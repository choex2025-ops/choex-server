package service

import (
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

// CalendarService 日程服务，负责日程的增删改查。
//
// 数据隔离：所有操作都必须传入 userID，确保用户只能操作自己的日程。
// 这是多用户系统的基本安全要求——"用户 A 不能看到/修改用户 B 的数据"。
type CalendarService struct{}

// NewCalendarService 创建日程服务实例。
func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

// List 获取指定用户的所有日程，按开始时间升序排列。
//
// GORM 查询链解析：
//
//	database.DB                          → 获取数据库连接
//	  .Where("user_id = ?", userID)      → 添加过滤条件：只查当前用户的数据
//	  .Order("start_time ASC")           → 按开始时间升序排列（ASC = ascending）
//	  .Find(&events)                     → 执行查询，结果填充到 events 切片
//
// 参数：
//   - userID：要查询的用户 ID
//
// 返回：日程列表 和 可能的错误
func (s *CalendarService) List(userID uint64) ([]model.Event, error) {
	var events []model.Event
	// Where("user_id = ?", userID) 中的 ? 是参数占位符
	// GORM 会把它替换为 userID 的值，这种方式可以防止 SQL 注入
	err := database.DB.Where("user_id = ?", userID).Order("start_time ASC").Find(&events).Error
	return events, err
}

// Create 创建一条新的日程记录。
//
// 参数：
//   - event：日程数据指针（UserID 应该在调用前设置好）
//
// 返回：可能的错误
func (s *CalendarService) Create(event *model.Event) error {
	return database.DB.Create(event).Error
}

// Update 更新指定日程的字段。
//
// 使用 map[string]any 作为更新数据的好处：
//   可以用一个接口实现部分更新（只更新传过来的字段，不传的保持原样）
//   例如：只传 {"title": "新标题"} 就只改标题，其他字段不变
//
// 安全性：同时检查 id 和 user_id，防止用户 A 修改用户 B 的日程
//
// 参数：
//   - id：要更新的日程 ID
//   - userID：当前登录用户的 ID
//   - updates：要更新的字段键值对
//
// 返回：可能的错误
func (s *CalendarService) Update(id uint64, userID uint64, updates map[string]any) error {
	return database.DB.Model(&model.Event{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

// Delete 删除指定日程。
//
// 安全性：同时检查 id 和 user_id，防止越权删除
//
// 参数：
//   - id：要删除的日程 ID
//   - userID：当前登录用户的 ID
//
// 返回：可能的错误
func (s *CalendarService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Event{}).Error
}
