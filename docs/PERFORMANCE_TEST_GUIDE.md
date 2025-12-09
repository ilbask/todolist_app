## TodoList App - 性能测试完整指南

## 🎯 测试目标

| 指标 | 目标值 |
|-----|-------|
| 日活用户 (DAU) | 100,000,000 |
| 写入QPS (WQPS) | 5,000 |
| 读取QPS (RQPS) | 50,000 |
| 平均响应时间 | < 100ms |
| P99响应时间 | < 500ms |
| 数据规模 | 10亿用户 |

---

## 📋 测试准备

### 1. 环境要求

**最低配置** (小规模测试: 10万用户):
- CPU: 4核
- 内存: 8GB
- 磁盘: 50GB可用空间
- MySQL: 8.0+

**推荐配置** (大规模测试: 10亿用户):
- CPU: 16核+
- 内存: 32GB+
- 磁盘: 1TB+ SSD
- MySQL: 8.0+ (优化配置)

### 2. 启动准备

```bash
# 1. 确保MySQL运行
brew services start mysql

# 2. 初始化分片数据库
export DB_PASS="115119_hH"
go run cmd/tools/init_sharding_v6.go

# 3. 验证分片
go run cmd/tools/verify_sharding.go

# 4. 启动应用
./start.sh

# 5. 验证应用
./quick_test.sh
```

---

## 🚀 执行测试

### 方案A: 渐进式测试（推荐新手）

适合：首次测试、验证系统、小规模测试

```bash
# 设置脚本权限
chmod +x performance_test.sh

# 运行测试（交互式）
./performance_test.sh
```

**测试流程**:
1. ✅ 自动生成10万用户数据
2. ✅ 运行所有API压测（60秒）
3. 💬 询问是否继续100万用户测试
4. 💬 询问是否继续10亿用户测试

**预计时间**:
- 10万用户: 5-10分钟
- 100万用户: 20-40分钟
- 10亿用户: 12-24小时

---

### 方案B: 10亿用户完整测试（后台运行）

适合：生产环境验证、长时间压测

```bash
# 设置权限
chmod +x performance_test_1b.sh

# 后台运行（推荐）
nohup ./performance_test_1b.sh > performance_1b.log 2>&1 &

# 查看进程
ps aux | grep performance_test_1b

# 实时查看日志
tail -f performance_1b.log

# 查看进度
tail -f performance_results/*/data_gen_1billion.log
```

**测试流程**:
1. ✅ 生成10亿用户数据（12-24小时）
2. ✅ 登录接口压测（5分钟）
3. ✅ 查询接口压测（5分钟）
4. ✅ 创建接口压测（5分钟）
5. ✅ 更新接口压测（5分钟）
6. ✅ 删除接口压测（3分钟）
7. ✅ 分享接口压测（3分钟）
8. ✅ 综合压测（10分钟）

**预计总时间**: 13-25小时

---

### 方案C: 单独测试（灵活控制）

适合：针对性测试、快速验证

#### 1. 数据生成测试

```bash
# 小规模（10万用户，2分钟）
go run cmd/tools/benchmark_data_gen.go -users=100000 -lists=10 -items=10 -workers=20

# 中规模（100万用户，20分钟）
go run cmd/tools/benchmark_data_gen.go -users=1000000 -lists=10 -items=10 -workers=50

# 大规模（1000万用户，3小时）
go run cmd/tools/benchmark_data_gen.go -users=10000000 -lists=10 -items=10 -workers=80

# 超大规模（10亿用户，12-24小时）
nohup go run cmd/tools/benchmark_data_gen.go \
    -users=1000000000 \
    -lists=10 \
    -items=10 \
    -workers=100 \
    -batch=5000 \
    > data_gen.log 2>&1 &
```

#### 2. API压力测试

```bash
# 登录接口
go run cmd/tools/benchmark_api.go -test=login -duration=60 -concurrency=200

# 查询接口
go run cmd/tools/benchmark_api.go -test=query -duration=60 -concurrency=500

# 创建接口
go run cmd/tools/benchmark_api.go -test=create -duration=60 -concurrency=100

# 更新接口
go run cmd/tools/benchmark_api.go -test=update -duration=60 -concurrency=100

# 删除接口
go run cmd/tools/benchmark_api.go -test=delete -duration=30 -concurrency=50

# 分享接口
go run cmd/tools/benchmark_api.go -test=share -duration=30 -concurrency=50

# 全部接口
go run cmd/tools/benchmark_api.go -test=all -duration=120 -concurrency=300
```

