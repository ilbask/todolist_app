#!/bin/bash

# TodoList App Startup Script
# Usage: ./start.sh [options]
#   -f, --foreground  Run in foreground (see logs in terminal)
#   -h, --help        Show this help message

set -eo pipefail  # Exit on error, but allow pipeline failures

APP_NAME="todo_app"
RUN_IN_FOREGROUND=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--foreground)
            RUN_IN_FOREGROUND=true
            shift
            ;;
        -h|--help)
            echo "Usage: ./start.sh [options]"
            echo "Options:"
            echo "  -f, --foreground  Run in foreground (see logs in terminal)"
            echo "  -h, --help        Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

echo "=========================================="
echo "   TodoList App - Startup Script"
echo "=========================================="

# 1. Check prerequisites
echo "📋 Checking prerequisites..."

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.24+ first."
    exit 1
fi
echo "   ✓ Go version: $(go version)"

# Check MySQL connection
if ! command -v mysql &> /dev/null; then
    echo "   ⚠️  MySQL client not found (optional for this script)"
fi

# 2. Create necessary directories
echo ""
echo "📁 Creating directories..."
mkdir -p log
mkdir -p uploads
echo "   ✓ log/ and uploads/ directories ready"

# 3. Set environment variables
echo ""
echo "🔧 Setting environment variables..."

# Only set DB_PASS if not already set
if [ -z "$DB_PASS" ]; then
    export DB_PASS="115119_hH"
    echo "   ✓ DB_PASS set to default (you can override with: export DB_PASS='your_password')"
else
    echo "   ✓ DB_PASS already set (using existing value)"
fi

if [ -z "$DB_USER" ]; then
    export DB_USER="root"
    echo "   ✓ DB_USER set to 'root'"
else
    echo "   ✓ DB_USER already set: $DB_USER"
fi

if [ -z "$DB_HOST" ]; then
    export DB_HOST="127.0.0.1"
fi

# 4. Kill existing processes
echo ""
echo "🔍 Checking for existing processes..."

KILLED_PROCESSES=false

# Kill existing compiled binary
if pgrep -f "./$APP_NAME" > /dev/null; then
    echo "   🛑 Stopping existing $APP_NAME process..."
    pkill -f "./$APP_NAME" || true
    KILLED_PROCESSES=true
fi

# Kill existing "go run" processes
if pgrep -f "go run cmd/api/main.go" > /dev/null; then
    echo "   🛑 Stopping existing 'go run' process..."
    pkill -f "go run cmd/api/main.go" || true
    KILLED_PROCESSES=true
fi

if [ "$KILLED_PROCESSES" = true ]; then
    echo "   ⏳ Waiting 2 seconds for ports to release..."
    sleep 2
else
    echo "   ℹ️  No running processes found"
fi

# 5. Check if port 8080 is available
echo ""
echo "🔌 Checking port 8080..."
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1 ; then
    echo "   ⚠️  Port 8080 is still in use!"
    echo "   Run: lsof -i :8080 to see which process is using it"
    echo "   Or: kill -9 \$(lsof -t -i:8080) to force kill it"
    read -p "   Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "   ❌ Aborted. Please free port 8080 and try again."
        exit 1
    fi
else
    echo "   ✓ Port 8080 is available"
fi

# 6. Build or Run
echo ""
echo "=========================================="

# Try to build first (may fail due to Go toolchain issues)
echo "🔨 Attempting to build application..."
if go build -o $APP_NAME cmd/api/main.go 2>/dev/null; then
    echo "✅ Build successful!"
    
    # 7. Start the application
    echo ""
    echo "🚀 Starting $APP_NAME..."
    
    if [ "$RUN_IN_FOREGROUND" = true ]; then
        echo "   Running in FOREGROUND mode (Ctrl+C to stop)"
        echo "=========================================="
        echo ""
        ./$APP_NAME
    else
        echo "   Running in BACKGROUND mode"
        nohup ./$APP_NAME > /dev/null 2>&1 &
        APP_PID=$!
        
        echo ""
        echo "=========================================="
        echo "✅ Application started successfully!"
        echo "=========================================="
        echo "   PID: $APP_PID"
        echo "   URL: http://localhost:8080"
        echo "   Logs: tail -f log/app.log"
        echo ""
        echo "To stop: pkill -f $APP_NAME"
        echo "To run in foreground: ./start.sh -f"
        echo "=========================================="
    fi
else
    # Build failed, try go run instead
    echo "⚠️  Build failed (likely Go toolchain issue)"
    echo "   Falling back to 'go run' mode..."
    
    echo ""
    echo "🚀 Starting with 'go run cmd/api/main.go'..."
    
    if [ "$RUN_IN_FOREGROUND" = true ]; then
        echo "   Running in FOREGROUND mode (Ctrl+C to stop)"
        echo "=========================================="
        echo ""
        go run cmd/api/main.go
    else
        echo "   Running in BACKGROUND mode"
        nohup go run cmd/api/main.go > /dev/null 2>&1 &
        APP_PID=$!
        
        echo ""
        echo "=========================================="
        echo "✅ Application started successfully!"
        echo "=========================================="
        echo "   PID: $APP_PID"
        echo "   URL: http://localhost:8080"
        echo "   Logs: tail -f log/app.log"
        echo ""
        echo "To stop: pkill -f 'go run cmd/api/main.go'"
        echo "To run in foreground: ./start.sh -f"
        echo "=========================================="
    fi
fi
