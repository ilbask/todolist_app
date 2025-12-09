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

	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/", dbUser, dbPass)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("==========================================")
	fmt.Println("   Checking for Missing Tables")
	fmt.Println("==========================================")
	fmt.Println()

	// Check User DBs (16 DBs, 64 tables each = 1024 total)
	fmt.Println("🔍 Checking User Databases (Expected: 1024 users_ and 1024 user_list_index_)")
	fmt.Println()

	missingUserTables := []string{}
	missingIndexTables := []string{}

	for dbIdx := 0; dbIdx < 16; dbIdx++ {
		dbName := fmt.Sprintf("todo_user_db_%d", dbIdx)
		
		for tableIdx := dbIdx * 64; tableIdx < (dbIdx+1)*64; tableIdx++ {
			// Check users_ table
			usersTable := fmt.Sprintf("users_%04d", tableIdx)
			if !tableExists(db, dbName, usersTable) {
				missingUserTables = append(missingUserTables, fmt.Sprintf("%s.%s", dbName, usersTable))
			}

			// Check user_list_index_ table
			indexTable := fmt.Sprintf("user_list_index_%04d", tableIdx)
			if !tableExists(db, dbName, indexTable) {
				missingIndexTables = append(missingIndexTables, fmt.Sprintf("%s.%s", dbName, indexTable))
			}
		}
	}

	if len(missingUserTables) > 0 {
		fmt.Printf("❌ Missing %d users_ tables:\n", len(missingUserTables))
		for _, table := range missingUserTables {
			fmt.Printf("   - %s\n", table)
		}
	} else {
		fmt.Println("✅ All 1024 users_ tables exist")
	}

	if len(missingIndexTables) > 0 {
		fmt.Printf("❌ Missing %d user_list_index_ tables:\n", len(missingIndexTables))
		for _, table := range missingIndexTables {
			fmt.Printf("   - %s\n", table)
		}
	} else {
		fmt.Println("✅ All 1024 user_list_index_ tables exist")
	}

	fmt.Println()

	// Check Data DBs (64 DBs, 64 tables each = 4096 total for each type)
	fmt.Println("🔍 Checking Data Databases (Expected: 4096 tables for each type)")
	fmt.Println()

	missingListsTables := []string{}
	missingItemsTables := []string{}
	missingCollabTables := []string{}

	for dbIdx := 0; dbIdx < 64; dbIdx++ {
		dbName := fmt.Sprintf("todo_data_db_%d", dbIdx)
		
		for tableIdx := dbIdx * 64; tableIdx < (dbIdx+1)*64; tableIdx++ {
			// Check todo_lists_tab_
			listsTable := fmt.Sprintf("todo_lists_tab_%04d", tableIdx)
			if !tableExists(db, dbName, listsTable) {
				missingListsTables = append(missingListsTables, fmt.Sprintf("%s.%s", dbName, listsTable))
			}

			// Check todo_items_tab_
			itemsTable := fmt.Sprintf("todo_items_tab_%04d", tableIdx)
			if !tableExists(db, dbName, itemsTable) {
				missingItemsTables = append(missingItemsTables, fmt.Sprintf("%s.%s", dbName, itemsTable))
			}

			// Check list_collaborators_tab_
			collabTable := fmt.Sprintf("list_collaborators_tab_%04d", tableIdx)
			if !tableExists(db, dbName, collabTable) {
				missingCollabTables = append(missingCollabTables, fmt.Sprintf("%s.%s", dbName, collabTable))
			}
		}
	}

	if len(missingListsTables) > 0 {
		fmt.Printf("❌ Missing %d todo_lists_tab_ tables:\n", len(missingListsTables))
		for i, table := range missingListsTables {
			if i < 10 {
				fmt.Printf("   - %s\n", table)
			}
		}
		if len(missingListsTables) > 10 {
			fmt.Printf("   ... and %d more\n", len(missingListsTables)-10)
		}
	} else {
		fmt.Println("✅ All 4096 todo_lists_tab_ tables exist")
	}

	if len(missingItemsTables) > 0 {
		fmt.Printf("❌ Missing %d todo_items_tab_ tables:\n", len(missingItemsTables))
		for i, table := range missingItemsTables {
			if i < 10 {
				fmt.Printf("   - %s\n", table)
			}
		}
		if len(missingItemsTables) > 10 {
			fmt.Printf("   ... and %d more\n", len(missingItemsTables)-10)
		}
	} else {
		fmt.Println("✅ All 4096 todo_items_tab_ tables exist")
	}

	if len(missingCollabTables) > 0 {
		fmt.Printf("❌ Missing %d list_collaborators_tab_ tables:\n", len(missingCollabTables))
		for i, table := range missingCollabTables {
			if i < 10 {
				fmt.Printf("   - %s\n", table)
			}
		}
		if len(missingCollabTables) > 10 {
			fmt.Printf("   ... and %d more\n", len(missingCollabTables)-10)
		}
	} else {
		fmt.Println("✅ All 4096 list_collaborators_tab_ tables exist")
	}

	fmt.Println()
	fmt.Println("==========================================")
	
	totalMissing := len(missingUserTables) + len(missingIndexTables) + 
		len(missingListsTables) + len(missingItemsTables) + len(missingCollabTables)
	
	if totalMissing > 0 {
		fmt.Printf("❌ Total missing tables: %d\n", totalMissing)
		fmt.Println("==========================================")
		fmt.Println()
		fmt.Println("To fix:")
		fmt.Println("  go run cmd/tools/fix_missing_tables.go")
	} else {
		fmt.Println("✅ All tables are present!")
		fmt.Println("==========================================")
	}
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

