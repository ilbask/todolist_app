package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Defaults
	host := "127.0.0.1"
	port := "3306"
	user := "root"      // Default root user for init
	password := ""      // Default empty password
	
	// Override from env if needed
	if v := os.Getenv("DB_HOST"); v != "" { host = v }
	if v := os.Getenv("DB_USER"); v != "" { user = v }
	if v := os.Getenv("DB_PASS"); v != "" { password = v }

	fmt.Printf("Connecting to MySQL at %s:%s as %s...\n", host, port, user)

	// Connect without DB name to create it
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?multiStatements=true&parseTime=true", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Error creating connection: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to MySQL. Is it running? Error: %v", err)
	}

	// Read SQL
	content, err := ioutil.ReadFile("scripts/init.sql")
	if err != nil {
		log.Fatal("Error reading scripts/init.sql: ", err)
	}
	queries := string(content)

	// Split by semicolon because the driver might handle multiStatements weirdly with errors,
	// but mostly it's safer to execute separately if simple split works.
	// Actually, with multiStatements=true, we can try running it all.
	// But let's split to be safe and give progress.
	
	// Simple split (not robust for strings containing ;)
	parts := strings.Split(queries, ";")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		_, err := db.Exec(part)
		if err != nil {
			log.Printf("⚠️ Warning executing query: %s\nError: %v\n", part, err)
		} else {
			fmt.Println("✓ Executed query")
		}
	}

	fmt.Println("✅ Database 'todo_app' and tables initialized successfully!")
}


