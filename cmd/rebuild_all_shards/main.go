package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

const (
	UserPhysicalDBs = 16
	UserTablesPerDB = 64
	TodoPhysicalDBs = 64
	TodoTablesPerDB = 64
)

func main() {
	log.Println("🔥 TodoList Shard Rebuild Tool - Will DROP and RECREATE all databases!")
	log.Println("⚠️  WARNING: This will DELETE ALL DATA in todo_user_db_* and todo_data_db_*")
	log.Println("⚠️  Press Ctrl+C now to abort, or Enter to continue...")
	fmt.Scanln()

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", dbUser, dbPass, dbHost, dbPort)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping MySQL: %v", err)
	}

	// Step 1: Drop all old databases
	log.Println("\n=== Step 1: Dropping old databases ===")
	dropAllDatabases(db)

	// Step 2: Create all databases
	log.Println("\n=== Step 2: Creating databases ===")
	createAllDatabases(db)

	// Step 3: Create user tables (16 DBs × 64 tables × 3 types)
	log.Println("\n=== Step 3: Creating user tables (16 DBs × 64 tables × 3 types) ===")
	createUserTables(db)

	// Step 4: Create todo tables (64 DBs × 64 tables × 3 types)
	log.Println("\n=== Step 4: Creating todo tables (64 DBs × 64 tables × 3 types) ===")
	createTodoTables(db)

	// Step 5: Verify counts
	log.Println("\n=== Step 5: Verifying database and table counts ===")
	verifyCounts(db)

	log.Println("\n✅ All shards rebuilt and verified successfully!")
}

func dropAllDatabases(db *sql.DB) {
	for i := 0; i < UserPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		if err != nil {
			log.Printf("⚠️  Failed to drop %s: %v", dbName, err)
		} else {
			log.Printf("  ✓ Dropped %s", dbName)
		}
	}

	for i := 0; i < TodoPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		if err != nil {
			log.Printf("⚠️  Failed to drop %s: %v", dbName, err)
		} else {
			log.Printf("  ✓ Dropped %s", dbName)
		}
	}
}

func createAllDatabases(db *sql.DB) {
	for i := 0; i < UserPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
		if err != nil {
			log.Fatalf("Failed to create %s: %v", dbName, err)
		}
		log.Printf("  ✓ Created %s", dbName)
	}

	for i := 0; i < TodoPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
		if err != nil {
			log.Fatalf("Failed to create %s: %v", dbName, err)
		}
		log.Printf("  ✓ Created %s", dbName)
	}
}

