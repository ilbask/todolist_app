package repository

import (
	"fmt"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/pkg/uid"
)

type shardedTodoRepoV2 struct {
	router    *sharding.RouterV2
	snowflake *uid.Snowflake
}

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
	db, suffix, err := r.router.GetTodoDB(list.ID)
	if err != nil {
		return err
	}
	
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	listTable := r.getListTable(suffix)
	query := fmt.Sprintf("INSERT INTO %s (list_id, owner_id, title) VALUES (?, ?, ?)", listTable)
	if _, err := tx.Exec(query, list.ID, list.OwnerID, list.Title); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 2. Index DB
	idxDB, _, err := r.router.GetIndexDB(list.OwnerID)
	if err != nil {
		return err
	}
	// Index table suffix derived from UserID
	idxSuffix := list.OwnerID % int64(r.router.UserLogicalShards)
	idxTable := r.getIndexTable(idxSuffix)
	
	idxQuery := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", idxTable)
	_, err = idxDB.Exec(idxQuery, list.OwnerID, list.ID, "OWNER")
	return err
}

func (r *shardedTodoRepoV2) GetListsByUserID(userID int64) ([]domain.TodoList, error) {
	// 1. Index DB
	idxDB, _, err := r.router.GetIndexDB(userID)
	if err != nil {
		return nil, err
	}
	idxSuffix := userID % int64(r.router.UserLogicalShards)
	idxTable := r.getIndexTable(idxSuffix)

	rows, err := idxDB.Query(fmt.Sprintf("SELECT list_id, role FROM %s WHERE user_id = ?", idxTable), userID)
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
		db, suffix, _ := r.router.GetTodoDB(ref.ID)
		table := r.getListTable(suffix)
		
		var l domain.TodoList
		err := db.QueryRow(fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table), ref.ID).
			Scan(&l.ID, &l.Title, &l.OwnerID)
		if err == nil {
			l.Role = domain.Role(ref.Role)
			lists = append(lists, l)
		}
	}
	return lists, nil
}

func (r *shardedTodoRepoV2) GetListByID(listID int64) (*domain.TodoList, error) {
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return nil, err
	}
	table := r.getListTable(suffix)
	
	l := &domain.TodoList{}
	err = db.QueryRow(fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table), listID).
		Scan(&l.ID, &l.Title, &l.OwnerID)
	return l, err
}

func (r *shardedTodoRepoV2) DeleteList(listID int64) error {
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return err
	}
	table := r.getListTable(suffix)
	_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE list_id = ?", table), listID)
	return err
}

func (r *shardedTodoRepoV2) AddCollaborator(listID, userID int64, role domain.Role) error {
	// 1. Collab Table
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return err
	}
	collabTable := r.getCollabTable(suffix)
	
	if _, err := db.Exec(fmt.Sprintf("INSERT INTO %s (list_id, user_id, role) VALUES (?, ?, ?)", collabTable), listID, userID, role); err != nil {
		return err
	}

	// 2. Index DB
	idxDB, _, err := r.router.GetIndexDB(userID)
	if err != nil {
		return err
	}
	idxSuffix := userID % int64(r.router.UserLogicalShards)
	idxTable := r.getIndexTable(idxSuffix)
	
	_, err = idxDB.Exec(fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", idxTable), userID, listID, role)
	return err
}

func (r *shardedTodoRepoV2) CreateItem(item *domain.TodoItem) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	item.ID = id

	db, suffix, err := r.router.GetTodoDB(item.ListID)
	if err != nil {
		return err
	}
	table := r.getItemTable(suffix)
	
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
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return nil, err
	}
	table := r.getItemTable(suffix)

	query := fmt.Sprintf(`
		SELECT item_id, list_id, content, name, description, status, priority, due_date, tags, is_done, created_at, updated_at
		FROM %s 
		WHERE list_id = ?
		ORDER BY created_at DESC
	`, table)
	
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
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return err
	}
	table := r.getItemTable(suffix)
	
	query := fmt.Sprintf(`
		UPDATE %s 
		SET name = ?, description = ?, status = ?, priority = ?, due_date = ?, tags = ?, is_done = ?, updated_at = CURRENT_TIMESTAMP
		WHERE item_id = ?
	`, table)
	
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
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return err
	}
	table := r.getItemTable(suffix)
	_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE item_id = ?", table), itemID)
	return err
}

// GetItemsByListIDWithFilter 根据筛选条件和排序获取items
func (r *shardedTodoRepoV2) GetItemsByListIDWithFilter(listID int64, filter *domain.ItemFilter, sort *domain.ItemSort) ([]domain.TodoItem, error) {
	db, suffix, err := r.router.GetTodoDB(listID)
	if err != nil {
		return nil, err
	}
	table := r.getItemTable(suffix)

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
