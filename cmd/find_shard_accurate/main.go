package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"hash/crc32"
)

var (
	userID = flag.Int64("user", 0, "User ID to lookup")
	listID = flag.Int64("list", 0, "List ID to lookup")
)

const (
	UserPhysicalDBs = 16
	UserTablesPerDB = 64
	TodoPhysicalDBs = 64
	TodoTablesPerDB = 64
)

func main() {
	flag.Parse()

	if *userID == 0 && *listID == 0 {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/find_shard_accurate -user=388711261560508400")
		fmt.Println("  go run cmd/find_shard_accurate -list=123456789")
		return
	}

	if *userID != 0 {
		findUserShard(*userID)
	}

	if *listID != 0 {
		findListShard(*listID)
	}
}

func findUserShard(userID int64) {
	dbIdx, tableIdx := locate(userID, UserPhysicalDBs, UserTablesPerDB)
	dbName := fmt.Sprintf("todo_user_db_%d", dbIdx)
	tableName := fmt.Sprintf("users_%04d", tableIdx)
	indexTableName := fmt.Sprintf("user_list_index_%04d", tableIdx)

	fmt.Println("==========================================")
	fmt.Printf("   User ID: %d\n", userID)
	fmt.Println("==========================================")
	fmt.Printf("DB Index:       %d (of %d)\n", dbIdx, UserPhysicalDBs)
	fmt.Printf("Table Index:    %d (of %d)\n", tableIdx, UserTablesPerDB)
	fmt.Printf("Physical DB:    %s (CRC32 hash %% %d)\n", dbName, UserPhysicalDBs)
	fmt.Printf("User Table:     %s\n", tableName)
	fmt.Printf("Index Table:    %s\n", indexTableName)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("📋 Query Examples:")
	fmt.Printf("  mysql> USE %s;\n", dbName)
	fmt.Printf("  mysql> SELECT * FROM %s WHERE user_id = %d;\n", tableName, userID)
	fmt.Printf("  mysql> SELECT * FROM %s WHERE user_id = %d;\n", indexTableName, userID)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("💡 Note: Uses CRC32 based db/table hashing (db = hash%%16, table = (hash/16)%%64)")
}

func findListShard(listID int64) {
	dbIdx, tableIdx := locate(listID, TodoPhysicalDBs, TodoTablesPerDB)
	dbName := fmt.Sprintf("todo_data_db_%d", dbIdx)
	listTableName := fmt.Sprintf("todo_lists_tab_%04d", tableIdx)
	itemTableName := fmt.Sprintf("todo_items_tab_%04d", tableIdx)
	collabTableName := fmt.Sprintf("list_collaborators_tab_%04d", tableIdx)

	fmt.Println("==========================================")
	fmt.Printf("   List ID: %d\n", listID)
	fmt.Println("==========================================")
	fmt.Printf("DB Index:       %d (of %d)\n", dbIdx, TodoPhysicalDBs)
	fmt.Printf("Table Index:    %d (of %d)\n", tableIdx, TodoTablesPerDB)
	fmt.Printf("Physical DB:    %s (CRC32 hash %% %d)\n", dbName, TodoPhysicalDBs)
	fmt.Printf("List Table:     %s\n", listTableName)
	fmt.Printf("Item Table:     %s\n", itemTableName)
	fmt.Printf("Collab Table:   %s\n", collabTableName)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("📋 Query Examples:")
	fmt.Printf("  mysql> USE %s;\n", dbName)
	fmt.Printf("  mysql> SELECT * FROM %s WHERE list_id = %d;\n", listTableName, listID)
	fmt.Printf("  mysql> SELECT * FROM %s WHERE list_id = %d;\n", itemTableName, listID)
	fmt.Printf("  mysql> SELECT * FROM %s WHERE list_id = %d;\n", collabTableName, listID)
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("💡 Note: Uses CRC32 based db/table hashing (db = hash%%64, table = (hash/64)%%64)")
}

func locate(id int64, dbCount, tableCount int) (int, int) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(id))
	hash := crc32.ChecksumIEEE(buf[:])
	dbIdx := int(hash % uint32(dbCount))
	tableIdx := int((hash / uint32(dbCount)) % uint32(tableCount))
	return dbIdx, tableIdx
}
