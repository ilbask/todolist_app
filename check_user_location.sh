#!/bin/bash

# Check which database contains a specific user ID

USER_ID=$1

if [ -z "$USER_ID" ]; then
    echo "Usage: ./check_user_location.sh <USER_ID>"
    echo "Example: ./check_user_location.sh 388711261560508400"
    exit 1
fi

echo "=========================================="
echo "   Searching for User ID: $USER_ID"
echo "=========================================="
echo ""

# Calculate logical shard
LOGICAL_SHARD=$((USER_ID % 1024))
TABLE_NAME=$(printf "users_%04d" $LOGICAL_SHARD)

echo "Expected logical shard: $LOGICAL_SHARD"
echo "Expected table: $TABLE_NAME"
echo ""
echo "Searching in all 16 user databases..."
echo "=========================================="

FOUND=0

for i in {0..15}; do
    DB_NAME="todo_user_db_$i"
    
    # Check if user exists in this database
    RESULT=$(mysql -u root -p${DB_PASS:-115119_hH} -e "
        USE $DB_NAME;
        SELECT COUNT(*) as cnt FROM $TABLE_NAME WHERE user_id = $USER_ID;
    " 2>/dev/null | tail -1)
    
    if [ "$RESULT" -gt 0 ] 2>/dev/null; then
        echo "✅ FOUND in $DB_NAME.$TABLE_NAME"
        FOUND=1
        
        # Show the user data
        echo ""
        echo "User Data:"
        mysql -u root -p${DB_PASS:-115119_hH} -e "
            USE $DB_NAME;
            SELECT user_id, email, is_verified, created_at FROM $TABLE_NAME WHERE user_id = $USER_ID;
        " 2>/dev/null
        break
    else
        echo "   ❌ Not in $DB_NAME"
    fi
done

echo "=========================================="

if [ $FOUND -eq 0 ]; then
    echo "⚠️  User ID $USER_ID not found in any database"
    echo ""
    echo "Possible reasons:"
    echo "  1. User hasn't been registered yet"
    echo "  2. Different DB_PASS needed"
    echo "  3. Databases not initialized"
fi

echo "=========================================="

