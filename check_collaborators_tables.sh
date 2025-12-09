#!/bin/bash

# 检查 list_collaborators_tab 表

DB_PASS=${DB_PASS:-"115119_hH"}

echo "=========================================="
echo "   检查 list_collaborators_tab 表"
echo "=========================================="
echo ""

# 总计数
TOTAL=$(mysql -u root -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'list_collaborators_tab_%';
" 2>/dev/null | tail -1)

echo "总计: $TOTAL 张 list_collaborators_tab_ 表"
echo ""

# 按数据库统计
echo "各数据库表数量:"
mysql -u root -p"$DB_PASS" -e "
SELECT 
    TABLE_SCHEMA as 'Database',
    COUNT(*) as 'Tables'
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'list_collaborators_tab_%'
GROUP BY TABLE_SCHEMA 
ORDER BY TABLE_SCHEMA;
" 2>/dev/null

echo ""
echo "示例表名（前10个）:"
mysql -u root -p"$DB_PASS" -e "
SELECT 
    TABLE_SCHEMA,
    TABLE_NAME
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'list_collaborators_tab_%'
ORDER BY TABLE_SCHEMA, TABLE_NAME
LIMIT 10;
" 2>/dev/null

echo ""
echo "示例表名（最后10个）:"
mysql -u root -p"$DB_PASS" -e "
SELECT 
    TABLE_SCHEMA,
    TABLE_NAME
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'list_collaborators_tab_%'
ORDER BY TABLE_SCHEMA DESC, TABLE_NAME DESC
LIMIT 10;
" 2>/dev/null

echo ""
echo "=========================================="
echo "✅ 验证结果:"
if [ "$TOTAL" -eq 4096 ]; then
    echo "   所有 4096 张 list_collaborators_tab 表都存在！"
else
    echo "   ⚠️ 缺少 $((4096 - TOTAL)) 张表"
fi
echo "=========================================="

