# TodoList App - 完整测试计划

## 🎯 测试目标

- **单元测试覆盖率**: > 80%
- **集成测试**: 所有API端点
- **性能测试**: 达到设计目标（5K WQPS, 50K RQPS）
- **压力测试**: 10亿用户数据

---

## 1. 单元测试 (Unit Tests)

### 1.1 Domain层测试

**文件**: `internal/domain/todo_test.go`

```go
package domain_test

import (
    "testing"
    "todolist-app/internal/domain"
)

func TestItemStatus_Valid(t *testing.T) {
    validStatuses := []domain.ItemStatus{
        domain.StatusNotStarted,
        domain.StatusInProgress,
        domain.StatusCompleted,
    }
    for _, status := range validStatuses {
        if status == "" {
            t.Errorf("Status should not be empty: %v", status)
        }
    }
}

func TestPriority_Valid(t *testing.T) {
    validPriorities := []domain.Priority{
        domain.PriorityHigh,
        domain.PriorityMedium,
        domain.PriorityLow,
    }
    for _, priority := range validPriorities {
        if priority == "" {
            t.Errorf("Priority should not be empty: %v", priority)
        }
    }
}
```

### 1.2 Service层测试 ✅

**已有**: `internal/service/auth_service_test.go`  
**已有**: `internal/service/todo_service_test.go`

**需要添加**: `internal/service/todo_service_extended_test.go`

```go
func TestCreateItemExtended_Success(t *testing.T)
func TestCreateItemExtended_InvalidPriority(t *testing.T)
func TestGetItemsFiltered_ByStatus(t *testing.T)
func TestGetItemsFiltered_ByPriority(t *testing.T)
func TestGetItemsFiltered_ByDueDate(t *testing.T)
func TestGetItemsFiltered_ByTags(t *testing.T)
func TestGetItemsSorted_ByDueDate(t *testing.T)
func TestGetItemsSorted_ByPriority(t *testing.T)
func TestGetItemsSorted_ByName(t *testing.T)
```

### 1.3 Repository层测试

**需要添加**: `internal/repository/sharded_todo_repo_test.go`

```go
func TestShardedTodoRepo_CreateItem(t *testing.T)
func TestShardedTodoRepo_GetItemsByListID(t *testing.T)
func TestShardedTodoRepo_UpdateItemWithListID(t *testing.T)
func TestShardedTodoRepo_DeleteItemWithListID(t *testing.T)
func TestShardedTodoRepo_GetItemsWithFilter(t *testing.T)
func TestShardedTodoRepo_ConsistentHashing(t *testing.T)
```

### 1.4 Infrastructure层测试

**需要添加**: `internal/infrastructure/sharding/router_v2_test.go`

```go
func TestRouterV2_GetUserDB(t *testing.T)
func TestRouterV2_GetTodoDB(t *testing.T)
func TestRouterV2_ConsistentHashDistribution(t *testing.T)
```

---

## 2. 集成测试 (Integration Tests)

### 2.1 API端点测试

**工具**: `test_api.sh` (已存在) ✅

**扩展测试**: `test_api_extended.sh`

```bash
#!/bin/bash
# 测试所有API端点

# 1. 注册和登录
test_auth()

# 2. 创建List
test_create_list()

# 3. 添加基本Item
test_add_basic_item()

# 4. 添加扩展Item（包含所有字段）
test_add_extended_item()

# 5. 筛选Item（按状态）
test_filter_by_status()

# 6. 筛选Item（按优先级）
test_filter_by_priority()

# 7. 筛选Item（按截止日期）
test_filter_by_due_date()

# 8. 筛选Item（按标签）
test_filter_by_tags()

# 9. 排序Item（按截止日期）
test_sort_by_due_date()

# 10. 排序Item（按优先级）
test_sort_by_priority()

# 11. 更新Item
test_update_item()

# 12. 删除Item
test_delete_item()

# 13. 分享List
test_share_list()

# 14. 多用户协作
test_collaboration()

# 15. CAPTCHA
test_captcha()
```

### 2.2 分片路由测试

**工具**: `test_sharding.sh`

```bash
#!/bin/bash
# 测试分片路由正确性

# 1. 验证所有表存在
go run cmd/tools/verify_sharding.go

# 2. 测试User路由
test_user_routing() {
    for i in {0..100}; do
        USER_ID=$((RANDOM * 1000000))
        # 验证路由到正确的DB和Table
    done
}

# 3. 测试List路由
test_list_routing() {
    for i in {0..100}; do
        LIST_ID=$((RANDOM * 1000000))
        # 验证路由到正确的DB和Table
    done
}

# 4. 测试数据分布均匀性
test_distribution()
```

