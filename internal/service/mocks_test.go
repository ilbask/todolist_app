package service

import (
	"todolist-app/internal/domain"
)

// --- Mock User Repository ---
type mockUserRepo struct {
	CreateFunc             func(user *domain.User) error
	GetByEmailFunc         func(email string) (*domain.User, error)
	GetByIDFunc            func(id int64) (*domain.User, error)
	UpdateVerificationFunc func(email string, isVerified bool) error
}

func (m *mockUserRepo) Create(user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(user)
	}
	return nil
}

func (m *mockUserRepo) GetByEmail(email string) (*domain.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(email)
	}
	return nil, nil
}

func (m *mockUserRepo) GetByID(id int64) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateVerification(email string, isVerified bool) error {
	if m.UpdateVerificationFunc != nil {
		return m.UpdateVerificationFunc(email, isVerified)
	}
	return nil
}

// --- Mock Email Service ---
type mockEmailService struct {
	SendVerificationCodeFunc func(to, code string) error
}

func (m *mockEmailService) SendVerificationCode(to, code string) error {
	if m.SendVerificationCodeFunc != nil {
		return m.SendVerificationCodeFunc(to, code)
	}
	return nil
}

// --- Mock Todo Repository ---
type mockTodoRepo struct {
	CreateListFunc       func(list *domain.TodoList) error
	GetListsByUserIDFunc func(userID int64) ([]domain.TodoList, error)
	GetListByIDFunc      func(listID int64) (*domain.TodoList, error)
	DeleteListFunc       func(listID int64) error
	AddCollaboratorFunc  func(listID, userID int64, role domain.Role) error
	CreateItemFunc       func(item *domain.TodoItem) error
	GetItemsByListIDFunc func(listID int64) ([]domain.TodoItem, error)
	UpdateItemFunc       func(item *domain.TodoItem) error
	DeleteItemFunc       func(itemID int64) error
}

func (m *mockTodoRepo) CreateList(list *domain.TodoList) error {
	if m.CreateListFunc != nil {
		return m.CreateListFunc(list)
	}
	return nil
}

func (m *mockTodoRepo) GetListsByUserID(userID int64) ([]domain.TodoList, error) {
	if m.GetListsByUserIDFunc != nil {
		return m.GetListsByUserIDFunc(userID)
	}
	return nil, nil
}

func (m *mockTodoRepo) GetListByID(listID int64) (*domain.TodoList, error) {
	if m.GetListByIDFunc != nil {
		return m.GetListByIDFunc(listID)
	}
	return nil, nil
}

func (m *mockTodoRepo) DeleteList(listID int64) error {
	if m.DeleteListFunc != nil {
		return m.DeleteListFunc(listID)
	}
	return nil
}

func (m *mockTodoRepo) AddCollaborator(listID, userID int64, role domain.Role) error {
	if m.AddCollaboratorFunc != nil {
		return m.AddCollaboratorFunc(listID, userID, role)
	}
	return nil
}

func (m *mockTodoRepo) CreateItem(item *domain.TodoItem) error {
	if m.CreateItemFunc != nil {
		return m.CreateItemFunc(item)
	}
	return nil
}

func (m *mockTodoRepo) GetItemsByListID(listID int64) ([]domain.TodoItem, error) {
	if m.GetItemsByListIDFunc != nil {
		return m.GetItemsByListIDFunc(listID)
	}
	return nil, nil
}

func (m *mockTodoRepo) UpdateItem(item *domain.TodoItem) error {
	if m.UpdateItemFunc != nil {
		return m.UpdateItemFunc(item)
	}
	return nil
}

func (m *mockTodoRepo) DeleteItem(itemID int64) error {
	if m.DeleteItemFunc != nil {
		return m.DeleteItemFunc(itemID)
	}
	return nil
}


