package domain

import "time"

type Role string

const (
	RoleOwner  Role = "OWNER"
	RoleEditor Role = "EDITOR"
	RoleViewer Role = "VIEWER"
)

// ItemStatus represents the status of a todo item
type ItemStatus string

const (
	StatusNotStarted ItemStatus = "not_started"
	StatusInProgress ItemStatus = "in_progress"
	StatusCompleted  ItemStatus = "completed"
)

// Priority represents the priority level of a todo item
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// TodoList represents a collection of items
type TodoList struct {
	ID        int64     `json:"id" db:"list_id"`
	OwnerID   int64     `json:"owner_id" db:"owner_id"`
	Title     string    `json:"title" db:"title"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Role      Role      `json:"role,omitempty"` // For output only
}

// TodoItem represents a single task with extended attributes
type TodoItem struct {
	ID          int64      `json:"id" db:"item_id"`
	ListID      int64      `json:"list_id" db:"list_id"`
	Name        string     `json:"name" db:"name"`                             // 名称
	Description string     `json:"description,omitempty" db:"description"`     // 描述
	Content     string     `json:"content,omitempty" db:"content"`             // 保留兼容性
	Status      ItemStatus `json:"status" db:"status"`                         // 状态
	Priority    Priority   `json:"priority" db:"priority"`                     // 优先级
	DueDate     *time.Time `json:"due_date,omitempty" db:"due_date"`           // 截止日期
	Tags        string     `json:"tags,omitempty" db:"tags"`                   // 标签(逗号分隔)
	IsDone      bool       `json:"is_done" db:"is_done"`                       // 保留兼容性
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty" db:"updated_at"`
}

// ItemFilter represents filter criteria for querying items
type ItemFilter struct {
	Status   *ItemStatus // Filter by status
	Priority *Priority   // Filter by priority
	DueBefore *time.Time // Due date before
	DueAfter  *time.Time // Due date after
	Tags      []string   // Filter by tags (any match)
}

// ItemSort represents sort criteria
type ItemSort struct {
	Field string // "due_date", "priority", "status", "name", "created_at"
	Desc  bool   // Descending order
}

// TodoRepository defines data persistence for lists and items
type TodoRepository interface {
	CreateList(list *TodoList) error
	GetListsByUserID(userID int64) ([]TodoList, error)
	GetListByID(listID int64) (*TodoList, error)
	DeleteList(listID int64) error
	
	AddCollaborator(listID, userID int64, role Role) error
	
	CreateItem(item *TodoItem) error
	GetItemsByListID(listID int64) ([]TodoItem, error)
	GetItemsByListIDWithFilter(listID int64, filter *ItemFilter, sort *ItemSort) ([]TodoItem, error)
	UpdateItemWithListID(listID int64, item *TodoItem) error
	DeleteItemWithListID(listID, itemID int64) error
}

// TodoService defines business logic
type TodoService interface {
	CreateList(userID int64, title string) (*TodoList, error)
	GetLists(userID int64) ([]TodoList, error)
	DeleteList(userID, listID int64) error
	ShareList(ownerID, listID int64, targetEmail string, role Role) error
	
	// Item operations (basic - for backward compatibility)
	AddItem(userID, listID int64, content string) (*TodoItem, error)
	GetItems(userID, listID int64) ([]TodoItem, error)
	UpdateItem(userID, listID, itemID int64, isDone bool) (*TodoItem, error)
	DeleteItem(userID, listID, itemID int64) error
	
	// Item operations (extended)
	CreateItemExtended(userID, listID int64, item *TodoItem) (*TodoItem, error)
	UpdateItemExtended(userID, listID int64, item *TodoItem) (*TodoItem, error)
	GetItemsFiltered(userID, listID int64, filter *ItemFilter, sort *ItemSort) ([]TodoItem, error)
}