---

## 3. 性能测试 (Performance Tests)

### 3.1 数据生成 ✅

**工具**: `cmd/tools/benchmark_data_gen.go` (已存在)

**用法**:
```bash
# 生成100万用户（测试）
go run cmd/tools/benchmark_data_gen.go -users=1000000 -lists=10 -items=10

# 生成10亿用户（压力测试）
go run cmd/tools/benchmark_data_gen.go -users=1000000000 -lists=10 -items=10 -workers=50
```

### 3.2 API压力测试 ✅

**工具**: `cmd/tools/benchmark_api.go` (已存在)

**测试场景**:

| 测试类型 | 目标QPS | 并发数 | 持续时间 |
|---------|---------|--------|---------|
| 注册 | 1,000 | 100 | 60s |
| 登录 | 10,000 | 200 | 60s |
| 查询List | 50,000 | 500 | 120s |
| 创建Item | 5,000 | 100 | 60s |
| 更新Item | 5,000 | 100 | 60s |
| 筛选查询 | 30,000 | 400 | 120s |

**运行示例**:
```bash
# 测试登录QPS
go run cmd/tools/benchmark_api.go -test=login -duration=60 -concurrency=200

# 测试查询QPS
go run cmd/tools/benchmark_api.go -test=query -duration=120 -concurrency=500

# 测试所有API
go run cmd/tools/benchmark_api.go -test=all -duration=60 -concurrency=100
```

### 3.3 数据库性能测试

**工具**: `test_db_performance.sh`

```bash
#!/bin/bash
# 测试数据库查询性能

# 1. 单表查询性能
test_single_table_query() {
    # 测试在1000万条记录的表中查询
    time mysql -e "SELECT * FROM users_0000 WHERE user_id = 123456 LIMIT 1"
}

# 2. 索引效率测试
test_index_performance() {
    # 测试有索引 vs 无索引的查询速度
}

# 3. JOIN性能测试
test_join_performance() {
    # 测试跨表JOIN（应避免跨分片）
}

# 4. 缓存命中率测试
test_cache_hit_rate() {
    # 测试Redis缓存命中率
}
```

---

## 4. 端到端测试 (E2E Tests)

### 4.1 用户场景测试

**场景1: 新用户注册并创建Todo**
```
1. 注册新用户
2. 验证邮箱
3. 登录
4. 创建Todo List
5. 添加3个Item（不同优先级）
6. 按优先级排序查看
7. 标记1个Item为完成
8. 删除1个Item
```

**场景2: 多用户协作**
```
1. 用户A创建List
2. 用户A分享给用户B（编辑权限）
3. 用户B添加Item
4. 用户A查看用户B添加的Item
5. 用户B更新Item状态
6. 用户A删除List
```

**场景3: 复杂筛选和排序**
```
1. 创建包含20个Item的List
2. 设置不同的优先级、状态、截止日期、标签
3. 筛选：status=in_progress
4. 筛选：priority=high
5. 筛选：due_before=明天
6. 筛选：tags包含"紧急"
7. 排序：按截止日期升序
8. 排序：按优先级降序
```

### 4.2 自动化E2E测试

**工具**: Selenium / Playwright

**文件**: `tests/e2e/test_workflows.js`

```javascript
describe('TodoList E2E Tests', () => {
    test('User registration and login', async () => {
        // 1. Navigate to homepage
        // 2. Click Register
        // 3. Fill form
        // 4. Submit
        // 5. Verify email
        // 6. Login
    });

    test('Create and manage todo list', async () => {
        // 1. Login
        // 2. Create list
        // 3. Add items
        // 4. Mark as done
        // 5. Delete items
    });

    test('Collaboration workflow', async () => {
        // 1. User A creates list
        // 2. User A shares with User B
        // 3. User B sees shared list
        // 4. User B edits items
        // 5. User A sees changes
    });
});
```

---

## 5. 安全测试 (Security Tests)

### 5.1 认证测试

```bash
# 1. 未授权访问测试
curl http://localhost:8080/api/lists
# 预期: 401 Unauthorized

# 2. 无效Token测试
curl -H "Authorization: Bearer invalid_token" http://localhost:8080/api/lists
# 预期: 401 Unauthorized

# 3. 跨用户访问测试
# 用户A的token访问用户B的数据
# 预期: 403 Forbidden
```

### 5.2 SQL注入测试

