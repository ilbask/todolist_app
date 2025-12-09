package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const (
	dataDBCount    = 64
	tablesPerData  = 64
	defaultHost    = "127.0.0.1"
	defaultPort    = 3306
	defaultCharset = "utf8mb4"
)

func main() {
	var (
		host = flag.String("host", envOrDefault("DB_HOST", defaultHost), "MySQL host")
		port = flag.Int("port", envOrDefaultInt("DB_PORT", defaultPort), "MySQL port")
		user = flag.String("user", envOrDefault("DB_USER", "root"), "MySQL user")
		pass = flag.String("pass", os.Getenv("DB_PASS"), "MySQL password")
	)
	flag.Parse()

	failures := false
	for dbIdx := 0; dbIdx < dataDBCount; dbIdx++ {
		dbName := fmt.Sprintf("todo_data_db_%d", dbIdx)
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=%s&multiStatements=true",
			*user, *pass, *host, *port, dbName, defaultCharset)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("❌ %s open failed: %v", dbName, err)
			failures = true
			continue
		}
		if err := db.Ping(); err != nil {
			log.Printf("❌ %s ping failed: %v", dbName, err)
			failures = true
			db.Close()
			continue
		}

		if err := ensureTodoTables(db, dbName); err != nil {
			log.Printf("❌ %s ensure tables failed: %v", dbName, err)
			failures = true
		}
		db.Close()
	}

	if failures {
		log.Fatal("Some todo_data_db_* shards were incomplete. See logs above.")
	}
	log.Println("✅ All todo_data_db_* shards contain list/item/collaborator tables (64×).")
}

func ensureTodoTables(db *sql.DB, schema string) error {
	for idx := 0; idx < tablesPerData; idx++ {
		if err := ensureListTable(db, idx); err != nil {
			return fmt.Errorf("todo_lists_tab_%04d: %w", idx, err)
		}
		if err := ensureItemTable(db, idx); err != nil {
			return fmt.Errorf("todo_items_tab_%04d: %w", idx, err)
		}
		if err := ensureCollabTable(db, idx); err != nil {
			return fmt.Errorf("list_collaborators_tab_%04d: %w", idx, err)
		}
	}

	missing := verifyTodoTables(db, schema)
	if len(missing) > 0 {
		return fmt.Errorf("missing tables: %v", missing)
	}

	log.Printf("✅ %s shard complete (%d logical tables ×3)", schema, tablesPerData)
	return nil
}

func ensureListTable(db *sql.DB, idx int) error {
	table := fmt.Sprintf("todo_lists_tab_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	list_id BIGINT UNSIGNED NOT NULL,
	owner_id BIGINT UNSIGNED NOT NULL,
	title VARCHAR(255) NOT NULL,
	version INT UNSIGNED DEFAULT 1,
	is_deleted TINYINT(1) DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (list_id),
	KEY idx_owner (owner_id)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func ensureItemTable(db *sql.DB, idx int) error {
	table := fmt.Sprintf("todo_items_tab_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	item_id BIGINT UNSIGNED NOT NULL,
	list_id BIGINT UNSIGNED NOT NULL,
	content TEXT,
	name VARCHAR(255),
	description TEXT,
	status VARCHAR(32),
	priority VARCHAR(32),
	due_date DATETIME NULL,
	tags JSON,
	is_done TINYINT(1) DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (item_id),
	KEY idx_list (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func ensureCollabTable(db *sql.DB, idx int) error {
	table := fmt.Sprintf("list_collaborators_tab_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	id BIGINT UNSIGNED AUTO_INCREMENT,
	list_id BIGINT UNSIGNED NOT NULL,
	user_id BIGINT UNSIGNED NOT NULL,
	role VARCHAR(50) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	UNIQUE KEY uk_list_user (list_id, user_id)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func verifyTodoTables(db *sql.DB, schema string) []string {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = ?`
	rows, err := db.Query(query, schema)
	if err != nil {
		log.Printf("⚠️ verification query failed for %s: %v", schema, err)
		return []string{"<verification error>"}
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			existing[name] = struct{}{}
		}
	}

	var missing []string
	for idx := 0; idx < tablesPerData; idx++ {
		for _, prefix := range []string{"todo_lists_tab_", "todo_items_tab_", "list_collaborators_tab_"} {
			name := fmt.Sprintf("%s%04d", prefix, idx)
			if _, ok := existing[name]; !ok {
				missing = append(missing, name)
			}
		}
	}
	return missing
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var parsed int
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return def
}
