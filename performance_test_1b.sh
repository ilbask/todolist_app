#!/bin/bash

# TodoList App - 10亿用户压力测试 (后台运行)
# 使用方法: nohup ./performance_test_1b.sh > performance_1b.log 2>&1 &

set -e

DB_PASS=${DB_PASS:-"115119_hH"}
DB_USER=${DB_USER:-"root"}

# 创建结果目录
mkdir -p performance_results
TEST_DIR="performance_results/1billion_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$TEST_DIR"

echo "=========================================="
echo "   10亿用户压力测试"
echo "=========================================="
echo "启动时间: $(date)"
echo "结果目录: $TEST_DIR"
echo ""

# 函数：记录时间戳
log_time() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 函数：检查磁盘空间
check_disk_space() {
    AVAILABLE=$(df -h . | tail -1 | awk '{print $4}')
    log_time "可用磁盘空间: $AVAILABLE"
}

# 函数：检查内存
check_memory() {
    if command -v free &> /dev/null; then
        FREE_MEM=$(free -h | grep Mem | awk '{print $4}')
        log_time "可用内存: $FREE_MEM"
    else
        log_time "可用内存: $(vm_stat | grep free | awk '{print $3}') pages"
    fi
}

# ==========================================
# 预检查
# ==========================================
log_time "执行预检查..."
check_disk_space
check_memory

# 检查MySQL连接
if ! mysql -u "$DB_USER" -p"$DB_PASS" -e "SELECT 1" > /dev/null 2>&1; then
    log_time "❌ MySQL连接失败！"
    exit 1
fi
log_time "✅ MySQL连接正常"

# 检查应用
if ! curl -s http://localhost:8080/api/captcha/generate > /dev/null 2>&1; then
    log_time "❌ 应用未运行！"
    exit 1
fi
log_time "✅ 应用运行正常"

echo ""

# ==========================================
# 阶段1: 数据生成 (10亿用户)
# ==========================================
log_time "=========================================="
log_time "   阶段1: 生成10亿用户数据"
log_time "=========================================="
log_time "目标: 1,000,000,000 用户"
log_time "每用户: 10个List"
log_time "每List: 10个Item"
log_time "总计: 10B Lists, 100B Items"
log_time ""

DATA_GEN_START=$(date +%s)

go run cmd/tools/benchmark_data_gen.go \
    -users=1000000000 \
    -lists=10 \
    -items=10 \
    -workers=100 \
    -batch=5000 \
    2>&1 | tee "$TEST_DIR/data_gen_1billion.log"

DATA_GEN_END=$(date +%s)
DATA_GEN_DURATION=$((DATA_GEN_END - DATA_GEN_START))

log_time "✅ 数据生成完成！"
log_time "用时: $((DATA_GEN_DURATION / 3600))小时 $((DATA_GEN_DURATION % 3600 / 60))分钟"
log_time "速度: $((1000000000 / DATA_GEN_DURATION)) 用户/秒"
log_time ""

check_disk_space
echo ""

# ==========================================
# 阶段2: 登录接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段2: 登录接口压测"
log_time "=========================================="
log_time "并发: 500, 持续: 300秒 (5分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=login \
    -duration=300 \
    -concurrency=500 \
    2>&1 | tee "$TEST_DIR/benchmark_login_1b.log"

log_time "✅ 登录压测完成"
echo ""

# ==========================================
# 阶段3: 查询接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段3: 查询接口压测"
log_time "=========================================="
log_time "并发: 1000, 持续: 300秒 (5分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=query \
    -duration=300 \
    -concurrency=1000 \
    2>&1 | tee "$TEST_DIR/benchmark_query_1b.log"

log_time "✅ 查询压测完成"
echo ""

# ==========================================
# 阶段4: 创建接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段4: 创建接口压测"
log_time "=========================================="
log_time "并发: 200, 持续: 300秒 (5分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=create \
    -duration=300 \
    -concurrency=200 \
    2>&1 | tee "$TEST_DIR/benchmark_create_1b.log"

