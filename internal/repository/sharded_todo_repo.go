package repository

import (
	"database/sql"
	"fmt"
	"todolist-app/internal/domain"
	"todolist-app/internal/pkg/uid"
)

const (
	ListShards = 4 // Simulating 4096
)

type shardedTodoRepo struct {
	db        *sql.DB
	snowflake *uid.Snowflake
}

func NewShardedTodoRepo(db *sql.DB) (domain.TodoRepository, error) {
	sf, err := uid.NewSnowflake(2, 1) // Worker 2
	if err != nil {
		return nil, err
	}
	return &shardedTodoRepo{db: db, snowflake: sf}, nil
}

// Routing: list_id % Shards
func (r *shardedTodoRepo) getListTable(listID int64) string {
	return fmt.Sprintf("todo_lists_%02d", listID%ListShards)
}

func (r *shardedTodoRepo) getItemTable(listID int64) string {
	return fmt.Sprintf("todo_items_%02d", listID%ListShards)
}

func (r *shardedTodoRepo) getCollabTable(listID int64) string {
	return fmt.Sprintf("list_collaborators_%02d", listID%ListShards)
}

func (r *shardedTodoRepo) getIndexTable(userID int64) string {
	// User-List Index is sharded by USER_ID
	return fmt.Sprintf("user_list_index_%02d", userID%ListShards) // Reuse same shard count for simplicity
}

func (r *shardedTodoRepo) CreateList(list *domain.TodoList) error {
	// 1. Generate List ID
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	list.ID = id

	// 2. Insert into TodoList Shard
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	listTable := r.getListTable(list.ID)
	query := fmt.Sprintf("INSERT INTO %s (list_id, owner_id, title) VALUES (?, ?, ?)", listTable)
	if _, err := tx.Exec(query, list.ID, list.OwnerID, list.Title); err != nil {
		tx.Rollback()
		return err
	}

	// 3. Insert into User-List Index (So we can find it later)
	indexTable := r.getIndexTable(list.OwnerID)
	idxQuery := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", indexTable)
	// Owner is also a collaborator/viewer
	if _, err := tx.Exec(idxQuery, list.OwnerID, list.ID, "OWNER"); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *shardedTodoRepo) GetListsByUserID(userID int64) ([]domain.TodoList, error) {
	// 1. Query Index Table to get List IDs
	indexTable := r.getIndexTable(userID)
	idxQuery := fmt.Sprintf("SELECT list_id, role FROM %s WHERE user_id = ?", indexTable)
	
	rows, err := r.db.Query(idxQuery, userID)
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

	// 2. Fetch actual lists from their respective shards
	// Note: In real world, we might parallelize this or group by shard.
	var lists []domain.TodoList
	for _, ref := range refs {
		table := r.getListTable(ref.ID)
		query := fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table)
		
		var l domain.TodoList
		err := r.db.QueryRow(query, ref.ID).Scan(&l.ID, &l.Title, &l.OwnerID)
		if err == nil {
			l.Role = domain.Role(ref.Role)
			lists = append(lists, l)
		}
	}

	return lists, nil
}

func (r *shardedTodoRepo) GetListByID(listID int64) (*domain.TodoList, error) {
	l := &domain.TodoList{}
	table := r.getListTable(listID)
	err := r.db.QueryRow(fmt.Sprintf("SELECT list_id, title, owner_id FROM %s WHERE list_id = ?", table), listID).
		Scan(&l.ID, &l.Title, &l.OwnerID)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *shardedTodoRepo) DeleteList(listID int64) error {
	// Note: We should also delete from User-List Index, but that requires knowing all users who have access.
	// For demo, we just delete the list data. Consistency would require eventual consistency job.
	table := r.getListTable(listID)
	_, err := r.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE list_id = ?", table), listID)
	return err
}

func (r *shardedTodoRepo) AddCollaborator(listID, userID int64, role domain.Role) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	// 1. Add to List Collaborators (Colocated with List)
	collabTable := r.getCollabTable(listID)
	q1 := fmt.Sprintf("INSERT INTO %s (list_id, user_id, role) VALUES (?, ?, ?)", collabTable)
	if _, err := tx.Exec(q1, listID, userID, role); err != nil {
		tx.Rollback()
		return err
	}

	// 2. Add to User-List Index (Sharded by User)
	indexTable := r.getIndexTable(userID)
	q2 := fmt.Sprintf("INSERT INTO %s (user_id, list_id, role) VALUES (?, ?, ?)", indexTable)
	if _, err := tx.Exec(q2, userID, listID, role); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *shardedTodoRepo) CreateItem(item *domain.TodoItem) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	item.ID = id
	
	table := r.getItemTable(item.ListID)
	query := fmt.Sprintf("INSERT INTO %s (item_id, list_id, content, is_done) VALUES (?, ?, ?, ?)", table)
	_, err = r.db.Exec(query, item.ID, item.ListID, item.Content, item.IsDone)
	return err
}

func (r *shardedTodoRepo) GetItemsByListID(listID int64) ([]domain.TodoItem, error) {
	table := r.getItemTable(listID)
	rows, err := r.db.Query(fmt.Sprintf("SELECT item_id, list_id, content, is_done FROM %s WHERE list_id = ?", table), listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.TodoItem
	for rows.Next() {
		var i domain.TodoItem
		if err := rows.Scan(&i.ID, &i.ListID, &i.Content, &i.IsDone); err == nil {
			items = append(items, i)
		}
	}
	return items, nil
}

func (r *shardedTodoRepo) UpdateItem(item *domain.TodoItem) error {
	// Issue: We need ListID to find the table.
	// But UpdateItem signature usually only takes ID.
	// In sharded env, we MUST provide ShardingKey (ListID) or query all shards (bad).
	// Refactoring TodoService to pass ListID is best.
	// Assume we hack it for now: try to find it? Or change Interface.
	// Let's assume we change interface or cheat.
	// Cheat: We don't know ListID from ItemID alone with Snowflake unless we embed shard info.
	// Better: Require ListID in UpdateItem.
	
	return fmt.Errorf("update item requires list_id for routing")
}

// Optimized Update with ListID
func (r *shardedTodoRepo) UpdateItemWithListID(listID int64, item *domain.TodoItem) error {
	table := r.getItemTable(listID)
	_, err := r.db.Exec(fmt.Sprintf("UPDATE %s SET is_done = ? WHERE item_id = ?", table), item.IsDone, item.ID)
	return err
}

func (r *shardedTodoRepo) DeleteItem(itemID int64) error {
     // Same issue as UpdateItem
     return fmt.Errorf("delete item requires list_id for routing")
}

func (r *shardedTodoRepo) DeleteItemWithListID(listID, itemID int64) error {
	table := r.getItemTable(listID)
	_, err := r.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE item_id = ?", table), itemID)
	return err
}


