package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// 预期配置
const (
	ExpectedUserDBs   = 16
	ExpectedDataDBs   = 64
	TablesPerUserDB   = 64
	TablesPerDataDB   = 64
	
	ExpectedUserTables        = 1024  // 16 * 64
	ExpectedIndexTables       = 1024  // 16 * 64
	ExpectedListsTables       = 4096  // 64 * 64
	ExpectedItemsTables       = 4096  // 64 * 64
	ExpectedCollabTables      = 4096  // 64 * 64
)

type CheckResult struct {
	Name     string
	Expected int
	Actual   int
	Status   string
}

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
	fmt.Println("   分库分表完整性检查")
	fmt.Println("==========================================")
	fmt.Println()

	allPassed := true
	var results []CheckResult

	// 1. 检查数据库数量
	fmt.Println("📊 第一步：检查数据库数量")
	fmt.Println()

	userDBCount := countDatabases(db, "todo_user_db_%")
	results = append(results, CheckResult{
		Name:     "User Databases (todo_user_db_*)",
		Expected: ExpectedUserDBs,
		Actual:   userDBCount,
		Status:   status(ExpectedUserDBs, userDBCount),
	})
	printResult("User DBs", ExpectedUserDBs, userDBCount)

	dataDBCount := countDatabases(db, "todo_data_db_%")
	results = append(results, CheckResult{
		Name:     "Data Databases (todo_data_db_*)",
		Expected: ExpectedDataDBs,
		Actual:   dataDBCount,
		Status:   status(ExpectedDataDBs, dataDBCount),
	})
	printResult("Data DBs", ExpectedDataDBs, dataDBCount)

	fmt.Println()

	// 2. 检查User相关表
	fmt.Println("📊 第二步：检查User数据库表")
	fmt.Println()

	usersCount := countTables(db, "todo_user_db_%", "users_%")
	results = append(results, CheckResult{
		Name:     "Users Tables (users_0000 ~ users_1023)",
		Expected: ExpectedUserTables,
		Actual:   usersCount,
		Status:   status(ExpectedUserTables, usersCount),
	})
	printResult("users_ 表", ExpectedUserTables, usersCount)

	indexCount := countTables(db, "todo_user_db_%", "user_list_index_%")
	results = append(results, CheckResult{
		Name:     "Index Tables (user_list_index_0000 ~ user_list_index_1023)",
		Expected: ExpectedIndexTables,
		Actual:   indexCount,
		Status:   status(ExpectedIndexTables, indexCount),
	})
	printResult("user_list_index_ 表", ExpectedIndexTables, indexCount)

	fmt.Println()

	// 3. 检查Data相关表
	fmt.Println("📊 第三步：检查Data数据库表")
	fmt.Println()

	listsCount := countTables(db, "todo_data_db_%", "todo_lists_tab_%")
	results = append(results, CheckResult{
		Name:     "Lists Tables (todo_lists_tab_0000 ~ todo_lists_tab_4095)",
		Expected: ExpectedListsTables,
		Actual:   listsCount,
		Status:   status(ExpectedListsTables, listsCount),
	})
	printResult("todo_lists_tab_ 表", ExpectedListsTables, listsCount)

	itemsCount := countTables(db, "todo_data_db_%", "todo_items_tab_%")
	results = append(results, CheckResult{
		Name:     "Items Tables (todo_items_tab_0000 ~ todo_items_tab_4095)",
		Expected: ExpectedItemsTables,
		Actual:   itemsCount,
		Status:   status(ExpectedItemsTables, itemsCount),
	})
	printResult("todo_items_tab_ 表", ExpectedItemsTables, itemsCount)

	collabCount := countTables(db, "todo_data_db_%", "list_collaborators_tab_%")
	results = append(results, CheckResult{
		Name:     "Collaborators Tables (list_collaborators_tab_0000 ~ list_collaborators_tab_4095)",
		Expected: ExpectedCollabTables,
		Actual:   collabCount,
		Status:   status(ExpectedCollabTables, collabCount),
	})
	printResult("list_collaborators_tab_ 表", ExpectedCollabTables, collabCount)

	fmt.Println()

	// 4. 检查每个DB的表分布
	fmt.Println("📊 第四步：检查表分布均匀性")
	fmt.Println()

	fmt.Println("User DBs 表分布:")
	checkUserDBDistribution(db)

	fmt.Println()
	fmt.Println("Data DBs 表分布（抽样检查前3个和后3个）:")
	checkDataDBDistribution(db)

	fmt.Println()

	// 5. 生成总结报告
	fmt.Println("==========================================")
	fmt.Println("   检查结果汇总")
	fmt.Println("==========================================")
	fmt.Println()

	for _, r := range results {
		fmt.Printf("%s %s\n", r.Status, r.Name)
		if r.Expected != r.Actual {
			fmt.Printf("   预期: %d, 实际: %d, 差异: %d\n", r.Expected, r.Actual, r.Actual-r.Expected)
			allPassed = false
		}
	}

	fmt.Println()
	fmt.Println("==========================================")

	if allPassed {
		fmt.Println("✅ 所有检查通过！分库分表配置完全符合预期")
		fmt.Println()
		fmt.Println("分片总览:")
		fmt.Printf("  • User DBs: %d 个数据库\n", ExpectedUserDBs)
		fmt.Printf("  • Data DBs: %d 个数据库\n", ExpectedDataDBs)
		fmt.Printf("  • Users表: %d 张 (分布在 %d 个DB)\n", ExpectedUserTables, ExpectedUserDBs)
		fmt.Printf("  • Index表: %d 张 (分布在 %d 个DB)\n", ExpectedIndexTables, ExpectedUserDBs)
		fmt.Printf("  • Lists表: %d 张 (分布在 %d 个DB)\n", ExpectedListsTables, ExpectedDataDBs)
		fmt.Printf("  • Items表: %d 张 (分布在 %d 个DB)\n", ExpectedItemsTables, ExpectedDataDBs)
		fmt.Printf("  • Collab表: %d 张 (分布在 %d 个DB)\n", ExpectedCollabTables, ExpectedDataDBs)
		fmt.Printf("  • 总表数: %d 张\n", ExpectedUserTables+ExpectedIndexTables+ExpectedListsTables+ExpectedItemsTables+ExpectedCollabTables)
	} else {
		fmt.Println("❌ 检查失败！存在表缺失或数量不符")
		fmt.Println()
		fmt.Println("修复建议:")
		fmt.Println("  1. 运行: go run cmd/tools/fix_missing_tables.go")
		fmt.Println("  2. 或重新初始化: go run cmd/tools/cleanup_db.go && go run cmd/tools/init_sharding_v6.go")
	}

	fmt.Println("==========================================")

	if !allPassed {
		os.Exit(1)
	}
}

