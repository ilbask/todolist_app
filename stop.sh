#!/bin/bash

# TodoList App Stop Script

APP_NAME="todo_app"

echo "=========================================="
echo "   TodoList App - Stop Script"
echo "=========================================="
echo ""

STOPPED_SOMETHING=false

# Stop compiled binary
if pgrep -f "./$APP_NAME" > /dev/null; then
    echo "🛑 Stopping $APP_NAME process..."
    pkill -f "./$APP_NAME"
    STOPPED_SOMETHING=true
fi

# Stop go run process
if pgrep -f "go run cmd/api/main.go" > /dev/null; then
    echo "🛑 Stopping 'go run' process..."
    pkill -f "go run cmd/api/main.go"
    STOPPED_SOMETHING=true
fi

# Check port 8080
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "⚠️  Port 8080 is still in use. Force killing..."
    kill -9 $(lsof -t -i:8080) 2>/dev/null || true
    STOPPED_SOMETHING=true
fi

if [ "$STOPPED_SOMETHING" = true ]; then
    echo ""
    echo "✅ All TodoList app processes stopped"
else
    echo "ℹ️  No running processes found"
fi

echo "=========================================="

