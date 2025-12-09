package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

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

	log.Println("==========================================")
	log.Println("   Migrating Todo Items Schema")
	log.Println("==========================================")
	log.Println("")
	log.Println("添加扩展字段:")
	log.Println("  - name (名称)")
	log.Println("  - description (描述)")
	log.Println("  - status (状态: not_started/in_progress/completed)")
	log.Println("  - priority (优先级: high/medium/low)")
	log.Println("  - due_date (截止日期)")
	log.Println("  - tags (标签)")
	log.Println("  - updated_at (更新时间)")
	log.Println("")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数
	migrated := 0
	failed := 0
	var mu sync.Mutex

	// 迁移所有64个Data DB中的todo_items_tab表
	for dbIdx := 0; dbIdx < 64; dbIdx++ {
		dbName := fmt.Sprintf("todo_data_db_%d", dbIdx)
		
		for tableIdx := dbIdx * 64; tableIdx < (dbIdx+1)*64; tableIdx++ {
			wg.Add(1)
			semaphore <- struct{}{}
			
			go func(db_name string, table_id int) {
				defer wg.Done()
				defer func() { <-semaphore }()
				
				tableName := fmt.Sprintf("todo_items_tab_%04d", table_id)
				
				if err := migrateTable(db, db_name, tableName); err != nil {
					log.Printf("❌ Failed to migrate %s.%s: %v", db_name, tableName, err)
					mu.Lock()
					failed++
					mu.Unlock()
				} else {
					mu.Lock()
					migrated++
					if migrated%100 == 0 {
						log.Printf("   Progress: %d/4096 tables migrated...", migrated)
					}
					mu.Unlock()
				}
			}(dbName, tableIdx)
		}
	}

	wg.Wait()

	log.Println("")
	log.Println("==========================================")
	log.Printf("✅ Migration Complete!")
	log.Printf("   Migrated: %d tables", migrated)
	if failed > 0 {
		log.Printf("   Failed: %d tables", failed)
	}
	log.Println("==========================================")
	log.Println("")
	log.Println("现在可以使用扩展功能了！")
	log.Println("  - 创建带扩展字段的Item")
	log.Println("  - 按状态/优先级/截止日期筛选")
	log.Println("  - 按截止日期/优先级/名称排序")
}

func migrateTable(db *sql.DB, dbName, tableName string) error {
	// 检查是否已经迁移过
	checkSQL := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = '%s' 
		AND TABLE_NAME = '%s' 
		AND COLUMN_NAME = 'name'
	`, dbName, tableName)
	
	var count int
	if err := db.QueryRow(checkSQL).Scan(&count); err != nil {
		return err
	}
	
	if count > 0 {
		// 已经迁移过，跳过
		return nil
	}

	// 执行迁移
	migrateSQL := fmt.Sprintf(`
		ALTER TABLE %s.%s
		ADD COLUMN name VARCHAR(255) NOT NULL DEFAULT '' AFTER content,
		ADD COLUMN description TEXT AFTER name,
		ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'not_started' AFTER description,
		ADD COLUMN priority VARCHAR(10) NOT NULL DEFAULT 'medium' AFTER status,
		ADD COLUMN due_date DATETIME AFTER priority,
		ADD COLUMN tags VARCHAR(500) AFTER due_date,
		ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at,
		ADD INDEX idx_status (status),
		ADD INDEX idx_priority (priority),
		ADD INDEX idx_due_date (due_date)
	`, dbName, tableName)

	_, err := db.Exec(migrateSQL)
	return err
}

