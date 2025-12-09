package repository

import (
	"database/sql"
	"fmt"
	"log"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/pkg/uid"
)

type shardedUserRepoV2 struct {
	router    *sharding.RouterV2
	snowflake *uid.Snowflake
}

func (r *shardedUserRepoV2) logSQL(action, table string, route *sharding.RouteInfo, query string, args ...interface{}) {
	if route != nil {
		log.Printf("🧭 [UserRepoV2] %s cluster=%s shard=%04d table=%s sql=%s args=%v",
			action, route.ClusterID, route.LogicalShard, table, query, args)
	} else {
		log.Printf("🧭 [UserRepoV2] %s table=%s sql=%s args=%v", action, table, query, args)
	}
}

// NewShardedUserRepoV2 creates a sharded user repository (v2 router-backed)
func NewShardedUserRepoV2(router *sharding.RouterV2) (domain.UserRepository, error) {
	sf, err := uid.NewSnowflake(1, 1)
	if err != nil {
		return nil, err
	}
	return &shardedUserRepoV2{router: router, snowflake: sf}, nil
}

func (r *shardedUserRepoV2) Create(u *domain.User) error {
	id, err := r.snowflake.NextID()
	if err != nil {
		return err
	}
	u.ID = id

	route, err := r.router.GetUserRoute(u.ID)
	if err != nil {
		return err
	}
	db := route.DB
	tableName := route.Table

	log.Printf("🗄️ user shard routing: user_id=%d -> table=%s db=%p", u.ID, tableName, db)
	query := fmt.Sprintf("INSERT INTO %s (user_id, email, password_hash, verification_code, is_verified) VALUES (?, ?, ?, ?, ?)", tableName)
	r.logSQL("CreateUser", tableName, route, query, u.ID, u.Email, "***", "***", u.IsVerified)
	_, err = db.Exec(query, u.ID, u.Email, u.PasswordHash, u.VerificationCode, u.IsVerified)
	if err != nil {
		return err
	}

	// also write email -> user_id index to speed up GetByEmail
	idxRoute, err := r.router.GetEmailIndexRoute(u.Email)
	if err != nil {
		return err
	}
	idxQuery := fmt.Sprintf("INSERT INTO %s (email, user_id) VALUES (?, ?)", idxRoute.Table)
	r.logSQL("InsertEmailIndex", idxRoute.Table, idxRoute, idxQuery, u.Email, u.ID)
	if _, err := idxRoute.DB.Exec(idxQuery, u.Email, u.ID); err != nil {
		return err
	}

	return err
}

func (r *shardedUserRepoV2) GetByEmail(email string) (*domain.User, error) {
	// 1) Lookup email index to find user_id
	idxRoute, err := r.router.GetEmailIndexRoute(email)
	if err != nil {
		return nil, err
	}

	var userID int64
	query := fmt.Sprintf("SELECT user_id FROM %s WHERE email = ?", idxRoute.Table)
	r.logSQL("EmailIndexLookup", idxRoute.Table, idxRoute, query, email)
	if err := idxRoute.DB.QueryRow(query, email).Scan(&userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 2) Fetch user by ID using sharded routing
	return r.GetByID(userID)
}

func (r *shardedUserRepoV2) GetByID(id int64) (*domain.User, error) {
	route, err := r.router.GetUserRoute(id)
	if err != nil {
		return nil, err
	}

	db := route.DB
	tableName := route.Table
	u := &domain.User{}
	query := fmt.Sprintf("SELECT user_id, email, password_hash, verification_code, is_verified, created_at FROM %s WHERE user_id = ?", tableName)
	r.logSQL("GetByID", tableName, route, query, id)
	err = db.QueryRow(query, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.VerificationCode, &u.IsVerified, &u.CreatedAt)
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

	route, err := r.router.GetUserRoute(user.ID)
	if err != nil {
		return err
	}

	db := route.DB
	tableName := route.Table
	query := fmt.Sprintf("UPDATE %s SET is_verified = ? WHERE user_id = ?", tableName)
	r.logSQL("UpdateVerification", tableName, route, query, isVerified, user.ID)
	_, err = db.Exec(query, isVerified, user.ID)
	return err
}
