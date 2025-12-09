package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	host := "127.0.0.1"
	port := "3306"
	user := "root"
	password := os.Getenv("DB_PASS")
	if password == "" {
		password = "115119_hH"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("❌ Failed to connect:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("❌ MySQL not reachable:", err)
	}

	log.Println("🔍 Verifying Sharding Setup...")
	log.Println()

	// Verify User DBs (16 DBs, 64 tables each)
	log.Println("📊 User DB Verification (Expected: 16 DBs, 64 tables/DB)")
	userDBErrors := 0
	for i := 0; i < 16; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		
		// Count users_* tables
		userCount := countTables(db, dbName, "users_%")
		indexCount := countTables(db, dbName, "user_list_index_%")
		
		status := "✅"
		if userCount != 64 || indexCount != 64 {
			status = "❌"
			userDBErrors++
		}
		
		log.Printf("  %s %s: users=%d, index=%d", status, dbName, userCount, indexCount)
	}

	log.Println()
	log.Println("📊 Data DB Verification (Expected: 64 DBs, 64 tables/DB of each type)")
	dataDBErrors := 0
	for i := 0; i < 64; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		
		listCount := countTables(db, dbName, "todo_lists_tab_%")
		itemCount := countTables(db, dbName, "todo_items_tab_%")
		collabCount := countTables(db, dbName, "list_collaborators_tab_%")
		
		status := "✅"
		if listCount != 64 || itemCount != 64 || collabCount != 64 {
			status = "❌"
			dataDBErrors++
		}
		
		// Only log errors or first/last few DBs to reduce noise
		if status == "❌" || i < 3 || i >= 61 {
			log.Printf("  %s %s: lists=%d, items=%d, collab=%d", status, dbName, listCount, itemCount, collabCount)
		}
	}

	log.Println()
	log.Println("========================================")
	if userDBErrors == 0 && dataDBErrors == 0 {
		log.Println("✅ ALL SHARDS VERIFIED SUCCESSFULLY!")
		log.Println("   - 16 User DBs with 1024 user tables")
		log.Println("   - 16 User DBs with 1024 index tables")
		log.Println("   - 64 Data DBs with 4096 lists/items/collab tables each")
	} else {
		log.Printf("❌ VERIFICATION FAILED:")
		if userDBErrors > 0 {
			log.Printf("   - %d User DBs have incorrect table counts", userDBErrors)
		}
		if dataDBErrors > 0 {
			log.Printf("   - %d Data DBs have incorrect table counts", dataDBErrors)
		}
		log.Println("   Run: go run cmd/tools/cleanup_db.go && go run cmd/tools/init_sharding_v6.go")
	}
	log.Println("========================================")
}

func countTables(db *sql.DB, dbName, pattern string) int {
	query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME LIKE '%s'", dbName, pattern)
	
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		return -1
	}
	return count
}


