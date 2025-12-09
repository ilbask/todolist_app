package repository

import (
	"database/sql"
	"fmt"
	"log"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/pkg/uid"
)

type shardedTodoRepoV2 struct {
	router    *sharding.RouterV2
	snowflake *uid.Snowflake
}

func (r *shardedTodoRepoV2) logSQL(action, table string, route *sharding.RouteInfo, query string, args ...interface{}) {
	if route != nil {
		log.Printf("🧭 [TodoRepoV2] %s cluster=%s shard=%04d table=%s sql=%s args=%v",
			action, route.ClusterID, route.LogicalShard, table, query, args)
	} else {
		log.Printf("🧭 [TodoRepoV2] %s table=%s sql=%s args=%v", action, table, query, args)
	}
}

// NewShardedTodoRepoV2 creates a sharded todo repository (v2 router-backed)
func NewShardedTodoRepoV2(router *sharding.RouterV2) (domain.TodoRepository, error) {
	sf, err := uid.NewSnowflake(2, 1)
	if err != nil {
		return nil, err
	}
	return &shardedTodoRepoV2{router: router, snowflake: sf}, nil
}

// Table Naming Strategy: todo_lists_tab_0000
func (r *shardedTodoRepoV2) getListTable(suffix int64) string {
	return fmt.Sprintf("todo_lists_tab_%04d", suffix)
}

func (r *shardedTodoRepoV2) getItemTable(suffix int64) string {
	return fmt.Sprintf("todo_items_tab_%04d", suffix)
}

func (r *shardedTodoRepoV2) getCollabTable(suffix int64) string {
	return fmt.Sprintf("list_collaborators_tab_%04d", suffix)
}

func (r *shardedTodoRepoV2) getIndexTable(suffix int64) string {
	// user_list_index_0000 (No _tab_ suffix specified in prompt for index?)
	// Prompt said: "user_list_index_0000~tuser_list_index_4096" (Wait, typo tuser?)
	// Assuming "user_list_index_0000".
	return fmt.Sprintf("user_list_index_%04d", suffix)
}

func (r *shardedTodoRepoV2) CreateList(list *domain.TodoList) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	list.ID = id

	// 1. Todo DB
	route, err := r.router.GetTodoRoute(list.ID)
	if err != nil {
		return err
	}
	db := route.DB
	suffix := route.LogicalShard

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	listTable := r.getListTable(suffix)
	query := fmt.Sprintf("INSERT INTO %s (list_id, owner_id, title) VALUES (?, ?, ?)", listTable)
	r.logSQL("CreateList", listTable, route, query, list.ID, list.OwnerID, list.Title)
	if _, err := tx.Exec(query, list.ID, list.OwnerID, list.Title); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("🗄️ list insert success: list_id=%d owner_id=%d table=%s suffix=%d", list.ID, list.OwnerID, listTable, suffix)

	// 2. Index DB
	idxRoute, err := r.router.GetIndexRoute(list.OwnerID)
	if err != nil {
		return err
	}
	idxDB := idxRoute.DB
	idxTable := idxRoute.Table

	idxQuery := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", idxTable)
	r.logSQL("CreateListIndex", idxTable, idxRoute, idxQuery, list.OwnerID, list.ID, "OWNER")
	_, err = idxDB.Exec(idxQuery, list.OwnerID, list.ID, "OWNER")
	if err != nil {
		log.Printf("❌ index insert failed: list_id=%d owner_id=%d idxTable=%s err=%v", list.ID, list.OwnerID, idxTable, err)
		// best-effort record for retry
		if recErr := r.recordIndexRetry(idxDB, idxTable, list.OwnerID, list.ID, "OWNER", err); recErr != nil {
			log.Printf("❌ failed to record index retry: list_id=%d owner_id=%d idxTable=%s err=%v", list.ID, list.OwnerID, idxTable, recErr)
		}
		return err
	}
	log.Printf("✅ index insert success: list_id=%d owner_id=%d idxTable=%s", list.ID, list.OwnerID, idxTable)
	return nil
}

func (r *shardedTodoRepoV2) recordIndexRetry(db *sql.DB, targetTable string, ownerID, listID int64, role string, cause error) error {
	if err := ensureIndexRetryTable(db); err != nil {
		return err
	}
	errMsg := ""
	if cause != nil {
		errMsg = cause.Error()
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
	}
	stmt := `
INSERT INTO user_list_index_retry (user_id, list_id, role, target_table, err_msg, retries)
VALUES (?, ?, ?, ?, ?, 0)
`
	log.Printf("🧭 [TodoRepoV2] RecordIndexRetry table=user_list_index_retry sql=%s args=%v", stmt, []interface{}{ownerID, listID, role, targetTable, errMsg})
	_, err := db.Exec(stmt, ownerID, listID, role, targetTable, errMsg)
	return err
}