log_time "✅ 创建压测完成"
echo ""

# ==========================================
# 阶段5: 更新接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段5: 更新接口压测"
log_time "=========================================="
log_time "并发: 200, 持续: 300秒 (5分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=update \
    -duration=300 \
    -concurrency=200 \
    2>&1 | tee "$TEST_DIR/benchmark_update_1b.log"

log_time "✅ 更新压测完成"
echo ""

# ==========================================
# 阶段6: 删除接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段6: 删除接口压测"
log_time "=========================================="
log_time "并发: 100, 持续: 180秒 (3分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=delete \
    -duration=180 \
    -concurrency=100 \
    2>&1 | tee "$TEST_DIR/benchmark_delete_1b.log"

log_time "✅ 删除压测完成"
echo ""

# ==========================================
# 阶段7: 分享协作接口压测
# ==========================================
log_time "=========================================="
log_time "   阶段7: 分享协作接口压测"
log_time "=========================================="
log_time "并发: 100, 持续: 180秒 (3分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=share \
    -duration=180 \
    -concurrency=100 \
    2>&1 | tee "$TEST_DIR/benchmark_share_1b.log"

log_time "✅ 分享压测完成"
echo ""

# ==========================================
# 阶段8: 综合压测
# ==========================================
log_time "=========================================="
log_time "   阶段8: 综合压测 (所有接口)"
log_time "=========================================="
log_time "并发: 500, 持续: 600秒 (10分钟)"
log_time ""

go run cmd/tools/benchmark_api.go \
    -test=all \
    -duration=600 \
    -concurrency=500 \
    2>&1 | tee "$TEST_DIR/benchmark_all_1b.log"

log_time "✅ 综合压测完成"
echo ""

# ==========================================
# 生成最终报告
# ==========================================
log_time "=========================================="
log_time "   生成性能测试报告"
log_time "=========================================="

# 提取关键指标
extract_metrics() {
    LOG_FILE=$1
    if [ -f "$LOG_FILE" ]; then
        QPS=$(grep "QPS:" "$LOG_FILE" | tail -1 | awk '{print $2}')
        AVG_LATENCY=$(grep "Avg Latency:" "$LOG_FILE" | tail -1 | awk '{print $3}')
        SUCCESS_RATE=$(grep "Success:" "$LOG_FILE" | tail -1 | awk '{print $3}')
        echo "QPS: $QPS, Avg Latency: $AVG_LATENCY ms, Success Rate: $SUCCESS_RATE"
    else
        echo "日志文件不存在"
    fi
}

# 生成报告
cat > "$TEST_DIR/FINAL_REPORT.md" << EOF
# TodoList App - 10亿用户压力测试报告

**测试完成时间**: $(date)
**测试持续时间**: $((DATA_GEN_DURATION / 3600))小时 + 压测时间
**数据规模**: 10亿用户, 100亿Lists, 1000亿Items

---

## 环境信息

- **应用版本**: v1.0
- **数据库**: MySQL 8.0
  - User DBs: 16个 (1024张表)
  - Data DBs: 64个 (4096×3张表)
- **缓存**: Redis
- **服务器**: $(uname -a)
- **CPU**: $(nproc) 核心
- **内存**: $(free -h 2>/dev/null | grep Mem | awk '{print $2}' || echo "N/A")

---

## 数据生成性能

- **用户数**: 1,000,000,000
- **生成时间**: $((DATA_GEN_DURATION / 3600))小时 $((DATA_GEN_DURATION % 3600 / 60))分钟
- **生成速度**: $((1000000000 / DATA_GEN_DURATION)) 用户/秒
- **写入QPS**: $((1000000000 * 10 * 10 / DATA_GEN_DURATION)) 条/秒 (包含List和Item)

---

## API性能测试结果

### 1. 登录接口
$(extract_metrics "$TEST_DIR/benchmark_login_1b.log")

