package service

import (
	"testing"
	"todolist-app/internal/domain"
)

func TestAuthService_Register(t *testing.T) {
	mockRepo := &mockUserRepo{}
	mockEmail := &mockEmailService{}
	svc := NewAuthService(mockRepo, mockEmail)

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		password := "password123"

		// Mock: User does not exist
		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return nil, nil
		}
		// Mock: Create success
		mockRepo.CreateFunc = func(user *domain.User) error {
			if user.Email != email {
				t.Errorf("expected email %s, got %s", email, user.Email)
			}
			if user.VerificationCode == "" {
				t.Error("expected verification code to be generated")
			}
			return nil
		}
		// Mock: Email success
		mockEmail.SendVerificationCodeFunc = func(to, code string) error {
			if to != email {
				t.Errorf("expected email to %s, got %s", email, to)
			}
			return nil
		}

		code, err := svc.Register(email, password)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if code == "" {
			t.Error("expected code to be returned")
		}
	})

	t.Run("UserAlreadyExists", func(t *testing.T) {
		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return &domain.User{Email: email}, nil
		}

		_, err := svc.Register("existing@example.com", "pass")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "user already exists" {
			t.Errorf("expected 'user already exists', got '%v'", err)
		}
	})
}

func TestAuthService_Verify(t *testing.T) {
	mockRepo := &mockUserRepo{}
	svc := NewAuthService(mockRepo, &mockEmailService{})

	t.Run("Success", func(t *testing.T) {
		email := "test@example.com"
		code := "1234"

		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return &domain.User{Email: email, VerificationCode: code}, nil
		}
		mockRepo.UpdateVerificationFunc = func(e string, isVerified bool) error {
			if e != email || !isVerified {
				t.Error("unexpected update params")
			}
			return nil
		}

		if err := svc.Verify(email, code); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("InvalidCode", func(t *testing.T) {
		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return &domain.User{Email: email, VerificationCode: "1234"}, nil
		}

		if err := svc.Verify("test@example.com", "9999"); err == nil {
			t.Error("expected error for invalid code")
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	mockRepo := &mockUserRepo{}
	svc := NewAuthService(mockRepo, &mockEmailService{})

	t.Run("Success", func(t *testing.T) {
		user := &domain.User{
			ID:           1,
			Email:        "test@example.com",
			PasswordHash: "secret",
			IsVerified:   true,
		}
		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return user, nil
		}

		token, u, err := svc.Login("test@example.com", "secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "1" { // In our mock implementation token is ID
			t.Errorf("expected token '1', got %s", token)
		}
		if u.ID != 1 {
			t.Error("expected user returned")
		}
	})

	t.Run("NotVerified", func(t *testing.T) {
		user := &domain.User{
			Email:        "test@example.com",
			PasswordHash: "secret",
			IsVerified:   false,
		}
		mockRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			return user, nil
		}

		_, _, err := svc.Login("test@example.com", "secret")
		if err == nil {
			t.Error("expected error for unverified user")
		}
	})
}