func ensureIndexRetryTable(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS user_list_index_retry (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  user_id BIGINT NOT NULL,
  list_id BIGINT NOT NULL,
  role VARCHAR(32) NOT NULL,
  target_table VARCHAR(64) NOT NULL,
  err_msg TEXT,
  retries INT NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_user (user_id),
  KEY idx_list (list_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`
	// ensure only once per connection; use Exec
	log.Printf("🧭 [TodoRepoV2] EnsureIndexRetryTable sql=%s", ddl)
	_, err := db.Exec(ddl)
	return err
}

func (r *shardedTodoRepoV2) GetListsByUserID(userID int64) ([]domain.TodoList, error) {
	// 1. Index DB
	idxRoute, err := r.router.GetIndexRoute(userID)
	if err != nil {
		return nil, err
	}
	idxDB := idxRoute.DB
	idxTable := idxRoute.Table

	query := fmt.Sprintf("SELECT list_id, role FROM %s WHERE user_id = ?", idxTable)
	r.logSQL("ListIndexQuery", idxTable, idxRoute, query, userID)
	rows, err := idxDB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type listRef struct {
		ID   int64
		Role string
	}
	var refs []listRef
	for rows.Next() {
		var ref listRef
		if err := rows.Scan(&ref.ID, &ref.Role); err == nil {
			refs = append(refs, ref)
		}
	}

	// 2. Todo DBs
	var lists []domain.TodoList
	for _, ref := range refs {
		route, _ := r.router.GetTodoRoute(ref.ID)
		db := route.DB
		suffix := route.LogicalShard
		table := r.getListTable(suffix)

		listQuery := fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table)
		r.logSQL("FetchList", table, route, listQuery, ref.ID)
		var l domain.TodoList
		err := db.QueryRow(listQuery, ref.ID).
			Scan(&l.ID, &l.Title, &l.OwnerID)
		if err == nil {
			l.Role = domain.Role(ref.Role)
			lists = append(lists, l)
		}
	}
	return lists, nil
}

func (r *shardedTodoRepoV2) GetListByID(listID int64) (*domain.TodoList, error) {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return nil, err
	}
	db := route.DB
	table := r.getListTable(route.LogicalShard)

	l := &domain.TodoList{}
	query := fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table)
	r.logSQL("GetListByID", table, route, query, listID)
	err = db.QueryRow(query, listID).
		Scan(&l.ID, &l.Title, &l.OwnerID)
	return l, err
}

func (r *shardedTodoRepoV2) DeleteList(listID int64) error {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return err
	}
	db := route.DB
	table := r.getListTable(route.LogicalShard)
	query := fmt.Sprintf("DELETE FROM %s WHERE list_id = ?", table)
	r.logSQL("DeleteList", table, route, query, listID)
	_, err = db.Exec(query, listID)
	return err
}

func (r *shardedTodoRepoV2) AddCollaborator(listID, userID int64, role domain.Role) error {
	// 1. Collab Table
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return err
	}
	db := route.DB
	collabTable := r.getCollabTable(route.LogicalShard)

	query := fmt.Sprintf("INSERT INTO %s (list_id, user_id, role) VALUES (?, ?, ?)", collabTable)
	r.logSQL("AddCollaborator", collabTable, route, query, listID, userID, role)
	if _, err := db.Exec(query, listID, userID, role); err != nil {
		return err
	}

	// 2. Index DB
	idxRoute, err := r.router.GetIndexRoute(userID)
	if err != nil {
		return err
	}
	idxDB := idxRoute.DB
	idxTable := idxRoute.Table

	idxQuery := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", idxTable)
	r.logSQL("AddCollaboratorIndex", idxTable, idxRoute, idxQuery, userID, listID, role)
	_, err = idxDB.Exec(idxQuery, userID, listID, role)
	return err
}

func (r *shardedTodoRepoV2) CreateItem(item *domain.TodoItem) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		log.Printf("❌ [TodoRepoV2] Snowflake NextID failed for list=%d err=%v", item.ListID, err)
		return err
	}
	item.ID = id

	route, err := r.router.GetTodoRoute(item.ListID)
	if err != nil {
		log.Printf("❌ [TodoRepoV2] routing failed for list=%d err=%v", item.ListID, err)
		return err
	}
	db := route.DB
	table := r.getItemTable(route.LogicalShard)

	// 使用扩展字段（向后兼容）
	name := item.Name
	if name == "" && item.Content != "" {
		name = item.Content // 向后兼容
	}

	status := item.Status
	if status == "" {
		status = domain.StatusNotStarted
	}

	priority := item.Priority
	if priority == "" {
		priority = domain.PriorityMedium
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (item_id, list_id, content, name, description, status, priority, due_date, tags, is_done) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, table)

	r.logSQL("CreateItem", table, route, query, item.ID, item.ListID, item.Content, name, item.Description, status, priority, item.DueDate, item.Tags, item.IsDone)
	_, err = db.Exec(query,
		item.ID,
		item.ListID,
		item.Content,
		name,
		item.Description,
		status,
		priority,
		item.DueDate,
		item.Tags,
		item.IsDone,
	)
	return err
}

func (r *shardedTodoRepoV2) GetItemsByListID(listID int64) ([]domain.TodoItem, error) {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return nil, err
	}
	db := route.DB
	table := r.getItemTable(route.LogicalShard)

	query := fmt.Sprintf(`
		SELECT item_id, list_id, content, name, description, status, priority, due_date, tags, is_done, created_at, updated_at
		FROM %s 
		WHERE list_id = ?
		ORDER BY created_at DESC
	`, table)

	r.logSQL("GetItemsByListID", table, route, query, listID)
	rows, err := db.Query(query, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TodoItem
	for rows.Next() {
		var i domain.TodoItem
		err := rows.Scan(&i.ID, &i.ListID, &i.Content, &i.Name, &i.Description, &i.Status, &i.Priority, &i.DueDate, &i.Tags, &i.IsDone, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			continue
		}
		items = append(items, i)
	}
	return items, nil
}

func (r *shardedTodoRepoV2) UpdateItem(item *domain.TodoItem) error {
	return fmt.Errorf("use UpdateItemWithListID")
}

func (r *shardedTodoRepoV2) UpdateItemWithListID(listID int64, item *domain.TodoItem) error {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return err
	}
	db := route.DB
	table := r.getItemTable(route.LogicalShard)

	query := fmt.Sprintf(`
		UPDATE %s 
		SET name = ?, description = ?, status = ?, priority = ?, due_date = ?, tags = ?, is_done = ?, updated_at = CURRENT_TIMESTAMP
		WHERE item_id = ?
	`, table)

	r.logSQL("UpdateItem", table, route, query, item.Name, item.Description, item.Status, item.Priority, item.DueDate, item.Tags, item.IsDone, item.ID)
	_, err = db.Exec(query,
		item.Name,
		item.Description,
		item.Status,
		item.Priority,
		item.DueDate,
		item.Tags,
		item.IsDone,
		item.ID,
	)
	return err
}

func (r *shardedTodoRepoV2) DeleteItem(itemID int64) error {
	return fmt.Errorf("use DeleteItemWithListID")
}

func (r *shardedTodoRepoV2) DeleteItemWithListID(listID, itemID int64) error {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return err
	}
	db := route.DB
	table := r.getItemTable(route.LogicalShard)
	query := fmt.Sprintf("DELETE FROM %s WHERE item_id = ?", table)
	r.logSQL("DeleteItem", table, route, query, itemID)
	_, err = db.Exec(query, itemID)
	return err
}

// GetItemsByListIDWithFilter 根据筛选条件和排序获取items
func (r *shardedTodoRepoV2) GetItemsByListIDWithFilter(listID int64, filter *domain.ItemFilter, sort *domain.ItemSort) ([]domain.TodoItem, error) {
	route, err := r.router.GetTodoRoute(listID)
	if err != nil {
		return nil, err
	}
	db := route.DB
	table := r.getItemTable(route.LogicalShard)

	// 构建SQL查询
	query := fmt.Sprintf(`
		SELECT item_id, list_id, content, name, description, status, priority, due_date, tags, is_done, created_at, updated_at
		FROM %s 
		WHERE list_id = ?
	`, table)

	args := []interface{}{listID}

	// 添加筛选条件
	if filter != nil {
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}
		if filter.Priority != nil {
			query += " AND priority = ?"
			args = append(args, *filter.Priority)
		}
		if filter.DueBefore != nil {
			query += " AND due_date < ?"
			args = append(args, *filter.DueBefore)
		}
		if filter.DueAfter != nil {
			query += " AND due_date > ?"
			args = append(args, *filter.DueAfter)
		}
		if len(filter.Tags) > 0 {
			// 简单实现：tags包含任意一个标签
			query += " AND ("
			for i, tag := range filter.Tags {
				if i > 0 {
					query += " OR "
				}
				query += "tags LIKE ?"
				args = append(args, "%"+tag+"%")
			}
			query += ")"
		}
	}

	// 添加排序
	if sort != nil && sort.Field != "" {
		switch sort.Field {
		case "due_date", "priority", "status", "name", "created_at":
			query += fmt.Sprintf(" ORDER BY %s", sort.Field)
			if sort.Desc {
				query += " DESC"
			} else {
				query += " ASC"
			}
		default:
			query += " ORDER BY created_at DESC" // 默认排序
		}
	} else {
		query += " ORDER BY created_at DESC" // 默认排序
	}

	r.logSQL("GetItemsFiltered", table, route, query, args...)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TodoItem
	for rows.Next() {
		var i domain.TodoItem
		err := rows.Scan(&i.ID, &i.ListID, &i.Content, &i.Name, &i.Description, &i.Status, &i.Priority, &i.DueDate, &i.Tags, &i.IsDone, &i.CreatedAt, &i.UpdatedAt)
		if err != nil {
			continue
		}
		items = append(items, i)
	}
	return items, nil
}
