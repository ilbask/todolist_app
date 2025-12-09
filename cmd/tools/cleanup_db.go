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
	password := "115119_hH"
	
	if v := os.Getenv("DB_PASS"); v != "" { password = v }

	// Connect to MySQL Root
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?multiStatements=true&parseTime=true", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Could not connect to MySQL:", err)
	}

	log.Println("🗑️  Cleaning up all databases...")

	// 1. Drop User DBs
	dropDBs(db, "todo_user_db", 16)
	
	// 2. Drop Data DBs
	dropDBs(db, "todo_data_db", 64)
	
	// 3. Drop Index DBs
	dropDBs(db, "todo_index_db", 16)
	
	// 4. Drop Legacy/Init DBs if any
	dropDBs(db, "todo_app", 1)

	log.Println("✅ All Databases Dropped Successfully!")
}

func dropDBs(db *sql.DB, prefix string, count int) {
	for i := 0; i < count; i++ {
		name := prefix
		if count > 1 {
			name = fmt.Sprintf("%s_%d", prefix, i)
		}
		
		_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
		if err != nil {
			log.Printf("⚠️ Failed to drop %s: %v", name, err)
		} else {
			// log.Printf("Dropped %s", name) // Too verbose for 64+ DBs
		}
	}
	log.Printf("Dropped pattern %s (%d DBs)", prefix, count)
}