func countDatabases(db *sql.DB, pattern string) int {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM information_schema.SCHEMATA 
		WHERE SCHEMA_NAME LIKE '%s'
	`, pattern)

	var count int
	db.QueryRow(query).Scan(&count)
	return count
}

func countTables(db *sql.DB, dbPattern, tablePattern string) int {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA LIKE '%s' 
		AND TABLE_NAME LIKE '%s'
	`, dbPattern, tablePattern)

	var count int
	db.QueryRow(query).Scan(&count)
	return count
}

func status(expected, actual int) string {
	if expected == actual {
		return "✅"
	}
	return "❌"
}

func printResult(name string, expected, actual int) {
	statusIcon := "✅"
	if expected != actual {
		statusIcon = "❌"
	}
	fmt.Printf("  %s %-30s 预期: %4d, 实际: %4d", statusIcon, name, expected, actual)
	if expected != actual {
		diff := actual - expected
		if diff > 0 {
			fmt.Printf(" (多 %d)", diff)
		} else {
			fmt.Printf(" (少 %d)", -diff)
		}
	}
	fmt.Println()
}

func checkUserDBDistribution(db *sql.DB) {
	for i := 0; i < ExpectedUserDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		
		usersCount := countTablesInDB(db, dbName, "users_%")
		indexCount := countTablesInDB(db, dbName, "user_list_index_%")
		
		statusIcon := "✅"
		if usersCount != TablesPerUserDB || indexCount != TablesPerUserDB {
			statusIcon = "❌"
		}
		
		fmt.Printf("  %s %s: users=%d, index=%d", statusIcon, dbName, usersCount, indexCount)
		if usersCount != TablesPerUserDB || indexCount != TablesPerUserDB {
			fmt.Printf(" (预期各 %d 张)", TablesPerUserDB)
		}
		fmt.Println()
	}
}

func checkDataDBDistribution(db *sql.DB) {
	// 检查前3个
	for i := 0; i < 3; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		printDataDBInfo(db, dbName)
	}
	
	fmt.Println("  ...")
	
	// 检查后3个
	for i := ExpectedDataDBs - 3; i < ExpectedDataDBs; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		printDataDBInfo(db, dbName)
	}
}

func printDataDBInfo(db *sql.DB, dbName string) {
	listsCount := countTablesInDB(db, dbName, "todo_lists_tab_%")
	itemsCount := countTablesInDB(db, dbName, "todo_items_tab_%")
	collabCount := countTablesInDB(db, dbName, "list_collaborators_tab_%")
	
	statusIcon := "✅"
	if listsCount != TablesPerDataDB || itemsCount != TablesPerDataDB || collabCount != TablesPerDataDB {
		statusIcon = "❌"
	}
	
	fmt.Printf("  %s %s: lists=%d, items=%d, collab=%d", statusIcon, dbName, listsCount, itemsCount, collabCount)
	if listsCount != TablesPerDataDB || itemsCount != TablesPerDataDB || collabCount != TablesPerDataDB {
		fmt.Printf(" (预期各 %d 张)", TablesPerDataDB)
	}
	fmt.Println()
}

func countTablesInDB(db *sql.DB, dbName, tablePattern string) int {
	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM information_schema.TABLES 
		WHERE TABLE_SCHEMA = '%s' 
		AND TABLE_NAME LIKE '%s'
	`, dbName, tablePattern)

	var count int
	db.QueryRow(query).Scan(&count)
	return count
}

