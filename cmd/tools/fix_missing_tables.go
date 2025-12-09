package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		log.Fatal("❌ DB_PASS environment variable required")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/?multiStatements=true", dbUser, dbPass)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("==========================================")
	fmt.Println("   Fixing Missing Tables")
	fmt.Println("==========================================")
	fmt.Println()

	// Fix User DBs
	fmt.Println("🔧 Fixing User Databases...")
	userTablesFixed := 0
	indexTablesFixed := 0

	for dbIdx := 0; dbIdx < 16; dbIdx++ {
		dbName := fmt.Sprintf("todo_user_db_%d", dbIdx)
		
		for tableIdx := dbIdx * 64; tableIdx < (dbIdx+1)*64; tableIdx++ {
			// Fix users_ table
			usersTable := fmt.Sprintf("users_%04d", tableIdx)
			if !tableExists(db, dbName, usersTable) {
				if err := createUserTable(db, dbName, tableIdx); err != nil {
					log.Printf("❌ Failed to create %s.%s: %v", dbName, usersTable, err)
				} else {
					userTablesFixed++
					if userTablesFixed <= 10 {
						fmt.Printf("   ✅ Created %s.%s\n", dbName, usersTable)
					}
				}
			}

			// Fix user_list_index_ table
			indexTable := fmt.Sprintf("user_list_index_%04d", tableIdx)
			if !tableExists(db, dbName, indexTable) {
				if err := createIndexTable(db, dbName, tableIdx); err != nil {
					log.Printf("❌ Failed to create %s.%s: %v", dbName, indexTable, err)
				} else {
					indexTablesFixed++
					if indexTablesFixed <= 10 {
						fmt.Printf("   ✅ Created %s.%s\n", dbName, indexTable)
					}
				}
			}
		}
	}

	if userTablesFixed > 10 {
		fmt.Printf("   ... and %d more users_ tables\n", userTablesFixed-10)
	}
	if indexTablesFixed > 10 {
		fmt.Printf("   ... and %d more user_list_index_ tables\n", indexTablesFixed-10)
	}

	fmt.Printf("✅ Fixed %d users_ tables and %d user_list_index_ tables\n", userTablesFixed, indexTablesFixed)
	fmt.Println()

	// Fix Data DBs
	fmt.Println("🔧 Fixing Data Databases...")
	listsTablesFixed := 0
	itemsTablesFixed := 0
	collabTablesFixed := 0

	for dbIdx := 0; dbIdx < 64; dbIdx++ {
		dbName := fmt.Sprintf("todo_data_db_%d", dbIdx)
		
		for tableIdx := dbIdx * 64; tableIdx < (dbIdx+1)*64; tableIdx++ {
			// Fix todo_lists_tab_
			listsTable := fmt.Sprintf("todo_lists_tab_%04d", tableIdx)
			if !tableExists(db, dbName, listsTable) {
				if err := createListsTable(db, dbName, tableIdx); err != nil {
					log.Printf("❌ Failed to create %s.%s: %v", dbName, listsTable, err)
				} else {
					listsTablesFixed++
				}
			}

			// Fix todo_items_tab_
			itemsTable := fmt.Sprintf("todo_items_tab_%04d", tableIdx)
			if !tableExists(db, dbName, itemsTable) {
				if err := createItemsTable(db, dbName, tableIdx); err != nil {
					log.Printf("❌ Failed to create %s.%s: %v", dbName, itemsTable, err)
				} else {
					itemsTablesFixed++
				}
			}

			// Fix list_collaborators_tab_
			collabTable := fmt.Sprintf("list_collaborators_tab_%04d", tableIdx)
			if !tableExists(db, dbName, collabTable) {
				if err := createCollabTable(db, dbName, tableIdx); err != nil {
					log.Printf("❌ Failed to create %s.%s: %v", dbName, collabTable, err)
				} else {
					collabTablesFixed++
				}
			}
		}
		
		if dbIdx%10 == 0 {
			fmt.Printf("   Processed %d/64 data databases...\n", dbIdx+1)
		}
	}

	fmt.Printf("✅ Fixed %d todo_lists_tab_ tables\n", listsTablesFixed)
	fmt.Printf("✅ Fixed %d todo_items_tab_ tables\n", itemsTablesFixed)
	fmt.Printf("✅ Fixed %d list_collaborators_tab_ tables\n", collabTablesFixed)
	fmt.Println()

	fmt.Println("==========================================")
	totalFixed := userTablesFixed + indexTablesFixed + listsTablesFixed + itemsTablesFixed + collabTablesFixed
	fmt.Printf("✅ Total tables created: %d\n", totalFixed)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Run verification:")
	fmt.Println("  go run cmd/tools/verify_sharding.go")
}

func tableExists(db *sql.DB, dbName, tableName string) bool {
	query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'", dbName, tableName)
	
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
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

func createListsTable(db *sql.DB, dbName string, tableID int) error {
	suffix := fmt.Sprintf("%04d", tableID)
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.todo_lists_tab_%s (
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
	
	_, err := db.Exec(query)
	return err
}

func createItemsTable(db *sql.DB, dbName string, tableID int) error {
	suffix := fmt.Sprintf("%04d", tableID)
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.todo_items_tab_%s (
		item_id BIGINT UNSIGNED NOT NULL,
		list_id BIGINT UNSIGNED NOT NULL,
		content TEXT,
		is_done TINYINT(1) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (item_id),
		KEY idx_list (list_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, suffix)
	
	_, err := db.Exec(query)
	return err
}

func createCollabTable(db *sql.DB, dbName string, tableID int) error {
	suffix := fmt.Sprintf("%04d", tableID)
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.list_collaborators_tab_%s (
		id BIGINT UNSIGNED AUTO_INCREMENT,
		list_id BIGINT UNSIGNED NOT NULL,
		user_id BIGINT UNSIGNED NOT NULL,
		role VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		UNIQUE KEY uk_list_user (list_id, user_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`, dbName, suffix)
	
	_, err := db.Exec(query)
	return err
}

