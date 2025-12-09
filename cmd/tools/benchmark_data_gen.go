package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"todolist-app/internal/infrastructure"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/pkg/uid"
)

var (
	numUsers     = flag.Int64("users", 1000000, "Number of users to generate (default: 1M, max: 1B)")
	listsPerUser = flag.Int("lists", 10, "Number of todo lists per user")
	itemsPerList = flag.Int("items", 10, "Number of items per list")
	batchSize    = flag.Int("batch", 1000, "Batch size for inserts")
	workers      = flag.Int("workers", 10, "Number of concurrent workers")
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("🚀 Starting Benchmark Data Generation")
	log.Printf("📊 Target: %d users, %d lists/user, %d items/list", *numUsers, *listsPerUser, *itemsPerList)
	log.Printf("⚙️  Workers: %d, Batch size: %d", *workers, *batchSize)

	// Initialize connections
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		log.Fatal("❌ DB_PASS environment variable required")
	}

	// Initialize Snowflake ID generator
	idGen, err := uid.NewSnowflake(1, 1)
	if err != nil {
		log.Fatal("❌ Failed to create ID generator:", err)
	}

	// Initialize Sharding Router
	router := sharding.NewShardingRouterV2(dbUser, dbPass)

	// Start generation
	startTime := time.Now()

	// Generate users in parallel
	generateUsers(router, idGen, *numUsers, *workers, *batchSize, *listsPerUser, *itemsPerList)

	duration := time.Since(startTime)
	log.Printf("✅ Data generation completed in %s", duration)
	log.Printf("📈 Throughput: %.2f users/sec", float64(*numUsers)/duration.Seconds())
}

func generateUsers(router *sharding.ShardingRouterV2, idGen *uid.Snowflake, totalUsers int64, workers, batchSize, listsPerUser, itemsPerList int) {
	var wg sync.WaitGroup
	userChan := make(chan int64, workers*10)

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			processUsers(router, idGen, userChan, workerID, batchSize, listsPerUser, itemsPerList)
		}(i)
	}

	// Feed user IDs to workers
	for i := int64(0); i < totalUsers; i++ {
		userChan <- i
	}
	close(userChan)

	wg.Wait()
}

func processUsers(router *sharding.ShardingRouterV2, idGen *uid.Snowflake, userChan <-chan int64, workerID, batchSize, listsPerUser, itemsPerList int) {
	for range userChan {
		userID := idGen.Generate()
		email := fmt.Sprintf("user_%d@benchmark.test", userID)

		// Insert user
		if err := insertUser(router, userID, email); err != nil {
			log.Printf("❌ Worker %d: Failed to insert user %d: %v", workerID, userID, err)
			continue
		}

		// Generate lists for this user
		for l := 0; l < listsPerUser; l++ {
			listID := idGen.Generate()
			title := fmt.Sprintf("List %d", l+1)

			if err := insertList(router, listID, userID, title); err != nil {
				log.Printf("❌ Worker %d: Failed to insert list %d: %v", workerID, listID, err)
				continue
			}

			// Generate items for this list
			for it := 0; it < itemsPerList; it++ {
				itemID := idGen.Generate()
				content := fmt.Sprintf("Item %d", it+1)

				if err := insertItem(router, itemID, listID, content); err != nil {
					log.Printf("❌ Worker %d: Failed to insert item %d: %v", workerID, itemID, err)
				}
			}
		}

		if userID%10000 == 0 {
			log.Printf("✅ Worker %d: Processed %d users", workerID, userID)
		}
	}
}

func insertUser(router *sharding.ShardingRouterV2, userID int64, email string) error {
	db, tableName := router.RouteUser(userID)

	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, email, password_hash, region_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`, tableName)

	_, err := db.Exec(query, userID, email, "benchmark_hash", "US")
	return err
}

func insertList(router *sharding.ShardingRouterV2, listID, ownerID int64, title string) error {
	db, tableName := router.RouteTodoData(listID, "todo_lists")

	query := fmt.Sprintf(`
		INSERT INTO %s (list_id, owner_id, title, version, is_deleted, created_at, updated_at)
		VALUES (?, ?, ?, 1, 0, NOW(), NOW())
	`, tableName)

	if _, err := db.Exec(query, listID, ownerID, title); err != nil {
		return err
	}

	// Also insert into user_list_index
	indexDB, indexTable := router.RouteUserListIndex(ownerID)
	indexQuery := fmt.Sprintf(`
		INSERT INTO %s (user_id, list_id, role, created_at)
		VALUES (?, ?, 1, NOW())
	`, indexTable)

	_, err := indexDB.Exec(indexQuery, ownerID, listID)
	return err
}

func insertItem(router *sharding.ShardingRouterV2, itemID, listID int64, content string) error {
	db, tableName := router.RouteTodoData(listID, "todo_items")

	query := fmt.Sprintf(`
		INSERT INTO %s (item_id, list_id, content, is_done, sort_rank, created_at, updated_at)
		VALUES (?, ?, ?, 0, '', NOW(), NOW())
	`, tableName)

	_, err := db.Exec(query, itemID, listID, content)
	return err
}


