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
)

func main() {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")

	router := sharding.NewRouterV2(UserLogicalShards, 0) // todo shards unused here

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

	// Create tables per logical shard using router mapping
	created := make(map[string]bool)
	for shard := 0; shard < UserLogicalShards; shard++ {
		nodeKey := router.UserRing.Get(fmt.Sprintf("%d", shard))
		cluster := router.Clusters[nodeKey]
		if cluster == nil {
			log.Fatalf("no cluster found for logical shard %d (nodeKey=%s)", shard, nodeKey)
		}

		table := fmt.Sprintf("user_email_index_%04d", shard)
		sqlStmt := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  email VARCHAR(255) NOT NULL,
  user_id BIGINT NOT NULL,
  PRIMARY KEY (email),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`, table)

		if _, err := cluster.DB.Exec(sqlStmt); err != nil {
			log.Fatalf("failed creating table %s: %v", table, err)
		}
		if !created[table] {
			created[table] = true
			log.Printf("✅ ensured %s on %s", table, nodeKey)
		}
	}

	log.Printf("✅ email index tables ensured across %d logical shards", UserLogicalShards)
}

