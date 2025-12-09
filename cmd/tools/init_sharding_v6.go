package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Sharding Configuration V6: Robust Initialization
const (
	UserDBCount     = 16
	UserTablePerDB  = 64 // 16 * 64 = 1024

	DataDBCount     = 64
	DataTablePerDB  = 64 // 64 * 64 = 4096
)

func main() {
	host := "127.0.0.1"
	port := "3306"
	user := "root"
	password := "115119_hH"
	
	if v := os.Getenv("DB_PASS"); v != "" { password = v }

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?multiStatements=true&parseTime=true", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	
	// Increase connection pool for heavy concurrency
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil { log.Fatal(err) }

	log.Println("🚀 Starting Sharding Initialization V6 (Robust)...")
	
	// 1. Create Databases (Sequential to ensure they exist)
	createDBs(db, "todo_user_db", UserDBCount)
	createDBs(db, "todo_data_db", DataDBCount)

	// 2. Init User DBs (Users + Index)
	// Strategy: Process DBs in chunks to avoid overwhelming MySQL
	// Total 16 DBs, we can do 4 parallel DB inits
	
	log.Printf("Initializing %d User Tables across %d DBs...", UserDBCount*UserTablePerDB, UserDBCount)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4) // Limit concurrency to 4 DBs at a time

	for dbIdx := 0; dbIdx < UserDBCount; dbIdx++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(d int) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			dbName := fmt.Sprintf("todo_user_db_%d", d)
			startTbl := d * UserTablePerDB
			endTbl := startTbl + UserTablePerDB
			
			// Sequential table creation within a DB is faster/safer than hitting same DB concurrently
			for t := startTbl; t < endTbl; t++ {
				if err := createUserTable(db, dbName, t); err != nil {
					log.Printf("❌ Failed users_%04d: %v", t, err)
				}
				if err := createIndexTable(db, dbName, t); err != nil {
					log.Printf("❌ Failed user_list_index_%04d: %v", t, err)
				}
			}
			log.Printf("✅ Initialized %s (Tables %04d-%04d)", dbName, startTbl, endTbl-1)
		}(dbIdx)
	}
	wg.Wait()

	// 3. Init Data DBs
	// Total 64 DBs. Limit concurrency.
	log.Printf("Initializing %d Todo Tables across %d DBs...", DataDBCount*DataTablePerDB, DataDBCount)
	
	for dbIdx := 0; dbIdx < DataDBCount; dbIdx++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(d int) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			dbName := fmt.Sprintf("todo_data_db_%d", d)
			startTbl := d * DataTablePerDB
			endTbl := startTbl + DataTablePerDB
			
			for t := startTbl; t < endTbl; t++ {
				if err := createTodoTables(db, dbName, t); err != nil {
					log.Printf("❌ Failed todo_tables_%04d: %v", t, err)
				}
			}
			// log.Printf("✅ Initialized %s", dbName) // Reduce noise
		}(dbIdx)
	}
	
	wg.Wait()
	log.Println("✅ All Shards Initialized Successfully (V6)!")
}

func createDBs(db *sql.DB, prefix string, count int) {
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s_%d", prefix, i)
		_, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", name))
		if err != nil { log.Printf("Failed to create DB %s: %v", name, err) }
	}
}

func createUserTable(db *sql.DB, dbName string, tableID int) error {
	tableName := fmt.Sprintf("users_%04d", tableID)
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
		user_id BIGINT UNSIGNED NOT NULL COMMENT 'Snowflake ID',
		email VARCHAR(128) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		verification_code VARCHAR(10),
		is_verified BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id),
		UNIQUE KEY uk_email (email)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableName)
	_, err := db.Exec(query)
	return err
}

func createIndexTable(db *sql.DB, dbName string, tableID int) error {
	tableName := fmt.Sprintf("user_list_index_%04d", tableID)
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
		user_id BIGINT UNSIGNED NOT NULL,
		list_id BIGINT UNSIGNED NOT NULL,
		role VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, list_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableName)
	_, err := db.Exec(query)
	return err
}

func createTodoTables(db *sql.DB, dbName string, tableID int) error {
	suffix := fmt.Sprintf("%04d", tableID)
	
	// Transaction or Batch? No, DDL implicitly commits.
	// Just chain execution.
	
	q1 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.todo_lists_tab_%s (
		list_id BIGINT UNSIGNED NOT NULL,
		owner_id BIGINT UNSIGNED NOT NULL,
		title VARCHAR(255) NOT NULL,
		version INT UNSIGNED DEFAULT 1,
		is_deleted TINYINT(1) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		PRIMARY KEY (list_id),
		KEY idx_owner (owner_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, suffix)
	if _, err := db.Exec(q1); err != nil { return err }

	q2 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.todo_items_tab_%s (
		item_id BIGINT UNSIGNED NOT NULL,
		list_id BIGINT UNSIGNED NOT NULL,
		content TEXT,
		is_done TINYINT(1) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (item_id),
		KEY idx_list (list_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, suffix)
	if _, err := db.Exec(q2); err != nil { return err }

	q3 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.list_collaborators_tab_%s (
		id BIGINT UNSIGNED AUTO_INCREMENT,
		list_id BIGINT UNSIGNED NOT NULL,
		user_id BIGINT UNSIGNED NOT NULL,
		role VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY uk_list_user (list_id, user_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, suffix)
	if _, err := db.Exec(q3); err != nil { return err }
	
	return nil
}


