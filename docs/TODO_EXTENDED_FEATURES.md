# Todo扩展功能实现指南

## ✅ 已完成的工作

### 1. Domain层更新 ✅
已在 `internal/domain/todo.go` 中添加：

```go
// 新增类型
type ItemStatus string  // not_started, in_progress, completed
type Priority string    // high, medium, low

// 扩展的TodoItem结构
type TodoItem struct {
    ID          int64
    ListID      int64
    Name        string     // 名称
    Description string     // 描述  
    Status      ItemStatus // 状态
    Priority    Priority   // 优先级
    DueDate     *time.Time // 截止日期
    Tags        string     // 标签(逗号分隔)
    // ... 其他字段
}

// 筛选和排序支持
type ItemFilter struct {
    Status    *ItemStatus
    Priority  *Priority
    DueBefore *time.Time
    DueAfter  *time.Time
    Tags      []string
}

type ItemSort struct {
    Field string // "due_date", "priority", "status", "name"
    Desc  bool
}
```

### 2. 分片表修复 ✅
- ✅ 16 User DBs，1024 `users_` 表
- ✅ 16 User DBs，1024 `user_list_index_` 表
- ✅ 64 Data DBs，4096 `todo_lists_tab_` 表
- ✅ 64 Data DBs，4096 `todo_items_tab_` 表
- ✅ 64 Data DBs，4096 `list_collaborators_tab_` 表

### 3. 代码清理 ✅
删除了废弃文件：
- ❌ `init_sharding_v2.go` ~ `v5.go`
- ❌ `find_shard.go` (不准确)
- ❌ `scripts/sharding_init.sql`

---

## 🔧 剩余实现步骤

### 步骤1: 更新数据库Schema

需要更新4096个 `todo_items_tab_*` 表的结构：

```sql
ALTER TABLE todo_items_tab_XXXX
    ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN description TEXT,
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'not_started',
    ADD COLUMN priority VARCHAR(10) NOT NULL DEFAULT 'medium',
    ADD COLUMN due_date DATETIME,
    ADD COLUMN tags VARCHAR(500),
    ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    ADD INDEX idx_status (status),
    ADD INDEX idx_priority (priority),
    ADD INDEX idx_due_date (due_date);
```

**实现工具**：创建 `cmd/tools/migrate_items_schema.go`

### 步骤2: 更新Repository层

在 `internal/repository/sharded_todo_repo_v2.go` 中实现：

```go
func (r *ShardedTodoRepoV2) GetItemsByListIDWithFilter(
    listID int64, 
    filter *domain.ItemFilter, 
    sort *domain.ItemSort,
) ([]domain.TodoItem, error) {
    // 1. 路由到正确的分片
    // 2. 构建SQL WHERE子句(根据filter)
    // 3. 构建ORDER BY子句(根据sort)
    // 4. 执行查询
    // 5. 返回结果
}

func (r *ShardedTodoRepoV2) CreateItemExtended(item *domain.TodoItem) error {
    // 插入包含所有新字段的item
}

func (r *ShardedTodoRepoV2) UpdateItemExtended(listID int64, item *domain.TodoItem) error {
    // 更新包含所有字段的item
}
```

### 步骤3: 更新Service层

在 `internal/service/todo_service.go` 中实现：

```go
func (s *todoService) CreateItemExtended(userID, listID int64, item *domain.TodoItem) (*domain.TodoItem, error) {
    // 验证权限
    // 设置默认值（如果未提供）
    // 调用repository创建
}

func (s *todoService) GetItemsFiltered(
    userID, listID int64,
    filter *domain.ItemFilter,
    sort *domain.ItemSort,
) ([]domain.TodoItem, error) {
    // 验证权限
    // 调用repository查询
}
```

同时更新 `internal/service/cached_todo_service.go` 添加缓存支持。

### 步骤4: 更新Handler层

在 `internal/handler/todo_handler.go` 中添加新端点：

```go
// POST /api/lists/{id}/items/extended
func (h *TodoHandler) CreateItemExtended(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Name        string    `json:"name"`
        Description string    `json:"description"`
        Status      string    `json:"status"`
        Priority    string    `json:"priority"`
        DueDate     string    `json:"due_date"` // ISO 8601 format
        Tags        []string  `json:"tags"`
    }
    // 解析请求
    // 调用service
    // 返回响应
}

// GET /api/lists/{id}/items/filtered
func (h *TodoHandler) GetItemsFiltered(w http.ResponseWriter, r *http.Request) {
    // 解析query参数: status, priority, due_before, due_after, tags, sort_by, sort_desc
    // 构建filter和sort
    // 调用service
    // 返回响应
}
```

### 步骤5: 更新前端

在 `web/app.js` 中添加：