---

## 📊 查看结果

### 1. 实时监控

```bash
# 查看应用日志
tail -f log/app.log

# 查看MySQL状态
mysql -u root -p -e "SHOW PROCESSLIST;"
mysql -u root -p -e "SHOW STATUS LIKE 'Threads_%';"
mysql -u root -p -e "SHOW STATUS LIKE 'Questions';"

# 查看Redis状态（如启用）
redis-cli INFO stats
redis-cli INFO keyspace

# 系统资源监控
top
htop
iostat -x 1
```

### 2. 测试结果

测试完成后，结果保存在 `performance_results/` 目录：

```bash
cd performance_results

# 列出所有测试
ls -lh

# 查看最新测试
cd $(ls -t | head -1)

# 查看报告
cat REPORT.md
# 或
cat FINAL_REPORT.md

# 查看具体日志
cat data_gen_*.log          # 数据生成日志
cat benchmark_login.log     # 登录压测
cat benchmark_query.log     # 查询压测
cat benchmark_create.log    # 创建压测
cat benchmark_update.log    # 更新压测
cat benchmark_share.log     # 分享压测
cat benchmark_all.log       # 综合压测
```

### 3. 性能指标提取

```bash
# 提取所有QPS数据
grep "QPS:" performance_results/latest/*.log

# 提取延迟数据
grep "Avg Latency:" performance_results/latest/*.log

# 提取成功率
grep "Success:" performance_results/latest/*.log

# 查看失败请求
grep "Failure:" performance_results/latest/*.log
```

---

## 📈 性能分析

### 1. 数据库性能

```bash
# 查看表大小
mysql -u root -p -e "
SELECT 
    TABLE_SCHEMA, 
    SUM(DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024 / 1024 AS size_gb,
    COUNT(*) AS table_count
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_%'
GROUP BY TABLE_SCHEMA
ORDER BY size_gb DESC;
"

# 查看总行数
mysql -u root -p -e "
SELECT 
    SUM(TABLE_ROWS) AS total_rows
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_user_db_%' 
AND TABLE_NAME LIKE 'users_%';
"

# 慢查询日志
mysql -u root -p -e "SHOW VARIABLES LIKE 'slow_query%';"
mysql -u root -p -e "SHOW GLOBAL STATUS LIKE 'Slow_queries';"
```

### 2. 分片分布验证

```bash
# 验证数据均匀分布
for i in {0..15}; do
    COUNT=$(mysql -u root -p115119_hH -e "
        SELECT SUM(TABLE_ROWS) 
        FROM information_schema.TABLES 
        WHERE TABLE_SCHEMA = 'todo_user_db_$i' 
        AND TABLE_NAME LIKE 'users_%';
    " | tail -1)
    echo "todo_user_db_$i: $COUNT users"
done
```

### 3. 缓存性能（如启用Redis）

```bash
# Redis命中率
redis-cli INFO stats | grep keyspace

# 缓存key统计
redis-cli DBSIZE

# 内存使用
redis-cli INFO memory | grep used_memory_human
```

---

## 🔧 性能优化

### 遇到性能瓶颈时的优化步骤

#### 1. MySQL优化

编辑 `/etc/my.cnf` 或 `/usr/local/etc/my.cnf`:

```ini
[mysqld]
# InnoDB缓冲池（建议设置为物理内存的70-80%）
innodb_buffer_pool_size = 16G

# InnoDB日志文件
innodb_log_file_size = 1G

# 连接数
max_connections = 1000

# 查询缓存（MySQL 8.0已移除）
# query_cache_size = 256M

# InnoDB并发线程
innodb_thread_concurrency = 16

# InnoDB IO容量
innodb_io_capacity = 2000
innodb_io_capacity_max = 4000

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 1
```

重启MySQL:
```bash
brew services restart mysql
```

#### 2. 应用层优化

