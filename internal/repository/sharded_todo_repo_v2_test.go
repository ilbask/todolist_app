package repository

import (
	"errors"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestRecordIndexRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := &shardedTodoRepoV2{}

	// Expect DDL creation
	mock.ExpectExec(regexp.QuoteMeta(`
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
`)).WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect insert into retry table
	mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO user_list_index_retry (user_id, list_id, role, target_table, err_msg, retries)
VALUES (?, ?, ?, ?, ?, 0)
`)).
		WithArgs(int64(1), int64(2), "OWNER", "user_list_index_0000", "boom").
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := repo.recordIndexRetry(db, "user_list_index_0000", 1, 2, "OWNER", errors.New("boom")); err != nil {
		t.Fatalf("recordIndexRetry failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEnsureIndexRetryTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
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
`)).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureIndexRetryTable(db); err != nil {
		t.Fatalf("ensureIndexRetryTable failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