```bash
# 1. 尝试SQL注入
curl -X POST http://localhost:8080/api/auth/login \
  -d '{"email":"admin@test.com'\'' OR 1=1--","password":"any"}'
# 预期: 登录失败，无SQL注入

# 2. 尝试在Item名称中注入
curl -X POST http://localhost:8080/api/lists/123/items/extended \
  -H "Authorization: Bearer TOKEN" \
  -d '{"name":"Item'; DROP TABLE users; --"}'
# 预期: 创建成功，但不执行SQL命令
```

### 5.3 XSS测试

```bash
# 尝试在Item描述中注入脚本
curl -X POST http://localhost:8080/api/lists/123/items/extended \
  -H "Authorization: Bearer TOKEN" \
  -d '{"name":"Test","description":"<script>alert(\"XSS\")</script>"}'
# 预期: 存储时转义，输出时不执行脚本
```

---

## 6. 测试自动化

### 6.1 CI/CD集成

**文件**: `.github/workflows/test.yml`

```yaml
name: Run Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      - run: go test -v ./...
      - run: go test -cover ./...

  integration-tests:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
        env:
          MYSQL_ROOT_PASSWORD: test123
      redis:
        image: redis:7
    steps:
      - uses: actions/checkout@v2
      - run: ./test_api.sh

  performance-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - run: go run cmd/tools/benchmark_api.go -test=all -duration=30
```

### 6.2 测试报告

**工具**: `gocov`, `gocov-html`

```bash
# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# 查看报告
open coverage.html
```

---

## 7. 测试检查清单

### ✅ 单元测试
- [ ] Domain层测试
- [x] Service层测试（基础）
- [ ] Service层测试（扩展功能）
- [ ] Repository层测试
- [ ] Infrastructure层测试

### ✅ 集成测试
- [x] 基础API测试（test_api.sh）
- [ ] 扩展API测试
- [ ] 分片路由测试
- [ ] 缓存测试

### ✅ 性能测试
- [x] 数据生成工具
- [x] API压力测试工具
- [ ] 数据库性能测试
- [ ] 缓存性能测试

### ⚠️ 端到端测试
- [ ] 用户注册流程
- [ ] Todo CRUD流程
- [ ] 协作流程
- [ ] 筛选排序流程

### ⚠️ 安全测试
- [ ] 认证测试
- [ ] 授权测试
- [ ] SQL注入测试
- [ ] XSS测试

---

## 8. 测试执行顺序

### 阶段1: 快速验证
```bash
# 1. 单元测试（2分钟）
go test -v ./internal/service/...

# 2. 快速API测试（1分钟）
./quick_test.sh

# 3. 分片验证（10秒）
go run cmd/tools/verify_sharding.go
```

### 阶段2: 全面测试
```bash
# 1. 所有单元测试（5分钟）
go test -v ./...

# 2. 集成测试（3分钟）
./test_api.sh

# 3. 性能测试（5分钟）
go run cmd/tools/benchmark_api.go -test=all -duration=60
```

### 阶段3: 压力测试
```bash
# 1. 生成大量数据（数小时）
go run cmd/tools/benchmark_data_gen.go -users=100000000

# 2. 长时间压力测试（数小时）
go run cmd/tools/benchmark_api.go -test=all -duration=3600
```

---

## 9. 性能基准

### 当前实现

| 指标 | 目标 | 当前状态 |
|-----|------|---------|
| 写QPS | 5,000 | ⏳ 待测试 |
| 读QPS | 50,000 | ⏳ 待测试 |
| 日活用户 | 100M | ✅ 架构支持 |
| 总用户 | 10亿 | ✅ 架构支持 |
| 平均响应时间 | < 100ms | ⏳ 待测试 |
| P99响应时间 | < 500ms | ⏳ 待测试 |

### 运行基准测试
```bash
# 启动所有服务
./start.sh

# 运行基准测试
go run cmd/tools/benchmark_api.go -test=all -duration=120 -concurrency=500 > benchmark_results.txt

# 分析结果
cat benchmark_results.txt
```

---

## 📋 测试优先级

### P0 (必须)
- ✅ 基础API测试
- ✅ 分片验证
- ⏳ Service层单元测试

### P1 (重要)
- ⏳ Repository层测试
- ⏳ 扩展功能测试
- ⏳ 性能基准测试

### P2 (建议)
- ⏳ E2E自动化测试
- ⏳ 安全测试
- ⏳ 负载测试（10亿用户）

---

## 🎯 结论

已提供的测试工具:
- ✅ `test_api.sh` - API集成测试
- ✅ `quick_test.sh` - 快速健康检查
- ✅ `verify_sharding.go` - 分片验证
- ✅ `benchmark_data_gen.go` - 数据生成
- ✅ `benchmark_api.go` - API压力测试

**下一步**: 补充单元测试和E2E测试，达到80%覆盖率。

