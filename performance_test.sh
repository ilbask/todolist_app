#!/bin/bash

# TodoList App - 性能测试脚本
# 渐进式压力测试：从小规模到10亿用户

set -e

# 配置
DB_PASS=${DB_PASS:-"115119_hH"}
DB_USER=${DB_USER:-"root"}

echo "=========================================="
echo "   TodoList App - 性能测试"
echo "=========================================="
echo ""

# 检查应用是否运行
echo "🔍 检查应用状态..."
if ! curl -s http://localhost:8080/api/captcha/generate > /dev/null 2>&1; then
    echo "❌ 应用未运行！请先启动："
    echo "   ./start.sh"
    exit 1
fi
echo "✅ 应用正在运行"
echo ""

# 创建测试目录
mkdir -p performance_results
TEST_DIR="performance_results/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$TEST_DIR"

echo "📊 测试结果将保存到: $TEST_DIR"
echo ""

# ==========================================
# 阶段1: 小规模测试 (10万用户)
# ==========================================
echo "=========================================="
echo "   阶段1: 小规模测试 (10万用户)"
echo "=========================================="
echo ""

echo "1️⃣ 生成测试数据: 100,000 用户..."
echo "   每用户: 10个List, 每List: 10个Item"
echo "   总计: 100万个List, 1000万个Item"
echo ""

START_TIME=$(date +%s)

go run cmd/tools/benchmark_data_gen.go \
    -users=100000 \
    -lists=10 \
    -items=10 \
    -workers=20 \
    -batch=1000 \
    2>&1 | tee "$TEST_DIR/data_gen_100k.log"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo "✅ 数据生成完成！用时: ${DURATION}秒"
echo "   数据生成速度: $((100000 / DURATION)) 用户/秒"
echo ""

# ==========================================
# 阶段2: API压力测试 (基于10万用户)
# ==========================================
echo "=========================================="
echo "   阶段2: API压力测试"
echo "=========================================="
echo ""

# 2.1 登录接口压测
echo "2️⃣.1 登录接口压测 (目标: 10K QPS)"
echo "   并发: 200, 持续: 60秒"
echo ""

go run cmd/tools/benchmark_api.go \
    -test=login \
    -duration=60 \
    -concurrency=200 \
    2>&1 | tee "$TEST_DIR/benchmark_login.log"

echo ""

# 2.2 查询接口压测
echo "2️⃣.2 查询接口压测 (目标: 50K QPS)"
echo "   并发: 500, 持续: 60秒"
echo ""

go run cmd/tools/benchmark_api.go \
    -test=query \
    -duration=60 \
    -concurrency=500 \
    2>&1 | tee "$TEST_DIR/benchmark_query.log"

echo ""

# 2.3 创建接口压测
echo "2️⃣.3 创建接口压测 (目标: 5K QPS)"
echo "   并发: 100, 持续: 60秒"
echo ""

go run cmd/tools/benchmark_api.go \
    -test=create \
    -duration=60 \
    -concurrency=100 \
    2>&1 | tee "$TEST_DIR/benchmark_create.log"

echo ""

# 2.4 更新接口压测
echo "2️⃣.4 更新接口压测 (目标: 5K QPS)"
echo "   并发: 100, 持续: 60秒"
echo ""

go run cmd/tools/benchmark_api.go \
    -test=update \
    -duration=60 \
    -concurrency=100 \
    2>&1 | tee "$TEST_DIR/benchmark_update.log"

echo ""

# 2.5 分享接口压测
echo "2️⃣.5 分享接口压测"
echo "   并发: 50, 持续: 30秒"
echo ""

go run cmd/tools/benchmark_api.go \
    -test=share \
    -duration=30 \
    -concurrency=50 \
    2>&1 | tee "$TEST_DIR/benchmark_share.log"

echo ""

# ==========================================
# 阶段3: 中规模测试 (100万用户) - 可选
# ==========================================
echo "=========================================="
echo "   阶段3: 中规模测试 (100万用户) - 可选"
echo "=========================================="
echo ""

read -p "是否继续100万用户测试？(预计需要10-20分钟) (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "3️⃣ 生成测试数据: 1,000,000 用户..."
    echo ""

    go run cmd/tools/benchmark_data_gen.go \
        -users=1000000 \
        -lists=10 \
        -items=10 \
        -workers=50 \
        -batch=2000 \
        2>&1 | tee "$TEST_DIR/data_gen_1m.log"

    echo ""
    echo "✅ 100万用户数据生成完成！"
    echo ""

    # 重新运行压测
    echo "重新运行API压测（基于100万用户）..."
    
    go run cmd/tools/benchmark_api.go \
        -test=all \
        -duration=120 \
        -concurrency=500 \
        2>&1 | tee "$TEST_DIR/benchmark_1m_all.log"
