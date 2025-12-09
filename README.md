# TodoList App - Enterprise-Grade Demo (v2.0 Extended)

A production-ready Todo application with advanced features including sharding, caching, CAPTCHA, async media processing, and **extended Todo item features** (status, priority, tags, filtering, sorting).

## Prerequisites

- **Go 1.24+**
- **MySQL 8.0+** (Local installation recommended)
- **Redis** (Optional - for caching)
- **Kafka** (Optional - for media uploads)
- **Docker** (Optional - for quick setup)

## Project Structure

This project follows Clean Architecture principles and the Standard Go Project Layout.

```text
.
├── cmd/                      # Entry points for applications
│   ├── api/                  # Main backend API server
│   │   └── main.go           # Server initialization & routing
│   ├── realtime/             # Real-time features (WebSocket, future)
│   └── tools/                # Utility tools
│       ├── benchmark_api.go           # API stress testing tool
│       ├── benchmark_data_gen.go      # Mass data generation (1B users)
│       ├── check_missing_tables.go    # Check for missing shard tables
│       ├── check_sharding_complete.go # Verify sharding completion
│       ├── cleanup_db.go              # Database cleanup utility
│   ├── find_shard_accurate/    # Locate specific shard for data
│       ├── fix_missing_tables.go      # Auto-fix missing shard tables
│       ├── init_sharding_v6.go        # Sharding initialization (v6)
│       ├── migrate_items_schema.go    # Schema migration for items
│       ├── setup_mysql.go             # Simple DB setup
│       └── verify_sharding.go         # Verify shard table counts
├── internal/                 # Private application code
│   ├── config/               # Configuration management
│   ├── domain/               # Core business entities & interfaces (Pure Go)
│   │   ├── user.go           # User entity & UserRepository interface
│   │   └── todo.go           # Todo entities & TodoRepository interface
│   ├── handler/              # HTTP Handlers (REST API layer)
│   │   ├── auth_handler.go   # Register, Verify, Login
│   │   ├── todo_handler.go   # CRUD operations for lists & items
│   │   ├── captcha_handler.go # CAPTCHA generation & verification
│   │   └── media_handler.go  # Media file uploads
│   ├── service/              # Business logic implementation
│   │   ├── auth_service.go         # User authentication & registration
│   │   ├── auth_service_test.go    # Auth service unit tests
│   │   ├── todo_service.go         # Todo CRUD & sharing logic
│   │   ├── todo_service_test.go    # Todo service unit tests
│   │   ├── cached_todo_service.go  # Redis-based caching layer
│   │   └── mocks_test.go           # Mock implementations for testing
│   ├── repository/           # Data access layer
│   │   ├── sharded_user_repo.go    # Legacy user repository
│   │   ├── sharded_user_repo_v2.go # Sharded user data access (v2)
│   │   ├── sharded_todo_repo.go    # Legacy todo repository
│   │   └── sharded_todo_repo_v2.go # Sharded todo data access (v2)
│   ├── infrastructure/       # External services & infra
│   │   ├── db.go             # Database abstraction layer
│   │   ├── mysql.go          # MySQL connection management
│   │   ├── redis.go          # Redis client wrapper
│   │   ├── kafka.go          # Kafka producer for async jobs
│   │   ├── email.go          # Email service (SMTP or mock)
│   │   ├── captcha.go        # CAPTCHA service
│   │   └── sharding/         # Sharding logic
│   │       └── router_v2.go  # Consistent hashing router (v2)
│   └── pkg/                  # Internal packages
│       ├── consistenthash/   # Consistent hashing implementation
│       │   └── ring.go       # Hash ring for sharding
│       └── uid/              # Distributed ID generation
│           └── snowflake.go  # Snowflake ID generator
├── pkg/                      # Public packages (shared utilities)
│   ├── auth/                 # Authentication utilities
│   ├── response/             # HTTP response helpers
│   └── utils/                # Common utility functions
├── web/                      # Frontend demo (HTML/JS)
│   ├── index.html            # Single-page app
│   └── app.js                # Frontend logic
├── docs/                     # Documentation
│   ├── API.md                       # Full API documentation
│   ├── EXTENDED_TODO_FEATURES.md    # Extended features documentation
│   ├── IMPLEMENTATION_SUMMARY.md    # Implementation details
│   ├── PERFORMANCE_TEST_GUIDE.md    # Performance testing guide
│   ├── TEST_PLAN.md                 # Test planning document
│   ├── TODO_EXTENDED_FEATURES.md    # Extended features roadmap
│   ├── TodoApp_Postman_Collection.json      # Postman collection (general)
│   └── TodoList_API_Postman_Collection.json # Postman collection (API)
├── scripts/                  # SQL initialization scripts
│   └── init.sql              # Initial database schema
├── configs/                  # Configuration files
├── log/                      # Application logs (auto-created)
│   ├── app.log               # Main application log
│   └── nohup.out             # Background process log
├── uploads/                  # Uploaded media files (auto-created)
├── *.sh                      # Shell scripts for management
│   ├── start.sh              # Start application
│   ├── stop.sh               # Stop application
│   ├── status.sh             # Check status
│   ├── test_api.sh           # Basic API testing
│   ├── test_extended_api.sh  # Extended features testing
│   ├── quick_test.sh         # Quick smoke test
│   ├── performance_test.sh   # Performance benchmarking
│   ├── performance_test_1b.sh # Large-scale perf test
│   ├── check_sharding.sh     # Check sharding status
│   ├── check_collaborators_tables.sh # Check collaborators tables
│   ├── check_user_location.sh # Find user shard location
│   └── list_all_users.sh     # List all registered users
├── docker-compose.yml        # Docker environment (MySQL, Redis, Kafka)
├── go.mod                    # Go module dependencies
├── go.sum                    # Go dependency checksums
├── QUICKSTART.md             # Quick start guide
├── IMPLEMENTATION_STATUS.md  # Feature implementation status
├── IMPLEMENTATION_COMPLETE.md # Completion summary
├── COMPLETION_SUMMARY.md     # Final completion report
└── README.md                 # This file
```

