package sharding

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"sync"
)

const (
	userTablesPerDB = 64
	todoTablesPerDB = 64
)

// DBCluster represents a physical database instance/schema
type DBCluster struct {
	ID string
	DB *sql.DB
}

// RouteInfo carries resolved shard metadata for logging/debug
type RouteInfo struct {
	DB           *sql.DB
	ClusterID    string
	DBIndex      int
	TableIndex   int
	LogicalShard int64
	Table        string
}

type RouterV2 struct {
	UserLogicalShards int
	TodoLogicalShards int

	userClusters []*DBCluster
	todoClusters []*DBCluster

	Clusters map[string]*DBCluster
	mu       sync.RWMutex
}

func NewRouterV2(userShards, todoShards int) *RouterV2 {
	return &RouterV2{
		UserLogicalShards: userShards,
		TodoLogicalShards: todoShards,
		userClusters:      make([]*DBCluster, 0),
		todoClusters:      make([]*DBCluster, 0),
		Clusters:          make(map[string]*DBCluster),
	}
}

// RegisterCluster adds a physical DB to the router
func (r *RouterV2) RegisterCluster(id string, db *sql.DB, isUserDB, isTodoDB bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	cluster := &DBCluster{ID: id, DB: db}
	r.Clusters[id] = cluster

	if isUserDB {
		if idx, ok := extractIndex(id); ok {
			r.userClusters = ensureClusterSlot(r.userClusters, idx)
			r.userClusters[idx] = cluster
		} else {
			r.userClusters = append(r.userClusters, cluster)
		}
	}
	if isTodoDB {
		if idx, ok := extractIndex(id); ok {
			r.todoClusters = ensureClusterSlot(r.todoClusters, idx)
			r.todoClusters[idx] = cluster
		} else {
			r.todoClusters = append(r.todoClusters, cluster)
		}
	}
}

// GetUserDB returns the physical DB and logical table name for a UserID
// Also used for User-List Index (colocated)
func (r *RouterV2) GetUserDB(userID int64) (*sql.DB, string, error) {
	route, err := r.GetUserRoute(userID)
	if err != nil {
		return nil, "", err
	}
	return route.DB, route.Table, nil
}

// GetUserRoute returns full routing details for a given user ID
func (r *RouterV2) GetUserRoute(userID int64) (*RouteInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hash := hashUint64(uint64(userID))
	return r.routeForHash(hash, r.userClusters, userTablesPerDB, "users_%04d")
}

// GetIndexDB returns physical DB and logical table name for User-List Index
// Routing Key: UserID
func (r *RouterV2) GetIndexDB(userID int64) (*sql.DB, string, error) {
	route, err := r.GetIndexRoute(userID)
	if err != nil {
		return nil, "", err
	}
	return route.DB, route.Table, nil
}

// GetIndexRoute returns routing metadata for the user list index
func (r *RouterV2) GetIndexRoute(userID int64) (*RouteInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hash := hashUint64(uint64(userID))
	return r.routeForHash(hash, r.userClusters, userTablesPerDB, "user_list_index_%04d")
}

// GetEmailIndexDB returns physical DB and logical table name for Email Index
// Routing Key: email (hashed)
func (r *RouterV2) GetEmailIndexDB(email string) (*sql.DB, string, error) {
	route, err := r.GetEmailIndexRoute(email)
	if err != nil {
		return nil, "", err
	}
	return route.DB, route.Table, nil
}

// GetEmailIndexRoute returns routing metadata for email index table
func (r *RouterV2) GetEmailIndexRoute(email string) (*RouteInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hash := crc32.ChecksumIEEE([]byte(email))
	return r.routeForHash(hash, r.userClusters, userTablesPerDB, "user_email_index_%04d")
}

// GetTodoDB returns physical DB and logical table suffix for a ListID
func (r *RouterV2) GetTodoDB(listID int64) (*sql.DB, int64, error) {
	route, err := r.GetTodoRoute(listID)
	if err != nil {
		return nil, 0, err
	}
	return route.DB, route.LogicalShard, nil
}

// GetTodoRoute returns routing metadata for todo shards
func (r *RouterV2) GetTodoRoute(listID int64) (*RouteInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hash := hashUint64(uint64(listID))
	return r.routeForHash(hash, r.todoClusters, todoTablesPerDB, "todo_shard_%04d")
}

func (r *RouterV2) routeForHash(hash uint32, clusters []*DBCluster, tablesPerDB int, tableFmt string) (*RouteInfo, error) {
	dbCount := len(clusters)
	if dbCount == 0 {
		return nil, errors.New("no database clusters registered")
	}

	dbIdx := int(hash % uint32(dbCount))
	cluster := clusters[dbIdx]
	if cluster == nil {
		return nil, fmt.Errorf("database cluster %d not registered", dbIdx)
	}

	tableIdx := int((hash / uint32(dbCount)) % uint32(tablesPerDB))
	tableName := fmt.Sprintf(tableFmt, tableIdx)

	return &RouteInfo{
		DB:           cluster.DB,
		ClusterID:    cluster.ID,
		DBIndex:      dbIdx,
		TableIndex:   tableIdx,
		LogicalShard: int64(tableIdx),
		Table:        tableName,
	}, nil
}

func ensureClusterSlot(src []*DBCluster, idx int) []*DBCluster {
	if idx < len(src) {
		return src
	}
	newSlice := make([]*DBCluster, idx+1)
	copy(newSlice, src)
	return newSlice
}

func extractIndex(id string) (int, bool) {
	parts := strings.Split(id, "_")
	if len(parts) == 0 {
		return 0, false
	}
	idx, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return idx, true
}

func hashUint64(v uint64) uint32 {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	return crc32.ChecksumIEEE(buf[:])
}
