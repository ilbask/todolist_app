package service

import (
	"testing"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure"
)

func TestTodoService_CreateList(t *testing.T) {
	mockRepo := &mockTodoRepo{}
	mockUserRepo := &mockUserRepo{}
	mockKafka := infrastructure.NewKafkaProducer([]string{"mock"})
	svc := NewTodoService(mockRepo, mockUserRepo, mockKafka)

	t.Run("Success", func(t *testing.T) {
		mockRepo.CreateListFunc = func(list *domain.TodoList) error {
			list.ID = 100
			return nil
		}

		list, err := svc.CreateList(1, "My List")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if list.ID != 100 {
			t.Errorf("expected list ID 100, got %d", list.ID)
		}
		if list.Title != "My List" {
			t.Errorf("expected title 'My List', got %s", list.Title)
		}
	})
}

func TestTodoService_ShareList(t *testing.T) {
	mockRepo := &mockTodoRepo{}
	mockUserRepo := &mockUserRepo{}
	mockKafka := infrastructure.NewKafkaProducer([]string{"mock"})
	svc := NewTodoService(mockRepo, mockUserRepo, mockKafka)

	t.Run("Success", func(t *testing.T) {
		ownerID := int64(1)
		listID := int64(10)
		targetEmail := "friend@example.com"
		targetID := int64(2)

		// Mock: Get List
		mockRepo.GetListByIDFunc = func(id int64) (*domain.TodoList, error) {
			if id == listID {
				return &domain.TodoList{ID: listID, OwnerID: ownerID}, nil
			}
			return nil, nil
		}
		// Mock: Find Target User
		mockUserRepo.GetByEmailFunc = func(email string) (*domain.User, error) {
			if email == targetEmail {
				return &domain.User{ID: targetID, Email: email}, nil
			}
			return nil, nil
		}
		// Mock: Add Collaborator
		mockRepo.AddCollaboratorFunc = func(lid, uid int64, role domain.Role) error {
			if lid != listID || uid != targetID {
				t.Errorf("unexpected params: %d, %d", lid, uid)
			}
			return nil
		}

		err := svc.ShareList(ownerID, listID, targetEmail, domain.RoleEditor)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		ownerID := int64(1)
		otherUserID := int64(99)
		listID := int64(10)

		mockRepo.GetListByIDFunc = func(id int64) (*domain.TodoList, error) {
			return &domain.TodoList{ID: listID, OwnerID: ownerID}, nil
		}

		// Try to share as non-owner
		err := svc.ShareList(otherUserID, listID, "friend@example.com", domain.RoleEditor)
		if err == nil {
			t.Error("expected permission denied error")
		}
		if err.Error() != "permission denied" {
			t.Errorf("expected 'permission denied', got '%v'", err)
		}
	})
}

func TestTodoService_AddItem(t *testing.T) {
	mockRepo := &mockTodoRepo{}
	mockUserRepo := &mockUserRepo{}
	mockKafka := infrastructure.NewKafkaProducer([]string{"mock"})
	svc := NewTodoService(mockRepo, mockUserRepo, mockKafka)

	t.Run("Success", func(t *testing.T) {
		mockRepo.CreateItemFunc = func(item *domain.TodoItem) error {
			item.ID = 50
			return nil
		}

		item, err := svc.AddItem(1, 10, "Buy Milk")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if item.ID != 50 {
			t.Errorf("expected item ID 50, got %d", item.ID)
		}
	})
}



