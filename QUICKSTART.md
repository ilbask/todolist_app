# TodoList App - Quick Start Guide

**5-Minute Setup for Local Development**

---

## Prerequisites Check

Before starting, ensure you have:

```bash
# Check Go version (need 1.24+)
go version

# Check MySQL (need 8.0+)
mysql --version

# Optional: Redis & Kafka
redis-cli --version
kafka-topics.sh --version
```

---

## Step 1: Clone & Install Dependencies

```bash
cd /Users/it/debao.huang/todo-app/todolist-app

# Install Go dependencies
go mod tidy
```

---

## Step 2: Start MySQL

```bash
# macOS
brew services start mysql

# Or manually
mysql.server start

# Verify connection
mysql -u root -p
```

---

## Step 3: Initialize Sharded Databases

```bash
# Set your MySQL password
export DB_PASS="115119_hH"  # Replace with your actual password
export DB_USER="root"

# Create 16 User DBs + 64 Data DBs (takes ~2-5 minutes)
go run cmd/tools/init_sharding_v6.go

# Verify setup (optional)
go run cmd/tools/verify_sharding.go
```

**Expected Output:**
```
✅ ALL SHARDS VERIFIED SUCCESSFULLY!
   - 16 User DBs with 1024 user tables
   - 16 User DBs with 1024 index tables
   - 64 Data DBs with 4096 lists/items/collab tables each
```

---

## Step 4: Start Application

### Option A: Using start.sh (Recommended)

```bash
# Background mode (default)
./start.sh

# Foreground mode (see logs in terminal)
./start.sh -f

# Get help
./start.sh -h
```

**What start.sh does:**
- ✅ Checks prerequisites (Go, directories)
- ✅ Kills existing processes
- ✅ Checks port 8080 availability
- ✅ Builds app (or falls back to `go run` if build fails)
- ✅ Starts application with proper environment variables

### Option B: Manual Start

```bash
# Set environment variables
export DB_PASS="115119_hH"
export DB_USER="root"

# Create directories
mkdir -p log uploads

# Start app
go run cmd/api/main.go
```

**Expected Output:**
```
==========================================
   TodoList App - Startup Script
==========================================
📋 Checking prerequisites...
   ✓ Go version: go version go1.24.11 darwin/amd64
📁 Creating directories...
   ✓ log/ and uploads/ directories ready
🔧 Setting environment variables...
   ✓ DB_PASS set to default
   ✓ DB_USER set to 'root'
🔍 Checking for existing processes...
   ℹ️  No running processes found
🔌 Checking port 8080...
   ✓ Port 8080 is available
==========================================
🔨 Attempting to build application...
✅ Build successful!
🚀 Starting todo_app...
==========================================
✅ Application started successfully!
==========================================
   PID: 12345
   URL: http://localhost:8080
   Logs: tail -f log/app.log

To stop: ./stop.sh
To check status: ./status.sh
==========================================
```

---

## Step 5: Test the Application

### Web UI (Easiest)

1. Open browser: **http://localhost:8080**
2. Click "Register"
3. Enter email & password
4. **Note the verification code** (displayed in alert)
5. Click "Verify" and paste the code
6. Click "Login" with same credentials
7. Create todo lists and add items!

### API Testing (Postman)

```bash
# Import collection
# File: docs/TodoApp_Postman_Collection.json

# Set variables:
# - base_url: http://localhost:8080/api
# - user_email: test@example.com

# Run requests in order:
# 1. Register
# 2. Verify (use code from Register response)
# 3. Login (saves token automatically)
# 4. Create List
# 5. Add Item
```

### cURL Testing

```bash
# 1. Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Response includes verification code:
# {"message":"Verification code sent to email","code":"123456"}

# 2. Verify
curl -X POST http://localhost:8080/api/auth/verify \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","code":"123456"}'

# 3. Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Save the token from response

# 4. Get Lists
curl http://localhost:8080/api/lists \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

## Optional: Start Redis & Kafka

### Redis (for caching)

```bash
# macOS
brew install redis
brew services start redis

# Verify
redis-cli ping  # Should return "PONG"

# Restart app to enable caching
./start.sh
```

### Kafka (for media uploads)

```bash
# macOS
brew install kafka
brew services start zookeeper
brew services start kafka

# Create topic
kafka-topics.sh --create --topic media-uploads \
  --bootstrap-server localhost:9092 \
  --partitions 3 --replication-factor 1

# Restart app to enable Kafka
./start.sh
```

---

## Troubleshooting

### Error: "failed to connect to mysql"

```bash
# Check MySQL is running
mysql -u root -p -e "SELECT 1"

