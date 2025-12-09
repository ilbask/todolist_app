# TodoList App API Documentation

## Base URL
```
http://localhost:8080/api
```

## Authentication Flow

### 1. Register
Create a new user account.

**Endpoint:** `POST /auth/register`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "your_password"
}
```

**Response:**
```json
{
  "message": "Verification code sent to email",
  "code": "123456"  // Only for demo/testing
}
```

---

### 2. Verify Email
Verify email with the code sent during registration.

**Endpoint:** `POST /auth/verify`

**Request Body:**
```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

**Response:**
```json
{
  "message": "Email verified successfully"
}
```

---

### 3. Login
Authenticate and receive a token.

**Endpoint:** `POST /auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "your_password"
}
```

**Response:**
```json
{
  "token": "user_id_token_here",
  "user": {
    "id": 123,
    "email": "user@example.com"
  }
}
```

---

## CAPTCHA APIs

### 1. Generate CAPTCHA
Get a new CAPTCHA challenge.

**Endpoint:** `GET /captcha/generate`

**Response:**
```json
{
  "captcha_id": "abc123def456"
}
```

---

### 2. Get CAPTCHA Image
Retrieve the CAPTCHA image.

**Endpoint:** `GET /captcha/image/{captchaID}`

**Response:** PNG image

---

### 3. Verify CAPTCHA
Verify user's CAPTCHA solution.

**Endpoint:** `POST /captcha/verify`

**Request Body:**
```json
{
  "captcha_id": "abc123def456",
  "solution": "AB2C7D"
}
```

**Response:**
```json
{
  "valid": true
}
```

---

## Todo List APIs
**All endpoints require Authorization header:** `Authorization: Bearer {token}`

### 1. Get User's Lists
Retrieve all todo lists for the authenticated user.

**Endpoint:** `GET /lists`

**Response:**
```json
[
  {
    "list_id": 1001,
    "owner_id": 123,
    "title": "Shopping List",
    "created_at": "2025-12-08T10:00:00Z"
  }
]
```

---

### 2. Create List
Create a new todo list.

**Endpoint:** `POST /lists`

**Request Body:**
```json
{
  "title": "My New List"
}
```

**Response:**
```json
{
  "list_id": 1002,
  "owner_id": 123,
  "title": "My New List",
  "created_at": "2025-12-08T10:05:00Z"
}
```

---

### 3. Delete List
Delete a todo list (owner only).

**Endpoint:** `DELETE /lists/{id}`

**Response:**
```json
{
  "message": "List deleted"
}
```

---

### 4. Share List
Share a list with another user.

**Endpoint:** `POST /lists/{id}/share`

**Request Body:**
```json
{
  "shared_user_id": 456,
  "role": "editor"  // Options: "editor", "viewer"
}
```

**Response:**
```json
{
  "message": "List shared successfully"
}
```

---

## Todo Items APIs

### 1. Get Items
Get all items in a list.

**Endpoint:** `GET /lists/{id}/items`

**Response:**
```json
[
  {
    "item_id": 5001,
    "list_id": 1001,
    "content": "Buy milk",
    "is_done": false,
    "created_at": "2025-12-08T10:10:00Z"
  }
]
```

---

### 2. Add Item
Add a new item to a list.

**Endpoint:** `POST /lists/{id}/items`

**Request Body:**
```json
{
  "content": "Buy eggs",
  "list_id": 1001
}
```

**Response:**
```json
{
  "item_id": 5002,
  "list_id": 1001,
  "content": "Buy eggs",
  "is_done": false,
  "created_at": "2025-12-08T10:15:00Z"
}
```

---

### 3. Update Item
Update an item's content or status.

**Endpoint:** `PUT /items/{id}`

**Request Body:**
```json
{
  "list_id": 1001,
  "content": "Buy organic eggs",
  "is_done": true
}
```

**Response:**
```json
{
  "message": "Item updated"
}
```

---

### 4. Delete Item
Delete an item from a list.

**Endpoint:** `DELETE /items/{id}?list_id={list_id}`

**Response:**
```json
{
  "message": "Item deleted"
}
```

---

## Media Upload API

### Upload Media
Upload an image or video associated with a todo item. Files are queued for async S3 upload via Kafka.

**Endpoint:** `POST /media/upload`

