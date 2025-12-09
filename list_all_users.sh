#!/bin/bash

# List all users in the system

echo "=========================================="
echo "   All Registered Users"
echo "=========================================="
echo ""

TOTAL_USERS=0

for i in {0..15}; do
    DB_NAME="todo_user_db_$i"
    
    echo "🔍 Checking $DB_NAME..."
    
    for j in {0..63}; do
        TABLE_NAME=$(printf "users_%04d" $((i * 64 + j)))
        
        # Count users in this table
        COUNT=$(mysql -u root -p${DB_PASS:-115119_hH} -e "
            USE $DB_NAME;
            SELECT COUNT(*) as cnt FROM $TABLE_NAME;
        " 2>/dev/null | tail -1)
        
        if [ "$COUNT" -gt 0 ] 2>/dev/null; then
            echo "   ✅ $TABLE_NAME: $COUNT user(s)"
            TOTAL_USERS=$((TOTAL_USERS + COUNT))
            
            # Show user details
            mysql -u root -p${DB_PASS:-115119_hH} -e "
                USE $DB_NAME;
                SELECT user_id, email, is_verified, created_at FROM $TABLE_NAME LIMIT 10;
            " 2>/dev/null
        fi
    done
done

echo ""
echo "=========================================="
echo "   Total Users: $TOTAL_USERS"
echo "=========================================="