## Quick Start Scripts

We provide convenient shell scripts for easy management:

| Script | Purpose |
|--------|---------|
| `./start.sh` | Start the application (auto-kills old processes, checks prerequisites) |
| `./start.sh -f` | Start in foreground mode (see logs in terminal) |
| `./stop.sh` | Stop the application |
| `./status.sh` | Check application status, port, logs, database |
| `./test_api.sh` | Run automated API tests (register, login, CRUD) |
| `./test_extended_api.sh` | Test extended Todo features (status, priority, tags, filtering) |
| `./quick_test.sh` | Quick smoke test for core functionality |
| `./performance_test.sh` | Performance benchmarking tests |
| `./performance_test_1b.sh` | Large-scale performance test (1B+ records) |
| `./check_sharding.sh` | Check sharding configuration and status |
| `./check_collaborators_tables.sh` | Verify collaborators tables in all shards |
| `./check_user_location.sh` | Find which shard a specific user is in |
| `./list_all_users.sh` | List all registered users across shards |
| `go run cmd/ensure_user_tables/main.go` | Ensure all `users_*`, `user_list_index_*`, `user_email_index_*` tables (16×64) exist |
| `go run cmd/ensure_todo_tables/main.go` | Ensure all `todo_lists/items/collaborators` tables exist in every `todo_data_db_*` (64×64) |
| `go run cmd/rebuild_all_shards/main.go` | ⚠️ DROP and recreate ALL databases with correct CRC32 sharding (DESTRUCTIVE) |
| `go run cmd/retry_list_index/main.go` | Retry failed inserts into user_list_index_* from retry table |
| `REALTIME_PORT=8091 go run cmd/realtime/main.go` | Start realtime WS fanout (Redis pub/sub) |
| `go test ./...` | Run all unit tests (includes retry table tests) |

---

### Option 1: Local MySQL (Recommended)