修改 `cmd/api/main.go`:

```go
// 增加数据库连接池
db.SetMaxOpenConns(200)  // 从100增加到200
db.SetMaxIdleConns(100)  // 从50增加到100
db.SetConnMaxLifetime(10 * time.Minute)
```

#### 3. 启用Redis缓存

```bash
# 安装并启动Redis
brew install redis
brew services start redis

# 重启应用以启用缓存
./stop.sh && ./start.sh
```

#### 4. 增加分片数量

如果单个分片压力过大，考虑扩展：

```bash
# 从16个User DB扩展到32个
# 需要重新设计分片策略和数据迁移
```

---

## 🐛 常见问题

### Q1: 数据生成速度太慢

**原因**: 
- 磁盘IO瓶颈
- MySQL配置不当
- 网络延迟

**解决**:
```bash
# 1. 使用SSD
# 2. 调整batch size
go run cmd/tools/benchmark_data_gen.go -batch=10000

# 3. 增加worker数量
go run cmd/tools/benchmark_data_gen.go -workers=200

# 4. 关闭binlog（测试环境）
mysql -e "SET GLOBAL sql_log_bin = 0;"
```

### Q2: API压测失败率高

**原因**:
- 连接池不足
- 数据库超载
- 网络超时

**解决**:
```bash
# 1. 降低并发数
go run cmd/tools/benchmark_api.go -concurrency=50

# 2. 增加超时时间
# 修改 cmd/tools/benchmark_api.go 中的 client.Timeout

# 3. 检查数据库连接
mysql -e "SHOW PROCESSLIST;" | wc -l
```

### Q3: 磁盘空间不足

**预估空间**:
- 1亿用户: ~50GB
- 10亿用户: ~500GB
- 100亿用户: ~5TB

**解决**:
```bash
# 检查空间
df -h

# 清理旧测试数据
go run cmd/tools/cleanup_db.go

# 或手动删除
mysql -u root -p -e "DROP DATABASE todo_user_db_0;"
```

### Q4: 内存不足

**解决**:
```bash
# 1. 降低MySQL缓冲池
innodb_buffer_pool_size = 4G  # 从16G降低

# 2. 降低worker数量
go run cmd/tools/benchmark_data_gen.go -workers=10

# 3. 分批测试
# 先测试1亿，清理后再测试下一个1亿
```

---

## 📝 测试清单

### 小规模测试 (10万用户)
- [ ] 分片验证通过
- [ ] 应用正常运行
- [ ] 数据生成成功
- [ ] 登录压测完成
- [ ] 查询压测完成
- [ ] CRUD压测完成
- [ ] 分享压测完成

### 中规模测试 (100万用户)
- [ ] 数据生成完成
- [ ] 所有API压测完成
- [ ] 性能指标达标
- [ ] 数据分布均匀

### 大规模测试 (10亿用户)
- [ ] 充足的磁盘空间（500GB+）
- [ ] 充足的内存（32GB+）
- [ ] 数据生成完成（12-24小时）
- [ ] 所有API压测完成
- [ ] 性能报告生成
- [ ] 达到设计目标

---

## 🎯 预期结果示例

### 理想性能指标

```
数据生成:
  - 10万用户: 2-5分钟
  - 100万用户: 20-40分钟
  - 10亿用户: 12-24小时
  - 生成速度: > 10,000 用户/秒

API性能:
  - 登录QPS: > 10,000
  - 查询QPS: > 50,000
  - 创建QPS: > 5,000
  - 更新QPS: > 5,000
  - 平均延迟: < 100ms
  - 成功率: > 99%
```

---

## 📧 报告问题

如果测试中遇到问题：

1. 查看日志: `log/app.log`
2. 查看测试日志: `performance_results/*/`
3. 检查系统资源: `top`, `df -h`
4. 检查MySQL: `SHOW PROCESSLIST`
5. 提交issue附带日志和配置

---

## 🚀 下一步

测试完成后：

1. ✅ 分析性能报告
2. ✅ 识别瓶颈
3. ✅ 应用优化建议
4. ✅ 重新测试验证
5. ✅ 记录最终性能数据
6. ✅ 部署到生产环境

Good Luck! 🎉

