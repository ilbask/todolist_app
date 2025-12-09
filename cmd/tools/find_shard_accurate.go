package main

import (
	"flag"
	"fmt"
	"hash/crc32"
	"sort"
	"strconv"
)

var (
	userID = flag.Int64("user", 0, "User ID to lookup")
	listID = flag.Int64("list", 0, "List ID to lookup")
)

const (
	UserLogicalShards = 1024
	TodoLogicalShards = 4096
	
	UserPhysicalDBs = 16
	TodoPhysicalDBs = 64
	
	Replicas = 50 // Same as in router_v2.go
)

// Simple consistent hash implementation
type ConsistentHash struct {
	replicas int
	keys     []int
	hashMap  map[int]string
}

func NewConsistentHash(replicas int) *ConsistentHash {
	return &ConsistentHash{
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
}

func (c *ConsistentHash) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < c.replicas; i++ {
			hash := int(crc32.ChecksumIEEE([]byte(strconv.Itoa(i) + node)))
			c.keys = append(c.keys, hash)
			c.hashMap[hash] = node
		}
	}
	sort.Ints(c.keys)
}

func (c *ConsistentHash) Get(key string) string {
	if len(c.keys) == 0 {
		return ""
	}
	hash := int(crc32.ChecksumIEEE([]byte(key)))
	idx := sort.Search(len(c.keys), func(i int) bool {
		return c.keys[i] >= hash
	})
	if idx == len(c.keys) {
		idx = 0
	}
	return c.hashMap[c.keys[idx]]
}

func main() {
	flag.Parse()

	if *userID == 0 && *listID == 0 {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/tools/find_shard_accurate.go -user=388711261560508400")
		fmt.Println("  go run cmd/tools/find_shard_accurate.go -list=123456789")
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
	// Step 1: Calculate logical shard ID
	logicalShardID := userID % int64(UserLogicalShards)
	
	// Step 2: Initialize consistent hash ring with all user DBs
	ring := NewConsistentHash(Replicas)
	for i := 0; i < UserPhysicalDBs; i++ {
		ring.Add(fmt.Sprintf("todo_user_db_%d", i))
	}
	
	// Step 3: Use consistent hash to find physical DB
	dbName := ring.Get(fmt.Sprintf("%d", logicalShardID))
	
	tableName := fmt.Sprintf("users_%04d", logicalShardID)
	indexTableName := fmt.Sprintf("user_list_index_%04d", logicalShardID)
	
	fmt.Println("==========================================")
	fmt.Printf("   User ID: %d\n", userID)
	fmt.Println("==========================================")
	fmt.Printf("Logical Shard:  %d (of %d)\n", logicalShardID, UserLogicalShards)
	fmt.Printf("Physical DB:    %s (via consistent hash)\n", dbName)
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
	fmt.Println("💡 Note: Uses consistent hashing (CRC32) like the actual system")
}

func findListShard(listID int64) {
	// Step 1: Calculate logical shard ID
	logicalShardID := listID % int64(TodoLogicalShards)
	
	// Step 2: Initialize consistent hash ring with all data DBs
	ring := NewConsistentHash(Replicas)
	for i := 0; i < TodoPhysicalDBs; i++ {
		ring.Add(fmt.Sprintf("todo_data_db_%d", i))
	}
	
	// Step 3: Use consistent hash to find physical DB
	dbName := ring.Get(fmt.Sprintf("%d", logicalShardID))
	
	listTableName := fmt.Sprintf("todo_lists_tab_%04d", logicalShardID)
	itemTableName := fmt.Sprintf("todo_items_tab_%04d", logicalShardID)
	collabTableName := fmt.Sprintf("list_collaborators_tab_%04d", logicalShardID)
	
	fmt.Println("==========================================")
	fmt.Printf("   List ID: %d\n", listID)
	fmt.Println("==========================================")
	fmt.Printf("Logical Shard:  %d (of %d)\n", logicalShardID, TodoLogicalShards)
	fmt.Printf("Physical DB:    %s (via consistent hash)\n", dbName)
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
	fmt.Println("💡 Note: Uses consistent hashing (CRC32) like the actual system")
}

