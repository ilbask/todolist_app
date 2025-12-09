package infrastructure

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func NewSQLiteConnection(dbPath string) (*sql.DB, error) {
	// Create file if not exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Enable Foreign Keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, err
	}

	log.Println("✅ Connected to SQLite Database")
	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		user_id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		verification_code TEXT,
		is_verified BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS todo_lists (
		list_id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(user_id)
	);

	CREATE TABLE IF NOT EXISTS todo_items (
		item_id INTEGER PRIMARY KEY AUTOINCREMENT,
		list_id INTEGER NOT NULL,
		content TEXT NOT NULL,
		is_done BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (list_id) REFERENCES todo_lists(list_id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS list_collaborators (
		list_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		PRIMARY KEY (list_id, user_id),
		FOREIGN KEY (list_id) REFERENCES todo_lists(list_id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
	);
	`
	_, err := db.Exec(schema)
	return err
}