#### 1. Install Dependencies
```bash
# macOS
brew install mysql redis kafka

# Or use Docker for Redis/Kafka
docker-compose up -d
```

#### 2. Initialize Sharded Databases

**Option A: Fresh Installation (Recommended)**
```bash
export DB_PASS="your_mysql_password"
export DB_USER="root"

# Create 16 User DBs (1024 tables) + 64 Data DBs (4096 tables each)
go run cmd/tools/init_sharding_v6.go

# Ensure each todo_user_db_* has 64 users/index tables (CRC32 sharding layout)
go run cmd/ensure_user_tables/main.go

# Ensure each todo_data_db_* has 64 list/item/collaboration tables (CRC32 layout)
go run cmd/ensure_todo_tables/main.go

# Verify setup (optional)
go run cmd/tools/verify_sharding.go
```

**Option B: Full Rebuild (⚠️ DESTRUCTIVE - Deletes ALL data)**
```bash
export DB_PASS="your_mysql_password"
export DB_USER="root"

# Drop all old databases and recreate with correct CRC32 sharding layout
# WARNING: This will delete all existing data!
go run cmd/rebuild_all_shards/main.go

# This tool will:
# 1. DROP all todo_user_db_* and todo_data_db_* databases
# 2. CREATE 16 user DBs with 64×3 tables each (users_, user_list_index_, user_email_index_)
# 3. CREATE 64 todo DBs with 64×3 tables each (todo_lists_tab_, todo_items_tab_, list_collaborators_tab_)
# 4. Verify all databases and tables exist
```

#### 3. Start Application
```bash
# Using start.sh (recommended)
chmod +x start.sh
./start.sh

# Or manually
export DB_PASS="your_password"
go run cmd/api/main.go
```

