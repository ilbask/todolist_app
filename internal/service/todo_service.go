package service

import (
	"errors"
	"log"
	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure"
)

type todoService struct {
	repo     domain.TodoRepository
	userRepo domain.UserRepository
	kafka    *infrastructure.KafkaProducer
}

// NewTodoService wires the repositories and optional kafka producer into a todoService.
func NewTodoService(repo domain.TodoRepository, userRepo domain.UserRepository, kafka *infrastructure.KafkaProducer) domain.TodoService {
	return &todoService{repo: repo, userRepo: userRepo, kafka: kafka}
}

func (s *todoService) CreateList(userID int64, title string) (*domain.TodoList, error) {
	list := &domain.TodoList{
		OwnerID: userID,
		Title:   title,
	}
	if err := s.repo.CreateList(list); err != nil {
		return nil, err
	}
	return list, nil
}

func (s *todoService) GetLists(userID int64) ([]domain.TodoList, error) {
	return s.repo.GetListsByUserID(userID)
}

func (s *todoService) DeleteList(userID, listID int64) error {
	list, err := s.repo.GetListByID(listID)
	if err != nil || list == nil {
		return errors.New("list not found")
	}
	if list.OwnerID != userID {
		return errors.New("permission denied")
	}
	return s.repo.DeleteList(listID)
}

func (s *todoService) ShareList(ownerID, listID int64, targetEmail string, role domain.Role) error {
	// 1. Validate Owner
	list, err := s.repo.GetListByID(listID)
	if err != nil || list == nil {
		return errors.New("list not found")
	}
	if list.OwnerID != ownerID {
		return errors.New("permission denied")
	}

	// 2. Find Target User
	targetUser, err := s.userRepo.GetByEmail(targetEmail)
	if err != nil || targetUser == nil {
		return errors.New("target user not found")
	}

	// 3. Add Collaborator
	if err := s.repo.AddCollaborator(listID, targetUser.ID, role); err != nil {
		return err
	}

	// 4. Async Notification (Kafka)
	s.kafka.Publish("list.shared", []byte(targetEmail))

	return nil
}

func (s *todoService) AddItem(userID, listID int64, content string) (*domain.TodoItem, error) {
	// TODO: Check permissions (is owner or collaborator)

	item := &domain.TodoItem{
		ListID:   listID,
		Content:  content,
		Name:     content, // 默认使用content作为name（向后兼容）
		IsDone:   false,
		Status:   domain.StatusNotStarted,
		Priority: domain.PriorityMedium,
	}
	if err := s.repo.CreateItem(item); err != nil {
		log.Printf("❌ [TodoService] CreateItem failed list=%d err=%v", item.ListID, err)
		return nil, err
	}

	// Real-time Push (via Kafka/Redis to WS Hub)
	// Here we assume the WS hub subscribes to this topic
	s.kafka.Publish("item.created", []byte(content))

	return item, nil
}

// CreateItemExtended 创建扩展item
func (s *todoService) CreateItemExtended(userID, listID int64, item *domain.TodoItem) (*domain.TodoItem, error) {
	// TODO: Check permissions (is owner or collaborator)

	log.Printf("📝 [TodoService] CreateItemExtended user=%d list=%d item=%+v", userID, listID, item)
	item.ListID = listID
	if item.Status == "" {
		item.Status = domain.StatusNotStarted
	}
	if item.Priority == "" {
		item.Priority = domain.PriorityMedium
	}

	if err := s.repo.CreateItem(item); err != nil {
		return nil, err
	}

	// Real-time Push
	s.kafka.Publish("item.created", []byte(item.Name))

	return item, nil
}

func (s *todoService) GetItems(userID, listID int64) ([]domain.TodoItem, error) {
	// TODO: Check permissions
	return s.repo.GetItemsByListID(listID)
}

func (s *todoService) UpdateItem(userID, listID, itemID int64, isDone bool) (*domain.TodoItem, error) {
	// Simplified: assuming item exists and user has rights
	item := &domain.TodoItem{
		ID:     itemID,
		IsDone: isDone,
	}
	if err := s.repo.UpdateItemWithListID(listID, item); err != nil {
		return nil, err
	}
	return item, nil
}

// UpdateItemExtended 更新扩展item
func (s *todoService) UpdateItemExtended(userID, listID int64, item *domain.TodoItem) (*domain.TodoItem, error) {
	// TODO: Check permissions
	if err := s.repo.UpdateItemWithListID(listID, item); err != nil {
		return nil, err
	}

	// Real-time Push
	s.kafka.Publish("item.updated", []byte(item.Name))

	return item, nil
}

// GetItemsFiltered 获取带筛选和排序的items
func (s *todoService) GetItemsFiltered(userID, listID int64, filter *domain.ItemFilter, sort *domain.ItemSort) ([]domain.TodoItem, error) {
	// TODO: Check permissions
	return s.repo.GetItemsByListIDWithFilter(listID, filter, sort)
}

func (s *todoService) DeleteItem(userID, listID, itemID int64) error {
	// TODO: Check permissions
	return s.repo.DeleteItemWithListID(listID, itemID)
}