### 2. 查询接口
$(extract_metrics "$TEST_DIR/benchmark_query_1b.log")

### 3. 创建接口
$(extract_metrics "$TEST_DIR/benchmark_create_1b.log")

### 4. 更新接口
$(extract_metrics "$TEST_DIR/benchmark_update_1b.log")

### 5. 删除接口
$(extract_metrics "$TEST_DIR/benchmark_delete_1b.log")

### 6. 分享接口
$(extract_metrics "$TEST_DIR/benchmark_share_1b.log")

### 7. 综合测试
$(extract_metrics "$TEST_DIR/benchmark_all_1b.log")

---

## 性能目标对比

| 指标 | 设计目标 | 实际结果 | 达标情况 |
|-----|---------|---------|---------|
| 日活用户 | 100M | 测试10亿用户 | ✅ 超标 |
| 写QPS | 5,000 | 查看日志 | ⏳ |
| 读QPS | 50,000 | 查看日志 | ⏳ |
| 平均延迟 | < 100ms | 查看日志 | ⏳ |
| P99延迟 | < 500ms | 查看日志 | ⏳ |

---

## 详细日志文件

所有测试日志保存在: \`$TEST_DIR/\`

\`\`\`bash
# 查看数据生成日志
cat $TEST_DIR/data_gen_1billion.log

# 查看登录压测
cat $TEST_DIR/benchmark_login_1b.log

# 查看查询压测
cat $TEST_DIR/benchmark_query_1b.log

# 查看所有QPS
grep "QPS:" $TEST_DIR/*.log
\`\`\`

---

## 数据库统计

\`\`\`bash
# 查看用户表总行数
mysql> SELECT SUM(TABLE_ROWS) FROM information_schema.TABLES 
       WHERE TABLE_SCHEMA LIKE 'todo_user_db_%' AND TABLE_NAME LIKE 'users_%';

# 查看List表总行数
mysql> SELECT SUM(TABLE_ROWS) FROM information_schema.TABLES 
       WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' AND TABLE_NAME LIKE 'todo_lists_tab_%';

# 查看Item表总行数
mysql> SELECT SUM(TABLE_ROWS) FROM information_schema.TABLES 
       WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' AND TABLE_NAME LIKE 'todo_items_tab_%';
\`\`\`

---

## 优化建议

基于测试结果的优化建议：

1. **数据库层面**
   - 调整InnoDB缓冲池大小
   - 优化慢查询
   - 添加必要的索引
   - 考虑读写分离

2. **应用层面**
   - 调整连接池大小
   - 优化缓存策略
   - 启用查询结果缓存
   - 实现批量操作

3. **架构层面**
   - 增加数据库分片数量
   - 部署多个应用实例
   - 使用负载均衡
   - 部署Redis集群

4. **监控告警**
   - 添加Prometheus监控
   - 设置性能告警
   - 实时追踪慢查询
   - 监控缓存命中率

---

## 结论

测试完成时间: $(date)

详细性能数据请参考各个日志文件。
EOF

log_time "✅ 最终报告已生成: $TEST_DIR/FINAL_REPORT.md"
echo ""

# ==========================================
# 完成
# ==========================================
log_time "=========================================="
log_time "   🎉 所有测试完成！"
log_time "=========================================="
log_time "总用时: 从 $(cat "$TEST_DIR/data_gen_1billion.log" | head -1) 开始"
log_time "结果目录: $TEST_DIR"
log_time ""
log_time "查看报告:"
log_time "  cat $TEST_DIR/FINAL_REPORT.md"
log_time ""
log_time "查看性能数据:"
log_time "  grep 'QPS:' $TEST_DIR/*.log"
log_time "  grep 'Success:' $TEST_DIR/*.log"
log_time "=========================================="

# 发送通知（如果配置了）
if command -v osascript &> /dev/null; then
    osascript -e 'display notification "10亿用户压力测试已完成！" with title "TodoList Performance Test"'
fi

