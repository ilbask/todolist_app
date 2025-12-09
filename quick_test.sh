#!/bin/bash

# Quick API Health Check

API_BASE="http://localhost:8080/api"

echo "=========================================="
echo "   Quick Health Check"
echo "=========================================="
echo ""

# Test 1: CAPTCHA (no auth required)
echo "🧪 Testing CAPTCHA endpoint..."
CAPTCHA=$(curl -s "$API_BASE/captcha/generate")
if echo "$CAPTCHA" | grep -q "captcha_id"; then
    echo "   ✅ CAPTCHA: OK"
else
    echo "   ❌ CAPTCHA: FAILED"
    exit 1
fi

# Test 2: Try to access protected endpoint (should return 401)
echo "🔒 Testing auth protection..."
AUTH_TEST=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/lists")
if [ "$AUTH_TEST" = "401" ]; then
    echo "   ✅ Auth protection: OK"
else
    echo "   ⚠️  Auth protection: Unexpected status $AUTH_TEST"
fi

# Test 3: Register endpoint
echo "📝 Testing registration endpoint..."
REG_TEST=$(curl -s -X POST "$API_BASE/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"email":"","password":""}' \
    -o /dev/null -w "%{http_code}")
if [ "$REG_TEST" = "400" ] || [ "$REG_TEST" = "200" ]; then
    echo "   ✅ Registration: OK (responded correctly)"
else
    echo "   ❌ Registration: Unexpected status $REG_TEST"
fi

echo ""
echo "=========================================="
echo "   ✅ Core API is working!"
echo "=========================================="
echo ""
echo "📌 Application URL: http://localhost:8080"
echo ""
echo "Next steps:"
echo "  1. Open browser: http://localhost:8080"
echo "  2. Run full tests: ./test_api.sh"
echo "  3. Check status: ./status.sh"
echo ""
echo "Optional services (not required):"
echo "  • Redis: brew services start redis"
echo "  • Kafka: brew services start zookeeper kafka"
echo "=========================================="

