package service

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure"
)

type authService struct {
	repo  domain.UserRepository
	email infrastructure.EmailService
}

func NewAuthService(repo domain.UserRepository, email infrastructure.EmailService) domain.AuthService {
	return &authService{
		repo:  repo,
		email: email,
	}
}

func (s *authService) Register(email, password string) (string, error) {
	log.Printf("📝 [Register] Attempting registration for Email: %s, Password: %s", email, password)

	// 1. Check if user exists
	existing, _ := s.repo.GetByEmail(email)
	if existing != nil {
		return "", errors.New("user already exists")
	}

	// 2. Generate Code
	code := fmt.Sprintf("%04d", rand.Intn(10000))

	// 3. Hash Password
	// hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// if err != nil {
	// 	return "", err
	// }

	// 4. Create User
	user := &domain.User{
		Email:            email,
		PasswordHash:     password, // Plain text storage
		VerificationCode: code,
		IsVerified:       false,
	}

	if err := s.repo.Create(user); err != nil {
		return "", err
	}

	// 5. Send Email
	s.email.SendVerificationCode(email, code)

	return code, nil
}

func (s *authService) Verify(email, code string) error {
	user, err := s.repo.GetByEmail(email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}
	if user.VerificationCode != code {
		return errors.New("invalid code")
	}
	return s.repo.UpdateVerification(email, true)
}

func (s *authService) Login(email, password string) (string, *domain.User, error) {
	user, err := s.repo.GetByEmail(email)
	if err != nil || user == nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Compare Hash
	// if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
	// 	return "", nil, errors.New("invalid credentials")
	// }
	if user.PasswordHash != password {
		return "", nil, errors.New("invalid credentials")
	}

	if !user.IsVerified {
		return "", nil, errors.New("account not verified")
	}

	// In real app, generate JWT
	token := fmt.Sprintf("%d", user.ID)
	return token, user, nil
}