#### 4. Access Application
- **Web App**: [http://localhost:8080](http://localhost:8080)
- **API Docs**: [docs/API.md](docs/API.md)
- **Logs**: `tail -f log/app.log`

---

### Option 2: Docker Setup (Simplified)

```bash
docker-compose up -d
go run cmd/tools/setup_mysql.go  # Simple single-DB setup
go run cmd/api/main.go
```

---

## Features

### Core Functionality
- ✅ **User Authentication**: Email-based registration with verification codes
- ✅ **Todo Lists**: Create, read, update, delete (CRUD) operations
- ✅ **Todo Items**: Full CRUD with completion status tracking
- ✅ **Multi-User Collaboration**: Share lists with role-based access (Owner/Editor/Viewer)

### Advanced Features
- 🚀 **Horizontal Sharding**: 
  - 16 User DBs (1024 tables) sharded by `user_id`
  - 64 Data DBs (4096 tables) sharded by `list_id`
  - Consistent hashing for easy expansion
- 💾 **Redis Caching**: Read-Aside pattern with 5-minute TTL
- 🖼️ **CAPTCHA**: Image-based human verification
- 📤 **Media Upload**: Async S3 upload via Kafka queue
- 🔐 **Security**: Plain-text password storage (demo mode) or bcrypt hashing
- 📊 **Snowflake IDs**: Distributed unique ID generation

### Architecture Highlights
- **Clean Architecture**: Separation of concerns (Domain → Service → Repository → Handler)
- **SOLID Principles**: Dependency injection, interface-driven design
- **Scalability**: Supports 100M DAU, 5K WQPS, 50K RQPS
- **Observability**: Structured logging to file and console

---

## Performance & Scale

### Target Metrics
- **Daily Active Users**: 100,000,000
- **Write QPS (WQPS)**: 5,000
- **Read QPS (RQPS)**: 50,000

### Benchmark Tools

#### 1. Generate Test Data
Create massive datasets for stress testing:
```bash
# Generate 1M users, 10 lists/user, 10 items/list (100M total items)
go run cmd/tools/benchmark_data_gen.go \
  -users=1000000 \
  -lists=10 \
  -items=10 \
  -workers=10 \
  -batch=1000

# For 1B users (requires ~12 hours and 500GB+ disk):
go run cmd/tools/benchmark_data_gen.go -users=1000000000 -workers=50

# Or use the shell script wrapper:
./performance_test.sh      # Standard performance test
./performance_test_1b.sh   # Large-scale test (1B+ records)
```

#### 2. API Stress Testing
```bash
# Test all endpoints for 60 seconds with 100 concurrent users
go run cmd/tools/benchmark_api.go -test=all -duration=60 -concurrency=100

# Test specific endpoint
go run cmd/tools/benchmark_api.go -test=login -duration=30 -concurrency=200

# Available test types: register, login, query, create, update, delete, share, all
```

#### 3. Sharding Management Tools
```bash
# Initialize sharding (16 User DBs + 64 Data DBs)
go run cmd/tools/init_sharding_v6.go

# Verify all shard tables exist
go run cmd/tools/verify_sharding.go

# Check for missing tables
go run cmd/tools/check_missing_tables.go

# Auto-fix missing tables
go run cmd/tools/fix_missing_tables.go

# Check sharding completion status
go run cmd/tools/check_sharding_complete.go

# Find which shard contains specific data
go run cmd/find_shard_accurate

# Migrate items schema (for upgrades)
go run cmd/tools/migrate_items_schema.go

# Cleanup all databases (⚠️  DANGEROUS - deletes all data)
go run cmd/tools/cleanup_db.go
```

---

## API Documentation

See [docs/API.md](docs/API.md) for complete API reference.

**Quick API Test Flow:**
1. **POST** `/api/auth/register` - Register user, receive verification code
2. **POST** `/api/auth/verify` - Verify email with code
3. **POST** `/api/auth/login` - Login, receive token
4. **GET** `/api/lists` - Get user's todo lists (requires `Authorization: Bearer {token}`)
5. **POST** `/api/lists` - Create new list
6. **POST** `/api/lists/{id}/items` - Add item to list
7. **POST** `/api/lists/{id}/share` - Share list with another user

**Postman Collection**: [docs/TodoList_API_Postman_Collection.json](docs/TodoList_API_Postman_Collection.json)

---

## 🆕 Extended Todo Features (v2.0)

### New Todo Item Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `name` | string | Task name | "Complete Project Report" |
| `description` | string | Detailed description | "Include Q4 data analysis..." |
| `due_date` | timestamp | Deadline | "2025-12-31T23:59:59Z" |
| `status` | enum | Current status | `not_started`, `in_progress`, `completed` |
| `priority` | enum | Priority level | `high`, `medium`, `low` |
| `tags` | string | Comma-separated tags | "work,urgent,Q4" |

### Extended API Endpoints

```http
# Create Extended Item (with all fields)
POST /api/lists/{id}/items/extended
{
  "name": "Complete Q4 Report",
  "description": "Include all data analysis",
  "status": "not_started",
  "priority": "high",
  "due_date": "2025-12-31T23:59:59Z",
  "tags": "work,urgent,Q4"
}

# Update Extended Item
PUT /api/items/{id}/extended
{
  "list_id": 123,
  "name": "Complete Q4 Report [Revised]",
  "status": "in_progress",
  ...
}

# Filter & Sort Items
GET /api/lists/{id}/items/filtered?priority=high&status=in_progress&sort=due_date&order=desc
```

### Filtering Options

- **By Status**: `?status=in_progress`
- **By Priority**: `?priority=high`
- **By Date Range**: `?due_before=2025-12-31&due_after=2025-01-01`
- **By Tags**: `?tags=work&tags=urgent`

### Sorting Options

- **Fields**: `due_date`, `priority`, `status`, `name`, `created_at`
- **Order**: `?sort=due_date` (ASC) or `?sort=due_date&order=desc` (DESC)

### Documentation

- 📖 **Full Feature Docs**: [docs/EXTENDED_TODO_FEATURES.md](docs/EXTENDED_TODO_FEATURES.md)
- 📊 **Implementation Status**: [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md)
- 🎯 **Completion Summary**: [COMPLETION_SUMMARY.md](COMPLETION_SUMMARY.md)

### Testing Extended Features

```bash
# Run extended features test suite
./test_extended_api.sh
```

---

## Configuration

### Environment Variables

```bash
# Database (Required)
export DB_USER="root"
export DB_PASS="your_mysql_password"
export DB_HOST="127.0.0.1"

# Redis (Optional - caching disabled if unavailable)
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD=""

# Kafka (Optional - async jobs disabled if unavailable)
export KAFKA_BROKERS="localhost:9092"
export KAFKA_MEDIA_TOPIC="media-uploads"

# Email (Optional - mock mode if not configured)
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your_email@gmail.com"
export SMTP_PASS="your_app_password"
export SMTP_FROM="noreply@todoapp.com"

# Media Storage (Optional)
export UPLOAD_DIR="./uploads"
export S3_BUCKET="your-s3-bucket-name"
```

---

## Testing

### Unit Tests
```bash
# Run all tests
go test -v ./...

# Test specific package
go test -v ./internal/service/...
go test -v ./internal/repository/...

# With coverage
go test -cover ./...
```

### Manual Testing
```bash
# Register user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"pass123"}'

# Get lists (replace TOKEN)
curl http://localhost:8080/api/lists \
  -H "Authorization: Bearer TOKEN"
```

---

## Troubleshooting

### MySQL Connection Errors
```bash
# Check MySQL is running
mysql -u root -p -e "SELECT 1"

# Verify databases exist
mysql -u root -p -e "SHOW DATABASES LIKE 'todo_%'"

# Check sharding status
./check_sharding.sh

# Verify all tables exist
go run cmd/tools/verify_sharding.go

# Check for missing tables
go run cmd/tools/check_missing_tables.go

# Auto-fix missing tables
go run cmd/tools/fix_missing_tables.go

# Recreate databases (⚠️  DANGEROUS - deletes all data)
go run cmd/tools/cleanup_db.go
go run cmd/tools/init_sharding_v6.go
```

### Sharding Issues
```bash
# Find which shard contains a specific user
./check_user_location.sh <user_id>

# Or use the Go tool
go run cmd/find_shard_accurate

# Check if all collaborators tables exist
./check_collaborators_tables.sh

# Verify sharding is complete
go run cmd/tools/check_sharding_complete.go
```

### Build Errors
```bash
# Update dependencies
go mod tidy

# Clear cache and rebuild
go clean -cache
go build -o todo_app cmd/api/main.go
```

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill existing todo_app
pkill -f todo_app

# Or use start.sh which auto-kills old processes
./start.sh
```

### Data Issues
```bash
# List all registered users
./list_all_users.sh

# Quick smoke test
./quick_test.sh

# Full API test suite
./test_api.sh

# Test extended features
./test_extended_api.sh
```

---

## Production Considerations

### Security Enhancements
- [ ] Enable bcrypt password hashing (currently plain-text for demo)
- [ ] Add JWT token generation/validation
- [ ] Implement rate limiting middleware
- [ ] Enable HTTPS/TLS
- [ ] Add SQL injection protection (use prepared statements)

### Scalability Improvements
- [ ] Add connection pooling tuning
- [ ] Implement circuit breakers for external services
- [ ] Add monitoring (Prometheus/Grafana)
- [ ] Set up read replicas for MySQL
- [ ] Deploy Kafka consumer workers for media processing

### Operational
- [ ] Add health check endpoints (`/health`, `/ready`)
- [ ] Implement graceful shutdown
- [  ] Set up log rotation
- [ ] Add distributed tracing (OpenTelemetry)

---

## License

MIT License - Free for educational and commercial use.

---

## Contributing

Contributions welcome! Please follow:
1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

---

## Support

For issues or questions:
- 📧 Email: support@todoapp.example
- 📝 GitHub Issues: [Create an issue](https://github.com/yourrepo/todolist-app/issues)
- 📚 API Docs: [docs/API.md](docs/API.md)
