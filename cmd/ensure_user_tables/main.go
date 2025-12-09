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
	userDBCount    = 16
	tablesPerDB    = 64
	defaultDBHost  = "127.0.0.1"
	defaultDBPort  = 3306
	defaultCharset = "utf8mb4"
)

func main() {
	var (
		host = flag.String("host", envOrDefault("DB_HOST", defaultDBHost), "MySQL host")
		port = flag.Int("port", envOrDefaultInt("DB_PORT", defaultDBPort), "MySQL port")
		user = flag.String("user", envOrDefault("DB_USER", "root"), "MySQL user")
		pass = flag.String("pass", os.Getenv("DB_PASS"), "MySQL password")
	)
	flag.Parse()

	failures := false
	for dbIdx := 0; dbIdx < userDBCount; dbIdx++ {
		dbName := fmt.Sprintf("todo_user_db_%d", dbIdx)
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

		if err := ensureTables(db, dbName); err != nil {
			log.Printf("❌ %s ensure tables failed: %v", dbName, err)
			failures = true
		}
		db.Close()
	}

	if failures {
		log.Fatal("Some shards failed to initialize/verify; check logs above.")
	}
	log.Println("✅ All todo_user_db_* shards contain complete users/index tables.")
}

func ensureTables(db *sql.DB, schema string) error {
	for t := 0; t < tablesPerDB; t++ {
		if err := ensureUserTable(db, t); err != nil {
			return fmt.Errorf("users_%04d: %w", t, err)
		}
		if err := ensureUserListIndex(db, t); err != nil {
			return fmt.Errorf("user_list_index_%04d: %w", t, err)
		}
		if err := ensureUserEmailIndex(db, t); err != nil {
			return fmt.Errorf("user_email_index_%04d: %w", t, err)
		}
	}

	missing := verifyTables(db, schema)
	if len(missing) > 0 {
		return fmt.Errorf("missing tables: %v", missing)
	}

	log.Printf("✅ %s shard complete (%d tables x 3)", schema, tablesPerDB)
	return nil
}

func ensureUserTable(db *sql.DB, idx int) error {
	table := fmt.Sprintf("users_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	user_id BIGINT UNSIGNED NOT NULL,
	email VARCHAR(255) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	verification_code VARCHAR(10),
	is_verified BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id),
	UNIQUE KEY uk_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func ensureUserListIndex(db *sql.DB, idx int) error {
	table := fmt.Sprintf("user_list_index_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	user_id BIGINT UNSIGNED NOT NULL,
	list_id BIGINT UNSIGNED NOT NULL,
	role VARCHAR(50) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	PRIMARY KEY (user_id, list_id),
	KEY idx_list (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func ensureUserEmailIndex(db *sql.DB, idx int) error {
	table := fmt.Sprintf("user_email_index_%04d", idx)
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	email VARCHAR(255) NOT NULL,
	user_id BIGINT UNSIGNED NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (email),
	KEY idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=%s;`, table, defaultCharset)
	_, err := db.Exec(stmt)
	return err
}

func verifyTables(db *sql.DB, schema string) []string {
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
	for t := 0; t < tablesPerDB; t++ {
		for _, prefix := range []string{"users_", "user_list_index_", "user_email_index_"} {
			name := fmt.Sprintf("%s%04d", prefix, t)
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
