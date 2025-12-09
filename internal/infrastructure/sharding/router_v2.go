package sharding

import (
	"database/sql"
	"fmt"
	"sync"
	"todolist-app/internal/pkg/consistenthash"
)

// DBCluster represents a physical database instance/schema
type DBCluster struct {
	ID   string
	DB   *sql.DB
}

type RouterV2 struct {
	UserLogicalShards int
	TodoLogicalShards int
	
	// Rings for routing
	UserRing  *consistenthash.Map
	TodoRing  *consistenthash.Map
	// Index tables are colocated with User tables (routed by UserID)
	// So we reuse UserRing for Index routing.
	
	Clusters map[string]*DBCluster
	mu       sync.RWMutex
}

func NewRouterV2(userShards, todoShards int) *RouterV2 {
	return &RouterV2{
		UserLogicalShards: userShards,
		TodoLogicalShards: todoShards,
		UserRing:          consistenthash.New(50, nil),
		TodoRing:          consistenthash.New(50, nil),
		Clusters:          make(map[string]*DBCluster),
	}
}

// RegisterCluster adds a physical DB to the router
func (r *RouterV2) RegisterCluster(id string, db *sql.DB, isUserDB, isTodoDB bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.Clusters[id] = &DBCluster{ID: id, DB: db}
	
	if isUserDB {
		r.UserRing.Add(id)
	}
	if isTodoDB {
		r.TodoRing.Add(id)
	}
}

// GetUserDB returns the physical DB and logical table name for a UserID
// Also used for User-List Index (colocated)
func (r *RouterV2) GetUserDB(userID int64) (*sql.DB, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logicalShardID := userID % int64(r.UserLogicalShards)
	
	// Consistent Hash Lookup
	nodeKey := r.UserRing.Get(fmt.Sprintf("%d", logicalShardID))
	
	cluster, ok := r.Clusters[nodeKey]
	if !ok {
		return nil, "", fmt.Errorf("no database cluster found for user shard %d", logicalShardID)
	}

	tableName := fmt.Sprintf("users_%04d", logicalShardID)
	return cluster.DB, tableName, nil
}

// GetIndexDB returns physical DB and logical table name for User-List Index
// Routing Key: UserID
func (r *RouterV2) GetIndexDB(userID int64) (*sql.DB, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logicalShardID := userID % int64(r.UserLogicalShards)
	nodeKey := r.UserRing.Get(fmt.Sprintf("%d", logicalShardID))
	
	cluster, ok := r.Clusters[nodeKey]
	if !ok {
		return nil, "", fmt.Errorf("no database cluster found for index shard %d", logicalShardID)
	}

	tableName := fmt.Sprintf("user_list_index_%04d", logicalShardID)
	return cluster.DB, tableName, nil
}

// GetTodoDB returns physical DB and logical table suffix for a ListID
func (r *RouterV2) GetTodoDB(listID int64) (*sql.DB, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	logicalShardID := listID % int64(r.TodoLogicalShards)
	nodeKey := r.TodoRing.Get(fmt.Sprintf("%d", logicalShardID))
	
	cluster, ok := r.Clusters[nodeKey]
	if !ok {
		return nil, 0, fmt.Errorf("no database cluster found for list shard %d", logicalShardID)
	}

	return cluster.DB, logicalShardID, nil
}