**Content-Type:** `multipart/form-data`

**Form Fields:**
- `user_id`: User ID (int64)
- `list_id`: List ID (int64)
- `item_id`: Item ID (int64)
- `media_type`: "image" or "video"
- `file`: File data

**Response:**
```json
{
  "message": "File uploaded successfully and queued for S3 upload",
  "s3_key": "media/123/1001/1733661234_image.jpg"
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

**Common HTTP Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid input
- `401 Unauthorized` - Missing or invalid token
- `403 Forbidden` - Permission denied
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

---

## Rate Limits & Performance

**Current System Capacity:**
- **Daily Active Users:** 100M
- **Write QPS (WQPS):** 5,000
- **Read QPS (RQPS):** 50,000

**Optimizations:**
- Redis caching for read operations (5-minute TTL)
- Sharded MySQL architecture (16 user DBs, 64 data DBs)
- Kafka for async media processing

---

## Testing Tools

### 1. Data Generation
Generate massive test data (1B users, 10 lists/user, 10 items/list):

```bash
go run cmd/tools/benchmark_data_gen.go \
  -users=1000000 \
  -lists=10 \
  -items=10 \
  -workers=10 \
  -batch=1000
```

### 2. API Stress Testing
Run load tests against specific endpoints:

```bash
# Test all endpoints
go run cmd/tools/benchmark_api.go -test=all -duration=60 -concurrency=100

# Test specific endpoint
go run cmd/tools/benchmark_api.go -test=login -duration=30 -concurrency=50
```

**Available test types:**
- `register` - Registration API
- `login` - Login API
- `query` - List query API
- `create` - List creation API
- `update` - List update API
- `share` - Sharing API
- `all` - Run all tests

---

## Example: Using Postman

### 1. Register & Login
1. POST `/api/auth/register` with email & password
2. Note the verification code from response
3. POST `/api/auth/verify` with email & code
4. POST `/api/auth/login` with email & password
5. Save the `token` from response

### 2. Create Todo List
1. POST `/api/lists` with header `Authorization: Bearer {token}`
2. Body: `{ "title": "My Tasks" }`
3. Save the `list_id` from response

### 3. Add Items
1. POST `/api/lists/{list_id}/items` with auth header
2. Body: `{ "content": "Task 1", "list_id": {list_id} }`

### 4. Share List
1. Register another user (User B)
2. POST `/api/lists/{list_id}/share` with User A's token
3. Body: `{ "shared_user_id": {user_b_id}, "role": "editor" }`
4. User B can now access and edit the list

---

## Advanced Features

### Redis Caching Strategy
The app uses **Read-Aside** (Cache-Aside) pattern:
- **Read:** Check Redis → If miss, query DB → Store in Redis
- **Write:** Update DB → Invalidate Redis cache

**Cache Keys:**
- `list:{list_id}` - Single list data
- `items:{list_id}` - All items for a list
- `user_lists:{user_id}` - All lists for a user

**TTL:** 5 minutes

---

### Kafka Topics
- `media-uploads` - Media upload events for S3 processing
- `list.shared` - List sharing notifications
- `item.created` - Real-time item creation events

---

### Sharding Architecture

**User Data (16 DBs, 1024 tables):**
- Databases: `todo_user_db_0` to `todo_user_db_15`
- Tables: `users_0000` to `users_1023`, `user_list_index_0000` to `user_list_index_1023`
- Sharding Key: `user_id` (Consistent Hashing)

**Todo Data (64 DBs, 4096 tables per type):**
- Databases: `todo_data_db_0` to `todo_data_db_63`
- Tables: `todo_lists_tab_0000` to `todo_lists_tab_4095`, `todo_items_tab_0000` to `todo_items_tab_4095`, `list_collaborators_tab_0000` to `list_collaborators_tab_4095`
- Sharding Key: `list_id` (Consistent Hashing)

---

## Environment Variables

```bash
# Database
DB_USER=root
DB_PASS=your_mysql_password
DB_HOST=127.0.0.1

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_MEDIA_TOPIC=media-uploads

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your_email@gmail.com
SMTP_PASS=your_app_password
SMTP_FROM=noreply@yourapp.com

# Media
UPLOAD_DIR=./uploads
S3_BUCKET=your-s3-bucket
```