# Verify password
export DB_PASS="your_actual_password"

# Recreate databases
go run cmd/tools/cleanup_db.go
go run cmd/tools/init_sharding_v6.go
```

### Error: "port 8080 already in use"

```bash
# Find process
lsof -i :8080

# Kill it
pkill -f todo_app

# Or use start.sh which auto-kills
./start.sh
```

### Error: "package encoding/pem is not in std"

```bash
# Reinstall Go
brew reinstall go

# Or use go run instead of go build
go run cmd/api/main.go
```

### Logs not appearing

```bash
# Check log file
tail -f log/app.log

# Or watch live output
./start.sh  # Runs in foreground for debugging
```

---

## Next Steps

### 1. Explore Features

- **Sharing**: Register 2nd user, share a list
- **CAPTCHA**: Visit `http://localhost:8080/api/captcha/generate`
- **Media Upload**: Use Postman to upload an image

### 2. Run Benchmarks

```bash
# Generate test data (100K users)
go run cmd/tools/benchmark_data_gen.go -users=100000 -workers=5

# Stress test login API
go run cmd/tools/benchmark_api.go -test=login -duration=30 -concurrency=50
```

### 3. Read Documentation

- **API Reference**: [docs/API.md](docs/API.md)
- **Implementation Summary**: [docs/IMPLEMENTATION_SUMMARY.md](docs/IMPLEMENTATION_SUMMARY.md)
- **Full README**: [README.md](README.md)

---

## Quick Reference

### Application Management

```bash
# Start app (background mode)
./start.sh

# Start app (foreground mode - see logs)
./start.sh -f

# Stop app
./stop.sh

# Check status
./status.sh

# View logs
tail -f log/app.log
```

### Database Services

```bash
# Start MySQL
brew services start mysql

# Start Redis (optional)
brew services start redis

# Start Kafka (optional)
brew services start zookeeper && brew services start kafka

# Stop all
brew services stop mysql redis zookeeper kafka
```

### Database Management

```bash
# Recreate databases
go run cmd/tools/cleanup_db.go && go run cmd/tools/init_sharding_v6.go

# Verify sharding
go run cmd/tools/verify_sharding.go

# Connect to MySQL
mysql -u root -p
mysql> SHOW DATABASES LIKE 'todo_%';
mysql> USE todo_user_db_0;
mysql> SHOW TABLES;
```

### Troubleshooting Commands

```bash
# Check what's using port 8080
lsof -i :8080

# Force kill process on port 8080
kill -9 $(lsof -t -i:8080)

# Check all todo_app processes
ps aux | grep todo_app

# View recent logs
tail -n 50 log/app.log

# Follow logs in real-time
tail -f log/app.log
```

---

## Automated API Testing

Run the complete API test suite automatically:

```bash
./test_api.sh
```

This script will:
1. ✅ Check if app is running
2. ✅ Register a test user
3. ✅ Verify email with code
4. ✅ Login and get token
5. ✅ Create a todo list
6. ✅ Get all lists
7. ✅ Add an item
8. ✅ Get all items
9. ✅ Update item status
10. ✅ Test CAPTCHA generation

**Expected Output:**
```
==========================================
   TodoList App - API Test
==========================================
Testing API at: http://localhost:8080/api
✅ App is running
📝 Registration successful! Code: 123456
✉️  Verification successful!
🔑 Login successful! Token: abc123...
📋 List created! ID: 1234567890
📑 Get lists successful!
➕ Item added! ID: 9876543210
📝 Get items successful!
✏️  Item updated!
🖼️  CAPTCHA generation successful!
==========================================
   ✅ ALL TESTS PASSED!
==========================================
```

---

## Success Checklist

- [ ] MySQL is running and accessible
- [ ] Sharding databases created (16 user + 64 data DBs)
- [ ] Application starts without errors
- [ ] `./status.sh` shows app is running
- [ ] `./test_api.sh` passes all tests
- [ ] Can access web UI at http://localhost:8080
- [ ] Can register and login via web UI
- [ ] Logs appear in `log/app.log`

If all checks pass: **🎉 You're ready to go!**

---

**Need Help?**
- Check [docs/IMPLEMENTATION_SUMMARY.md](docs/IMPLEMENTATION_SUMMARY.md) for detailed info
- Review logs: `tail -f log/app.log`
- Check MySQL: `mysql -u root -p -e "SHOW DATABASES LIKE 'todo_%'"`

**Happy Coding! 🚀**