```javascript
// 创建扩展item
async function createItemExtended(listId, itemData) {
    const res = await fetch(`${API_BASE}/lists/${listId}/items/extended`, {
        method: 'POST',
        headers: { 
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(itemData)
    });
    return res.json();
}

// 筛选和排序
async function getFilteredItems(listId, filter) {
    const params = new URLSearchParams(filter);
    const res = await fetch(`${API_BASE}/lists/${listId}/items/filtered?${params}`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });
    return res.json();
}
```

### 步骤6: 单元测试

创建 `internal/service/todo_service_extended_test.go`：

```go
func TestCreateItemExtended(t *testing.T) {
    // 测试创建包含所有扩展字段的item
}

func TestGetItemsFiltered_ByStatus(t *testing.T) {
    // 测试按状态筛选
}

func TestGetItemsFiltered_ByPriority(t *testing.T) {
    // 测试按优先级筛选
}

func TestGetItemsFiltered_ByDueDate(t *testing.T) {
    // 测试按截止日期筛选
}

func TestGetItemsFiltered_ByTags(t *testing.T) {
    // 测试按标签筛选
}

func TestGetItems_SortByDueDate(t *testing.T) {
    // 测试按截止日期排序
}

func TestGetItems_SortByPriority(t *testing.T) {
    // 测试按优先级排序
}
```

---

## 📋 实现工具脚本

### 1. Schema迁移工具

创建 `cmd/tools/migrate_items_schema.go`：

```go
// 遍历所有 64 个 todo_data_db
// 对每个DB中的 64 张 todo_items_tab_XXXX 表
// 执行 ALTER TABLE 添加新字段
```

### 2. 数据迁移工具

创建 `cmd/tools/migrate_items_data.go`：

```go
// 对于现有数据：
// - name = content (向后兼容)
// - status = completed if is_done else not_started
// - priority = medium (默认)
```

---

## 🧪 测试用例

### API测试用例

1. **创建扩展Item**
```bash
curl -X POST http://localhost:8080/api/lists/123/items/extended \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "完成项目文档",
    "description": "编写完整的API文档和使用指南",
    "status": "in_progress",
    "priority": "high",
    "due_date": "2025-12-15T23:59:59Z",
    "tags": ["文档", "紧急"]
  }'
```

2. **筛选Item（按状态）**
```bash
curl "http://localhost:8080/api/lists/123/items/filtered?status=in_progress" \
  -H "Authorization: Bearer TOKEN"
```

3. **筛选Item（按优先级和截止日期）**
```bash
curl "http://localhost:8080/api/lists/123/items/filtered?priority=high&due_before=2025-12-20" \
  -H "Authorization: Bearer TOKEN"
```

4. **排序Item**
```bash
curl "http://localhost:8080/api/lists/123/items/filtered?sort_by=due_date&sort_desc=false" \
  -H "Authorization: Bearer TOKEN"
```

5. **按标签筛选**
```bash
curl "http://localhost:8080/api/lists/123/items/filtered?tags=紧急,重要" \
  -H "Authorization: Bearer TOKEN"
```

---

## 📊 更新Postman集合

在 `docs/TodoApp_Postman_Collection.json` 中添加：

```json
{
  "name": "Create Item (Extended)",
  "request": {
    "method": "POST",
    "url": "{{base_url}}/lists/{{list_id}}/items/extended",
    "body": {
      "mode": "raw",
      "raw": "{\"name\":\"...\",\"description\":\"...\",\"status\":\"...\",\"priority\":\"...\",\"due_date\":\"...\",\"tags\":[...]}"
    }
  }
},
{
  "name": "Get Items (Filtered)",
  "request": {
    "method": "GET",
    "url": "{{base_url}}/lists/{{list_id}}/items/filtered?status=in_progress&priority=high"
  }
}
```

---

## 🎯 优先级建议

1. **高优先级（核心功能）**：
   - ✅ Schema迁移（添加新字段）
   - ✅ Repository层实现（CRUD + 筛选排序）
   - ✅ Service层实现
   - ✅ Handler层实现

2. **中优先级（增强功能）**：
   - ⚠️ 缓存支持（扩展 cached_todo_service.go）
   - ⚠️ 前端UI更新
   - ⚠️ Postman集合更新

3. **低优先级（优化）**：
   - 📊 性能优化（索引调优）
   - 📊 批量操作API
   - 📊 导出/导入功能

---

## 📝 已提供的基础

- ✅ Domain模型已更新（`ItemStatus`, `Priority`, `ItemFilter`, `ItemSort`）
- ✅ Repository/Service接口已扩展
- ✅ 分片架构已就绪（4096个todo_items表）
- ✅ 一致性哈希路由已实现
- ✅ 缓存层已就绪

**下一步**：执行Schema迁移，然后实现Repository、Service、Handler层代码。