func createUserTables(db *sql.DB) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4) // Limit concurrency

	for dbIdx := 0; dbIdx < UserPhysicalDBs; dbIdx++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			dbName := fmt.Sprintf("todo_user_db_%d", idx)
			for t := 0; t < UserTablesPerDB; t++ {
				tableIdx := fmt.Sprintf("%04d", t)

				// Create users_ table
				usersSQL := fmt.Sprintf(`
CREATE TABLE %s.users_%s (
  user_id BIGINT UNSIGNED NOT NULL COMMENT 'Snowflake ID',
  email VARCHAR(255) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  verification_code VARCHAR(10),
  is_verified BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id),
  UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(usersSQL); err != nil {
					log.Fatalf("Failed to create users_%s in %s: %v", tableIdx, dbName, err)
				}

				// Create user_list_index_ table
				indexSQL := fmt.Sprintf(`
CREATE TABLE %s.user_list_index_%s (
  user_id BIGINT UNSIGNED NOT NULL,
  list_id BIGINT UNSIGNED NOT NULL,
  role VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, list_id),
  KEY idx_list_id (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(indexSQL); err != nil {
					log.Fatalf("Failed to create user_list_index_%s in %s: %v", tableIdx, dbName, err)
				}

				// Create user_email_index_ table
				emailIndexSQL := fmt.Sprintf(`
CREATE TABLE %s.user_email_index_%s (
  email VARCHAR(255) NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (email),
  KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(emailIndexSQL); err != nil {
					log.Fatalf("Failed to create user_email_index_%s in %s: %v", tableIdx, dbName, err)
				}
			}
			log.Printf("  ✓ Created %d tables in %s", UserTablesPerDB*3, dbName)
		}(dbIdx)
	}
	wg.Wait()
}

func createTodoTables(db *sql.DB) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8) // Higher concurrency for more DBs

	for dbIdx := 0; dbIdx < TodoPhysicalDBs; dbIdx++ {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			dbName := fmt.Sprintf("todo_data_db_%d", idx)
			for t := 0; t < TodoTablesPerDB; t++ {
				tableIdx := fmt.Sprintf("%04d", t)

				// Create todo_lists_tab_ table
				listsSQL := fmt.Sprintf(`
CREATE TABLE %s.todo_lists_tab_%s (
  list_id BIGINT UNSIGNED NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL,
  version INT UNSIGNED DEFAULT 1,
  is_deleted TINYINT(1) DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (list_id),
  KEY idx_owner (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(listsSQL); err != nil {
					log.Fatalf("Failed to create todo_lists_tab_%s in %s: %v", tableIdx, dbName, err)
				}

				// Create todo_items_tab_ table
				itemsSQL := fmt.Sprintf(`
CREATE TABLE %s.todo_items_tab_%s (
  item_id BIGINT UNSIGNED NOT NULL,
  list_id BIGINT UNSIGNED NOT NULL,
  content TEXT,
  name VARCHAR(255),
  description TEXT,
  status VARCHAR(50) DEFAULT 'not_started',
  priority VARCHAR(50) DEFAULT 'medium',
  due_date TIMESTAMP NULL,
  tags VARCHAR(500),
  is_done TINYINT(1) DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (item_id),
  KEY idx_list (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(itemsSQL); err != nil {
					log.Fatalf("Failed to create todo_items_tab_%s in %s: %v", tableIdx, dbName, err)
				}

				// Create list_collaborators_tab_ table
				collabSQL := fmt.Sprintf(`
CREATE TABLE %s.list_collaborators_tab_%s (
  id BIGINT UNSIGNED AUTO_INCREMENT,
  list_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  role VARCHAR(50) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_list_user (list_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, tableIdx)

				if _, err := db.Exec(collabSQL); err != nil {
					log.Fatalf("Failed to create list_collaborators_tab_%s in %s: %v", tableIdx, dbName, err)
				}
			}
			log.Printf("  ✓ Created %d tables in %s", TodoTablesPerDB*3, dbName)
		}(dbIdx)
	}
	wg.Wait()
}

func verifyCounts(db *sql.DB) {
	// Verify user databases
	var userDBCount int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'todo_user_db_%'").Scan(&userDBCount)
	if err != nil {
		log.Fatalf("Failed to count user databases: %v", err)
	}
	log.Printf("  User databases: %d (expected %d)", userDBCount, UserPhysicalDBs)
	if userDBCount != UserPhysicalDBs {
		log.Fatalf("❌ User database count mismatch!")
	}

	// Verify todo databases
	var todoDBCount int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'todo_data_db_%'").Scan(&todoDBCount)
	if err != nil {
		log.Fatalf("Failed to count todo databases: %v", err)
	}
	log.Printf("  Todo databases: %d (expected %d)", todoDBCount, TodoPhysicalDBs)
	if todoDBCount != TodoPhysicalDBs {
		log.Fatalf("❌ Todo database count mismatch!")
	}

	// Verify user tables
	for i := 0; i < UserPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		var tableCount int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '%s'", dbName)).Scan(&tableCount)
		if err != nil {
			log.Fatalf("Failed to count tables in %s: %v", dbName, err)
		}
		expectedTables := UserTablesPerDB * 3 // users_, user_list_index_, user_email_index_
		if tableCount != expectedTables {
			log.Fatalf("❌ %s has %d tables, expected %d", dbName, tableCount, expectedTables)
		}
	}
	log.Printf("  ✓ All user databases have correct table counts (%d tables each)", UserTablesPerDB*3)

	// Verify todo tables
	for i := 0; i < TodoPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		var tableCount int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '%s'", dbName)).Scan(&tableCount)
		if err != nil {
			log.Fatalf("Failed to count tables in %s: %v", dbName, err)
		}
		expectedTables := TodoTablesPerDB * 3 // todo_lists_tab_, todo_items_tab_, list_collaborators_tab_
		if tableCount != expectedTables {
			log.Fatalf("❌ %s has %d tables, expected %d", dbName, tableCount, expectedTables)
		}
	}
	log.Printf("  ✓ All todo databases have correct table counts (%d tables each)", TodoTablesPerDB*3)

	// Final summary
	log.Printf("\n📊 Final Summary:")
	log.Printf("  • User DBs: %d × %d tables × 3 types = %d total tables", UserPhysicalDBs, UserTablesPerDB, UserPhysicalDBs*UserTablesPerDB*3)
	log.Printf("  • Todo DBs: %d × %d tables × 3 types = %d total tables", TodoPhysicalDBs, TodoTablesPerDB, TodoPhysicalDBs*TodoTablesPerDB*3)
	log.Printf("  • Grand Total: %d databases, %d tables", UserPhysicalDBs+TodoPhysicalDBs, (UserPhysicalDBs*UserTablesPerDB*3)+(TodoPhysicalDBs*TodoTablesPerDB*3))
}
