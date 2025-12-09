package repository

import (
	"fmt"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/pkg/uid"
)

type shardedUserRepoV2 struct {
	router    *sharding.RouterV2
	snowflake *uid.Snowflake
}

func NewShardedUserRepoV2(router *sharding.RouterV2) (domain.UserRepository, error) {
	sf, err := uid.NewSnowflake(1, 1)
	if err != nil {
		return nil, err
	}
	return &shardedUserRepoV2{router: router, snowflake: sf}, nil
}

func (r *shardedUserRepoV2) getTableName(id int64) string {
	// Table: users_0000 ... users_1023
	// Using sharding logic: ID % 1024
	return fmt.Sprintf("users_%04d", id%int64(r.router.UserLogicalShards))
}

func (r *shardedUserRepoV2) Create(u *domain.User) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	u.ID = id

	db, _, err := r.router.GetUserDB(u.ID)
	if err != nil {
		return err
	}

	tableName := r.getTableName(u.ID)
	query := fmt.Sprintf("INSERT INTO %s (user_id, email, password_hash, verification_code, is_verified) VALUES (?, ?, ?, ?, ?)", tableName)
	_, err = db.Exec(query, u.ID, u.Email, u.PasswordHash, u.VerificationCode, u.IsVerified)
	return err
}

func (r *shardedUserRepoV2) GetByEmail(email string) (*domain.User, error) {
	// Full Scan across all logical shards
	for i := 0; i < r.router.UserLogicalShards; i++ {
		db, _, _ := r.router.GetUserDB(int64(i))
		tableName := fmt.Sprintf("users_%04d", i)
		
		u := &domain.User{}
		query := fmt.Sprintf("SELECT user_id, email, password_hash, verification_code, is_verified FROM %s WHERE email = ?", tableName)
		err := db.QueryRow(query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.VerificationCode, &u.IsVerified)
		if err == nil {
			return u, nil
		}
	}
	return nil, nil
}

func (r *shardedUserRepoV2) GetByID(id int64) (*domain.User, error) {
	db, _, err := r.router.GetUserDB(id)
	if err != nil {
		return nil, err
	}

	tableName := r.getTableName(id)
	u := &domain.User{}
	query := fmt.Sprintf("SELECT user_id, email, is_verified FROM %s WHERE user_id = ?", tableName)
	err = db.QueryRow(query, id).Scan(&u.ID, &u.Email, &u.IsVerified)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *shardedUserRepoV2) UpdateVerification(email string, isVerified bool) error {
	user, err := r.GetByEmail(email)
	if err != nil || user == nil {
		return fmt.Errorf("user not found for update")
	}

	db, _, err := r.router.GetUserDB(user.ID)
	if err != nil {
		return err
	}

	tableName := r.getTableName(user.ID)
	query := fmt.Sprintf("UPDATE %s SET is_verified = ? WHERE user_id = ?", tableName)
	_, err = db.Exec(query, isVerified, user.ID)
	return err
}