else
    echo "⏭️  跳过100万用户测试"
fi

echo ""

# ==========================================
# 阶段4: 大规模测试 (10亿用户) - 超长时间
# ==========================================
echo "=========================================="
echo "   阶段4: 大规模测试 (10亿用户)"
echo "=========================================="
echo ""
echo "⚠️  警告：生成10亿用户数据需要："
echo "   - 时间: 约12-24小时"
echo "   - 磁盘: 约500GB+"
echo "   - 内存: 推荐16GB+"
echo ""

read -p "是否继续10亿用户测试？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "4️⃣ 生成测试数据: 1,000,000,000 用户..."
    echo "   启动时间: $(date)"
    echo ""

    go run cmd/tools/benchmark_data_gen.go \
        -users=1000000000 \
        -lists=10 \
        -items=10 \
        -workers=100 \
        -batch=5000 \
        2>&1 | tee "$TEST_DIR/data_gen_1b.log"

    echo ""
    echo "✅ 10亿用户数据生成完成！"
    echo "   完成时间: $(date)"
    echo ""

    # 最终压测
    echo "运行最终压测..."
    
    go run cmd/tools/benchmark_api.go \
        -test=all \
        -duration=300 \
        -concurrency=1000 \
        2>&1 | tee "$TEST_DIR/benchmark_1b_all.log"
else
    echo "⏭️  跳过10亿用户测试"
    echo ""
    echo "💡 如需后台运行10亿用户测试，使用:"
    echo "   nohup ./performance_test_1b.sh > performance.log 2>&1 &"
fi

# ==========================================
# 生成测试报告
# ==========================================
echo ""
echo "=========================================="
echo "   生成测试报告"
echo "=========================================="
echo ""

# 创建测试报告
cat > "$TEST_DIR/REPORT.md" << EOF
# TodoList App - 性能测试报告

**测试时间**: $(date)
**测试环境**: 
- 应用版本: v1.0
- 数据库: MySQL 8.0 (分片: 16 User DBs + 64 Data DBs)
- 缓存: Redis (如启用)
- 服务器配置: $(uname -m)

---

## 测试结果摘要

### 数据规模
- 用户数: 根据日志查看
- Todo Lists: 用户数 × 10
- Todo Items: 用户数 × 10 × 10

### API性能指标

查看各个 benchmark_*.log 文件获取详细数据：
- \`benchmark_login.log\` - 登录接口性能
- \`benchmark_query.log\` - 查询接口性能
- \`benchmark_create.log\` - 创建接口性能
- \`benchmark_update.log\` - 更新接口性能
- \`benchmark_share.log\` - 分享接口性能

### 性能目标对比

| 指标 | 目标 | 实际 | 达标 |
|-----|------|------|------|
| 写QPS | 5,000 | 查看日志 | ⏳ |
| 读QPS | 50,000 | 查看日志 | ⏳ |
| 平均延迟 | < 100ms | 查看日志 | ⏳ |
| P99延迟 | < 500ms | 查看日志 | ⏳ |

---

## 详细日志

所有测试日志保存在: $TEST_DIR/

查看方式：
\`\`\`bash
cd $TEST_DIR
cat benchmark_login.log | grep "QPS"
cat benchmark_query.log | grep "Avg Latency"
\`\`\`

---

## 优化建议

根据测试结果，可能的优化方向：
1. 调整数据库连接池大小
2. 优化Redis缓存策略
3. 添加更多数据库分片
4. 启用查询缓存
5. 优化慢查询

EOF

echo "✅ 测试报告已生成: $TEST_DIR/REPORT.md"
echo ""

# 显示摘要
echo "=========================================="
echo "   测试完成摘要"
echo "=========================================="
echo ""
echo "📁 测试结果目录: $TEST_DIR"
echo ""
echo "📋 生成的文件:"
ls -lh "$TEST_DIR"
echo ""
echo "📊 查看报告:"
echo "   cat $TEST_DIR/REPORT.md"
echo ""
echo "📈 分析性能数据:"
echo "   grep 'QPS:' $TEST_DIR/*.log"
echo "   grep 'Avg Latency:' $TEST_DIR/*.log"
echo "   grep 'Success:' $TEST_DIR/*.log"
echo ""
echo "=========================================="

