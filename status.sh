#!/bin/bash

# TodoList App Status Check Script

APP_NAME="todo_app"

echo "=========================================="
echo "   TodoList App - Status Check"
echo "=========================================="
echo ""

# Check if process is running
echo "📊 Process Status:"
if pgrep -f "./$APP_NAME" > /dev/null; then
    PID=$(pgrep -f "./$APP_NAME")
    echo "   ✅ $APP_NAME is running (PID: $PID)"
elif pgrep -f "go run cmd/api/main.go" > /dev/null; then
    PID=$(pgrep -f "go run cmd/api/main.go")
    echo "   ✅ App is running via 'go run' (PID: $PID)"
else
    echo "   ❌ App is NOT running"
fi

echo ""

# Check port 8080
echo "🔌 Port Status:"
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    PORT_PID=$(lsof -t -i:8080)
    PORT_PROCESS=$(ps -p $PORT_PID -o comm= 2>/dev/null || echo "unknown")
    echo "   ✅ Port 8080 is in use by PID $PORT_PID ($PORT_PROCESS)"
else
    echo "   ❌ Port 8080 is FREE"
fi

echo ""

# Check if API is responding
echo "🌐 API Health:"
if curl -s http://localhost:8080/api/captcha/generate > /dev/null 2>&1; then
    echo "   ✅ API is responding"
    echo "   URL: http://localhost:8080"
else
    echo "   ❌ API is NOT responding"
fi

echo ""

# Check logs
echo "📜 Recent Logs (last 5 lines):"
if [ -f "log/app.log" ]; then
    echo "   ---"
    tail -n 5 log/app.log | sed 's/^/   /'
    echo "   ---"
    echo "   Full logs: tail -f log/app.log"
else
    echo "   ⚠️  No log file found (log/app.log)"
fi

echo ""

# Check database connectivity
echo "🗄️  Database Status:"
if [ -n "$DB_PASS" ]; then
    if mysql -u "${DB_USER:-root}" -p"$DB_PASS" -e "SELECT 1" > /dev/null 2>&1; then
        DB_COUNT=$(mysql -u "${DB_USER:-root}" -p"$DB_PASS" -e "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'todo_%'" 2>/dev/null | tail -1)
        echo "   ✅ MySQL is accessible"
        echo "   Databases: $DB_COUNT todo_* databases found"
    else
        echo "   ❌ Cannot connect to MySQL"
    fi
else
    echo "   ⚠️  DB_PASS not set (cannot test connection)"
fi

echo ""
echo "=========================================="
echo "Commands:"
echo "  Start:   ./start.sh"
echo "  Stop:    ./stop.sh"
echo "  Logs:    tail -f log/app.log"
echo "  Test:    curl http://localhost:8080"
echo "=========================================="

