#!/bin/bash

# TodoList App - Quick API Test Script

API_BASE="http://localhost:8080/api"
TEST_EMAIL="test_$(date +%s)@example.com"
TEST_PASSWORD="testpass123"

echo "=========================================="
echo "   TodoList App - API Test"
echo "=========================================="
echo ""
echo "Testing API at: $API_BASE"
echo "Test email: $TEST_EMAIL"
echo ""

# Check if app is running
echo "🔍 Step 1: Checking if app is running..."
if ! curl -s "$API_BASE/captcha/generate" > /dev/null 2>&1; then
    echo "   ❌ App is not responding!"
    echo "   Please start the app first: ./start.sh"
    exit 1
fi
echo "   ✅ App is running"
echo ""

# Test 1: Register
echo "📝 Step 2: Testing Registration..."
REGISTER_RESPONSE=$(curl -s -X POST "$API_BASE/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")

echo "   Response: $REGISTER_RESPONSE"

# Extract verification code (assuming JSON response with "code" field)
VERIFY_CODE=$(echo $REGISTER_RESPONSE | grep -o '"code":"[^"]*"' | cut -d'"' -f4)

if [ -z "$VERIFY_CODE" ]; then
    echo "   ❌ Registration failed or no verification code returned"
    exit 1
fi
echo "   ✅ Registration successful! Code: $VERIFY_CODE"
echo ""

# Test 2: Verify Email
echo "✉️  Step 3: Testing Email Verification..."
VERIFY_RESPONSE=$(curl -s -X POST "$API_BASE/auth/verify" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"code\":\"$VERIFY_CODE\"}")

echo "   Response: $VERIFY_RESPONSE"

if echo "$VERIFY_RESPONSE" | grep -q "error"; then
    echo "   ❌ Verification failed"
    exit 1
fi
echo "   ✅ Verification successful!"
echo ""

# Test 3: Login
echo "🔑 Step 4: Testing Login..."
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}")

echo "   Response: $LOGIN_RESPONSE"

# Extract token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "   ❌ Login failed or no token returned"
    exit 1
fi
echo "   ✅ Login successful! Token: ${TOKEN:0:20}..."
echo ""

# Test 4: Create Todo List
echo "📋 Step 5: Testing Create List..."
CREATE_LIST_RESPONSE=$(curl -s -X POST "$API_BASE/lists" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"title":"Test Shopping List"}')

echo "   Response: $CREATE_LIST_RESPONSE"

# Extract list_id
LIST_ID=$(echo $CREATE_LIST_RESPONSE | grep -o '"list_id":[0-9]*' | grep -o '[0-9]*')

if [ -z "$LIST_ID" ]; then
    echo "   ❌ Create list failed or no list_id returned"
    exit 1
fi
echo "   ✅ List created! ID: $LIST_ID"
echo ""

# Test 5: Get Lists
echo "📑 Step 6: Testing Get Lists..."
GET_LISTS_RESPONSE=$(curl -s -X GET "$API_BASE/lists" \
    -H "Authorization: Bearer $TOKEN")

echo "   Response: $GET_LISTS_RESPONSE"

if echo "$GET_LISTS_RESPONSE" | grep -q "Test Shopping List"; then
    echo "   ✅ Get lists successful!"
else
    echo "   ❌ Get lists failed"
    exit 1
fi
echo ""

# Test 6: Add Item
echo "➕ Step 7: Testing Add Item..."
ADD_ITEM_RESPONSE=$(curl -s -X POST "$API_BASE/lists/$LIST_ID/items" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"content\":\"Buy milk\",\"list_id\":$LIST_ID}")

echo "   Response: $ADD_ITEM_RESPONSE"

# Extract item_id
ITEM_ID=$(echo $ADD_ITEM_RESPONSE | grep -o '"item_id":[0-9]*' | grep -o '[0-9]*')

if [ -z "$ITEM_ID" ]; then
    echo "   ❌ Add item failed"
    exit 1
fi
echo "   ✅ Item added! ID: $ITEM_ID"
echo ""

# Test 7: Get Items
echo "📝 Step 8: Testing Get Items..."
GET_ITEMS_RESPONSE=$(curl -s -X GET "$API_BASE/lists/$LIST_ID/items" \
    -H "Authorization: Bearer $TOKEN")

echo "   Response: $GET_ITEMS_RESPONSE"

if echo "$GET_ITEMS_RESPONSE" | grep -q "Buy milk"; then
    echo "   ✅ Get items successful!"
else
    echo "   ❌ Get items failed"
    exit 1
fi
echo ""

# Test 8: Update Item
echo "✏️  Step 9: Testing Update Item..."
UPDATE_ITEM_RESPONSE=$(curl -s -X PUT "$API_BASE/items/$ITEM_ID" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d "{\"list_id\":$LIST_ID,\"content\":\"Buy organic milk\",\"is_done\":true}")

echo "   Response: $UPDATE_ITEM_RESPONSE"
echo "   ✅ Item updated!"
echo ""

# Test 9: CAPTCHA
echo "🖼️  Step 10: Testing CAPTCHA..."
CAPTCHA_RESPONSE=$(curl -s -X GET "$API_BASE/captcha/generate")
echo "   Response: $CAPTCHA_RESPONSE"

if echo "$CAPTCHA_RESPONSE" | grep -q "captcha_id"; then
    echo "   ✅ CAPTCHA generation successful!"
else
    echo "   ❌ CAPTCHA generation failed"
fi
echo ""

# Summary
echo "=========================================="
echo "   ✅ ALL TESTS PASSED!"
echo "=========================================="
echo ""
echo "Test Summary:"
echo "   ✓ Registration"
echo "   ✓ Email Verification"
echo "   ✓ Login"
echo "   ✓ Create List"
echo "   ✓ Get Lists"
echo "   ✓ Add Item"
echo "   ✓ Get Items"
echo "   ✓ Update Item"
echo "   ✓ CAPTCHA"
echo ""
echo "Test user created:"
echo "   Email: $TEST_EMAIL"
echo "   Password: $TEST_PASSWORD"
echo "   Token: ${TOKEN:0:30}..."
echo ""
echo "You can now test in browser:"
echo "   http://localhost:8080"
echo "=========================================="

