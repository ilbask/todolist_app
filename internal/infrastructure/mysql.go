package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQLConnection(host string, port int, user, password, dbName string) (*sql.DB, error) {
	// 1. DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true", 
		user, password, host, port, dbName)
	
	var db *sql.DB
	var err error

	// 2. Retry Loop (Useful for waiting for DB startup)
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
	log.Println("✅ Connected to MySQL Database")
				db.SetMaxOpenConns(25)
				db.SetMaxIdleConns(5)
				db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
		}
		
		log.Printf("⏳ Waiting for MySQL (%s:%d)... (%d/10) Error: %v", host, port, i+1, err)
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to mysql: %v", err)
}


