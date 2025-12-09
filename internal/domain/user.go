package domain

import "time"

// User represents a user in the system
type User struct {
	ID               int64     `json:"id" db:"user_id"`
	Email            string    `json:"email" db:"email"`
	PasswordHash     string    `json:"-" db:"password_hash"`
	VerificationCode string    `json:"-" db:"verification_code"`
	IsVerified       bool      `json:"is_verified" db:"is_verified"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// UserRepository defines the interface for user data persistence
type UserRepository interface {
	Create(user *User) error
	GetByEmail(email string) (*User, error)
	GetByID(id int64) (*User, error)
	UpdateVerification(email string, isVerified bool) error
}

// AuthService defines the business logic for authentication
type AuthService interface {
	Register(email, password string) (string, error) // Returns verification code
	Verify(email, code string) error
	Login(email, password string) (string, *User, error) // Returns token, user, error
}

