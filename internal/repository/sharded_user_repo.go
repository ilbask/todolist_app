package repository

import (
	"database/sql"
	"fmt"
	"todolist-app/internal/domain"
	"todolist-app/internal/pkg/uid"
)

const (
	UserShards = 4 // Simulating 1024 with 4 for demo
)

type shardedUserRepo struct {
	db        *sql.DB
	snowflake *uid.Snowflake
}

func NewShardedUserRepo(db *sql.DB) (domain.UserRepository, error) {
	// WorkerID 1 for User Repo
	sf, err := uid.NewSnowflake(1, 1)
	if err != nil {
		return nil, err
	}
	return &shardedUserRepo{db: db, snowflake: sf}, nil
}

func (r *shardedUserRepo) getTableName(userID int64) string {
	return fmt.Sprintf("users_%02d", userID%UserShards)
}

// Determine shard for email is tricky without a lookup table or consistent hashing.
// For this demo, we will SCAN all shards for GetByEmail (Not efficient for prod, but standard fallback).
// In prod: maintain Email -> UserID mapping table (Global Index).
func (r *shardedUserRepo) getAllTables() []string {
	var tables []string
	for i := 0; i < UserShards; i++ {
		tables = append(tables, fmt.Sprintf("users_%02d", i))
	}
	return tables
}

func (r *shardedUserRepo) Create(u *domain.User) error {
	// Generate ID first
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	u.ID = id

	tableName := r.getTableName(u.ID)
	query := fmt.Sprintf("INSERT INTO %s (user_id, email, password_hash, verification_code, is_verified) VALUES (?, ?, ?, ?, ?)", tableName)
	
	_, err = r.db.Exec(query, u.ID, u.Email, u.PasswordHash, u.VerificationCode, u.IsVerified)
	return err
}

func (r *shardedUserRepo) GetByEmail(email string) (*domain.User, error) {
	// Scatter-Gather: Check all shards (or use Global Index in prod)
	for _, table := range r.getAllTables() {
		u := &domain.User{}
		query := fmt.Sprintf("SELECT user_id, email, password_hash, verification_code, is_verified FROM %s WHERE email = ?", table)
		err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.VerificationCode, &u.IsVerified)
		if err == nil {
			return u, nil // Found
		}
	}
	return nil, nil // Not found in any shard
}

func (r *shardedUserRepo) GetByID(id int64) (*domain.User, error) {
	u := &domain.User{}
	tableName := r.getTableName(id)
	query := fmt.Sprintf("SELECT user_id, email, is_verified FROM %s WHERE user_id = ?", tableName)
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.Email, &u.IsVerified)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *shardedUserRepo) UpdateVerification(email string, isVerified bool) error {
	// We need UserID to find shard, but we only have Email.
	// Reuse GetByEmail logic to find user first.
	user, err := r.GetByEmail(email)
	if err != nil || user == nil {
		return fmt.Errorf("user not found for update")
	}

	tableName := r.getTableName(user.ID)
	query := fmt.Sprintf("UPDATE %s SET is_verified = ? WHERE user_id = ?", tableName)
	_, err = r.db.Exec(query, isVerified, user.ID)
	return err
}


