package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"todolist-app/internal/infrastructure/sharding"
)

// Constants aligned with cmd/api/main.go
const (
	UserLogicalShards = 1024
	UserPhysicalDBs   = 16
	MaxRetries        = 5
	BatchSize         = 100
)

func main() {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")

	router := sharding.NewRouterV2(UserLogicalShards, 0) // Todo shards unused here

	// Connect user DB clusters
	var userDBs []*sql.DB
	for i := 0; i < UserPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true", dbUser, dbPass, dbName)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("failed to open %s: %v", dbName, err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("failed to connect %s: %v", dbName, err)
		}
		router.RegisterCluster(fmt.Sprintf("todo_user_db_%d", i), db, true, false)
		userDBs = append(userDBs, db)
	}
	defer func() {
		for _, db := range userDBs {
			db.Close()
		}
	}()

	for shard := 0; shard < UserLogicalShards; shard++ {
		nodeKey := router.UserRing.Get(fmt.Sprintf("%d", shard))
		cluster := router.Clusters[nodeKey]
		if cluster == nil {
			log.Fatalf("no cluster found for logical shard %d (nodeKey=%s)", shard, nodeKey)
		}
		if err := ensureRetryTable(cluster.DB); err != nil {
			log.Fatalf("ensure retry table failed on %s: %v", nodeKey, err)
		}
		if err := processRetryBatch(cluster.DB, nodeKey); err != nil {
			log.Printf("❌ retry processing failed on %s: %v", nodeKey, err)
		}
	}

	log.Println("✅ retry job completed")
}

func ensureRetryTable(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS user_list_index_retry (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  list_id BIGINT NOT NULL,
  role VARCHAR(32) NOT NULL,
  target_table VARCHAR(64) NOT NULL,
  err_msg TEXT,
  retries INT NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_user (user_id),
  KEY idx_list (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
	_, err := db.Exec(ddl)
	return err
}

func processRetryBatch(db *sql.DB, node string) error {
	rows, err := db.Query(`
SELECT id, user_id, list_id, role, target_table, retries
FROM user_list_index_retry
WHERE retries < ?
ORDER BY id ASC
LIMIT ?`, MaxRetries, BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	type rec struct {
		id          int64
		userID      int64
		listID      int64
		role        string
		targetTable string
		retries     int
	}
	var batch []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.userID, &r.listID, &r.role, &r.targetTable, &r.retries); err != nil {
			return err
		}
		batch = append(batch, r)
	}

	for _, r := range batch {
		if err := retryOne(db, r); err != nil {
			log.Printf("❌ retry failed node=%s id=%d user=%d list=%d table=%s err=%v", node, r.id, r.userID, r.listID, r.targetTable, err)
		} else {
			log.Printf("✅ retry success node=%s id=%d user=%d list=%d table=%s", node, r.id, r.userID, r.listID, r.targetTable)
		}
	}
	return nil
}

func retryOne(db *sql.DB, r struct {
	id          int64
	userID      int64
	listID      int64
	role        string
	targetTable string
	retries     int
}) error {
	query := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", r.targetTable)
	if _, err := db.Exec(query, r.userID, r.listID, r.role); err != nil {
		_, _ = db.Exec(`UPDATE user_list_index_retry SET retries = retries + 1, last_error = ? WHERE id = ?`, err.Error(), r.id)
		return err
	}
	_, _ = db.Exec(`DELETE FROM user_list_index_retry WHERE id = ?`, r.id)
	return nil
}

