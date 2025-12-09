#!/bin/bash

# 快速检查分库分表配置

DB_PASS=${DB_PASS:-"115119_hH"}
DB_USER=${DB_USER:-"root"}

echo "=========================================="
echo "   分库分表快速检查"
echo "=========================================="
echo ""

# 1. 检查数据库数量
echo "📊 数据库数量:"
USER_DBS=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.SCHEMATA 
WHERE SCHEMA_NAME LIKE 'todo_user_db_%';
" 2>/dev/null | tail -1)

DATA_DBS=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.SCHEMATA 
WHERE SCHEMA_NAME LIKE 'todo_data_db_%';
" 2>/dev/null | tail -1)

echo "  User DBs: $USER_DBS (预期: 16)"
echo "  Data DBs: $DATA_DBS (预期: 64)"
echo ""

# 2. 检查表数量
echo "📊 表数量统计:"

USERS_TABLES=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_user_db_%' 
AND TABLE_NAME LIKE 'users_%';
" 2>/dev/null | tail -1)

INDEX_TABLES=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_user_db_%' 
AND TABLE_NAME LIKE 'user_list_index_%';
" 2>/dev/null | tail -1)

LISTS_TABLES=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'todo_lists_tab_%';
" 2>/dev/null | tail -1)

ITEMS_TABLES=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'todo_items_tab_%';
" 2>/dev/null | tail -1)

COLLAB_TABLES=$(mysql -u "$DB_USER" -p"$DB_PASS" -e "
SELECT COUNT(*) 
FROM information_schema.TABLES 
WHERE TABLE_SCHEMA LIKE 'todo_data_db_%' 
AND TABLE_NAME LIKE 'list_collaborators_tab_%';
" 2>/dev/null | tail -1)

# 打印结果
check_count() {
    NAME=$1
    EXPECTED=$2
    ACTUAL=$3
    
    if [ "$ACTUAL" -eq "$EXPECTED" ]; then
        echo "  ✅ $NAME: $ACTUAL (符合预期)"
    else
        echo "  ❌ $NAME: $ACTUAL (预期: $EXPECTED, 差异: $((ACTUAL - EXPECTED)))"
    fi
}

check_count "users_ 表         " 1024 "$USERS_TABLES"
check_count "user_list_index_ 表" 1024 "$INDEX_TABLES"
check_count "todo_lists_tab_ 表  " 4096 "$LISTS_TABLES"
check_count "todo_items_tab_ 表  " 4096 "$ITEMS_TABLES"
check_count "list_collaborators_ 表" 4096 "$COLLAB_TABLES"

TOTAL_TABLES=$((USERS_TABLES + INDEX_TABLES + LISTS_TABLES + ITEMS_TABLES + COLLAB_TABLES))
echo ""
echo "  总计: $TOTAL_TABLES 张表 (预期: 14336)"

echo ""
echo "=========================================="

# 判断是否全部通过
if [ "$USER_DBS" -eq 16 ] && \
   [ "$DATA_DBS" -eq 64 ] && \
   [ "$USERS_TABLES" -eq 1024 ] && \
   [ "$INDEX_TABLES" -eq 1024 ] && \
   [ "$LISTS_TABLES" -eq 4096 ] && \
   [ "$ITEMS_TABLES" -eq 4096 ] && \
   [ "$COLLAB_TABLES" -eq 4096 ]; then
    echo "✅ 所有检查通过！"
    echo ""
    echo "分片架构已就绪:"
    echo "  • 支持 10亿+ 用户"
    echo "  • 支持 100亿+ Todo Lists"
    echo "  • 支持 1000亿+ Todo Items"
    echo "  • 一致性哈希路由"
    echo "  • 水平扩展能力"
else
    echo "❌ 检查失败！请运行:"
    echo "   go run cmd/tools/fix_missing_tables.go"
fi

echo "=========================================="

