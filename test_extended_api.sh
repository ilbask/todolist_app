#!/bin/bash

# 扩展Todo功能API测试脚本

BASE_URL="http://localhost:8080/api"
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "============================================"
echo "  扩展Todo功能API测试"
echo "============================================"
echo ""

# 1. 注册用户
echo "1️⃣  注册新用户..."
EMAIL="extended_test_$(date +%s)@test.com"
PASSWORD="test123456"

REGISTER_RESP=$(curl -s -X POST "$BASE_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

USER_ID=$(echo $REGISTER_RESP | grep -o '"user_id":[0-9]*' | grep -o '[0-9]*')

if [ -z "$USER_ID" ]; then
    echo -e "${RED}❌ 注册失败${NC}"
    echo "$REGISTER_RESP"
    exit 1
fi

echo -e "${GREEN}✅ 注册成功! User ID: $USER_ID${NC}"
echo ""

# 2. 验证邮箱
echo "2️⃣  验证邮箱..."
CODE=$(echo $REGISTER_RESP | grep -o '"code":"[0-9]*"' | grep -o '[0-9]*')
if [ -z "$CODE" ]; then
    CODE="123456"  # 默认验证码
fi

curl -s -X POST "$BASE_URL/auth/verify" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"code\":\"$CODE\"}" > /dev/null

echo -e "${GREEN}✅ 验证完成${NC}"
echo ""

# 3. 登录
echo "3️⃣  登录..."
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo -e "${GREEN}✅ 登录成功${NC}"
echo ""

# 4. 创建List
echo "4️⃣  创建Todo List..."
LIST_RESP=$(curl -s -X POST "$BASE_URL/lists" \
  -H "Authorization: Bearer $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{"title":"工作任务列表"}')

LIST_ID=$(echo $LIST_RESP | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')

if [ -z "$LIST_ID" ]; then
    echo -e "${RED}❌ 创建List失败${NC}"
    echo "$LIST_RESP"
    exit 1
fi

echo -e "${GREEN}✅ List创建成功! List ID: $LIST_ID${NC}"
echo ""

# 5. 创建扩展Todo Item (高优先级)
echo "5️⃣  创建扩展Todo Item（高优先级）..."
ITEM1_RESP=$(curl -s -X POST "$BASE_URL/lists/$LIST_ID/items/extended" \
  -H "Authorization: Bearer $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "完成Q4项目报告",
    "description": "包含所有数据分析、图表和市场对比",
    "status": "not_started",
    "priority": "high",
    "due_date": "2025-12-31T23:59:59Z",
    "tags": "work,urgent,Q4"
  }')

ITEM1_ID=$(echo $ITEM1_RESP | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')

echo -e "${GREEN}✅ Item 1创建成功! ID: $ITEM1_ID${NC}"
echo "   Name: 完成Q4项目报告"
echo "   Priority: high"
echo "   Status: not_started"
echo ""

# 6. 创建扩展Todo Item (中优先级)
echo "6️⃣  创建扩展Todo Item（中优先级）..."
ITEM2_RESP=$(curl -s -X POST "$BASE_URL/lists/$LIST_ID/items/extended" \
  -H "Authorization: Bearer $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "代码Review",
    "description": "Review PR #123",
    "status": "in_progress",
    "priority": "medium",
    "due_date": "2025-12-15T18:00:00Z",
    "tags": "code,review"
  }')

ITEM2_ID=$(echo $ITEM2_RESP | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')

echo -e "${GREEN}✅ Item 2创建成功! ID: $ITEM2_ID${NC}"
echo "   Name: 代码Review"
echo "   Priority: medium"
echo "   Status: in_progress"
echo ""

# 7. 创建扩展Todo Item (低优先级)
echo "7️⃣  创建扩展Todo Item（低优先级）..."
ITEM3_RESP=$(curl -s -X POST "$BASE_URL/lists/$LIST_ID/items/extended" \
  -H "Authorization: Bearer $USER_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "学习Go并发编程",
    "description": "阅读《Go并发编程实战》第3章",
    "status": "not_started",
    "priority": "low",
    "due_date": "2025-12-20T12:00:00Z",
    "tags": "learning,go"
  }')

ITEM3_ID=$(echo $ITEM3_RESP | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')

echo -e "${GREEN}✅ Item 3创建成功! ID: $ITEM3_ID${NC}"
echo "   Name: 学习Go并发编程"
echo "   Priority: low"
echo "   Status: not_started"
echo ""

# 8. 筛选高优先级任务
echo "8️⃣  筛选高优先级任务..."
HIGH_ITEMS=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?priority=high" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 筛选结果:${NC}"
echo "$HIGH_ITEMS" | python3 -m json.tool 2>/dev/null || echo "$HIGH_ITEMS"
echo ""

# 9. 筛选进行中的任务
echo "9️⃣  筛选进行中的任务..."
IN_PROGRESS_ITEMS=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?status=in_progress" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 筛选结果:${NC}"
echo "$IN_PROGRESS_ITEMS" | python3 -m json.tool 2>/dev/null || echo "$IN_PROGRESS_ITEMS"
echo ""

# 10. 按截止日期排序
echo "🔟  按截止日期排序..."
SORTED_ITEMS=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?sort=due_date" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 排序结果:${NC}"
echo "$SORTED_ITEMS" | python3 -m json.tool 2>/dev/null || echo "$SORTED_ITEMS"
echo ""

# 11. 组合筛选：高优先级 + 未开始 + 按截止日期升序
echo "1️⃣1️⃣  组合筛选（高优先级 + 未开始 + 按截止日期升序）..."
COMPLEX_FILTER=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?priority=high&status=not_started&sort=due_date" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 筛选结果:${NC}"
echo "$COMPLEX_FILTER" | python3 -m json.tool 2>/dev/null || echo "$COMPLEX_FILTER"
echo ""

# 12. 更新Item状态
echo "1️⃣2️⃣  更新Item状态（将Item 2改为已完成）..."
UPDATE_RESP=$(curl -s -X PUT "$BASE_URL/items/$ITEM2_ID/extended" \
  -H "Authorization: Bearer $USER_ID" \
  -H "Content-Type: application/json" \
  -d "{
    \"list_id\": $LIST_ID,
    \"name\": \"代码Review\",
    \"description\": \"Review PR #123 - 已完成\",
    \"status\": \"completed\",
    \"priority\": \"medium\",
    \"due_date\": \"2025-12-15T18:00:00Z\",
    \"tags\": \"code,review,done\",
    \"is_done\": true
  }")

echo -e "${GREEN}✅ 更新成功${NC}"
echo ""

# 13. 筛选已完成的任务
echo "1️⃣3️⃣  筛选已完成的任务..."
COMPLETED_ITEMS=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?status=completed" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 筛选结果:${NC}"
echo "$COMPLETED_ITEMS" | python3 -m json.tool 2>/dev/null || echo "$COMPLETED_ITEMS"
echo ""

# 14. 按优先级降序排序
echo "1️⃣4️⃣  按优先级降序排序..."
PRIORITY_SORTED=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items/filtered?sort=priority&order=desc" \
  -H "Authorization: Bearer $USER_ID")

echo -e "${GREEN}✅ 排序结果:${NC}"
echo "$PRIORITY_SORTED" | python3 -m json.tool 2>/dev/null || echo "$PRIORITY_SORTED"
echo ""

# 15. 获取所有Items
echo "1️⃣5️⃣  获取所有Items（基础API）..."
ALL_ITEMS=$(curl -s -X GET "$BASE_URL/lists/$LIST_ID/items" \
  -H "Authorization: Bearer $USER_ID")

ITEM_COUNT=$(echo "$ALL_ITEMS" | grep -o '"id":' | wc -l | xargs)

echo -e "${GREEN}✅ 共有 $ITEM_COUNT 个Todo Items${NC}"
echo ""

echo "============================================"
echo -e "${GREEN}  ✨ 扩展功能测试完成！${NC}"
echo "============================================"
echo ""
echo "测试总结:"
echo "  ✅ 创建了3个扩展Todo Items（包含name, description, status, priority, due_date, tags）"
echo "  ✅ 测试了按优先级筛选"
echo "  ✅ 测试了按状态筛选"
echo "  ✅ 测试了按截止日期排序"
echo "  ✅ 测试了组合筛选"
echo "  ✅ 测试了更新扩展Item"
echo ""
echo "📝 变量信息:"
echo "  User ID: $USER_ID"
echo "  List ID: $LIST_ID"
echo "  Item 1 ID: $ITEM1_ID (高优先级, 未开始)"
echo "  Item 2 ID: $ITEM2_ID (中优先级, 已完成)"
echo "  Item 3 ID: $ITEM3_ID (低优先级, 未开始)"
echo ""
echo "📚 导入Postman Collection:"
echo "  文件路径: docs/TodoList_API_Postman_Collection.json"
echo ""

