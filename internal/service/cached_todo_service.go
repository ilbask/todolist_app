package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"todolist-app/internal/domain"
	"todolist-app/internal/infrastructure"
)

// CachedTodoService wraps TodoService with Redis caching (Read-Aside pattern)
type CachedTodoService struct {
	base  domain.TodoService
	redis *infrastructure.RedisClient
	ttl   time.Duration
	ctx   context.Context
}

// NewCachedTodoService creates a cached todo service
func NewCachedTodoService(base domain.TodoService, redis *infrastructure.RedisClient) domain.TodoService {
	return &CachedTodoService{
		base:  base,
		redis: redis,
		ttl:   5 * time.Minute, // Default TTL for cached data
		ctx:   context.Background(),
	}
}

// Cache key helpers
func itemsKey(listID int64) string {
	return fmt.Sprintf("items:%d", listID)
}

func userListsKey(userID int64) string {
	return fmt.Sprintf("user_lists:%d", userID)
}

// CreateList creates a list and invalidates user's list cache
func (s *CachedTodoService) CreateList(userID int64, title string) (*domain.TodoList, error) {
	list, err := s.base.CreateList(userID, title)
	if err != nil {
		return nil, err
	}

	// Invalidate user's list cache
	if s.redis.IsAvailable() {
		if err := s.redis.Del(s.ctx, userListsKey(userID)); err != nil {
			log.Printf("⚠️ Failed to invalidate cache for user %d: %v", userID, err)
		}
	}

	return list, nil
}

// GetLists retrieves all lists for a user (with cache)
func (s *CachedTodoService) GetLists(userID int64) ([]domain.TodoList, error) {
	cacheKey := userListsKey(userID)

	// Try cache first
	if s.redis.IsAvailable() {
		cached, err := s.redis.Get(s.ctx, cacheKey)
		if err == nil {
			var lists []domain.TodoList
			if json.Unmarshal([]byte(cached), &lists) == nil {
				log.Printf("🎯 Cache HIT: %s", cacheKey)
				return lists, nil
			}
		}
		log.Printf("📭 Cache MISS: %s", cacheKey)
	}

	// Fallback to database
	lists, err := s.base.GetLists(userID)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if s.redis.IsAvailable() {
		if data, err := json.Marshal(lists); err == nil {
			s.redis.Set(s.ctx, cacheKey, data, s.ttl)
		}
	}

	return lists, nil
}

// DeleteList deletes a list and invalidates cache
func (s *CachedTodoService) DeleteList(userID, listID int64) error {
	if err := s.base.DeleteList(userID, listID); err != nil {
		return err
	}

	// Invalidate cache
	if s.redis.IsAvailable() {
		s.redis.Del(s.ctx, itemsKey(listID), userListsKey(userID))
	}

	return nil
}

// ShareList shares a list (no caching for now, just pass-through)
func (s *CachedTodoService) ShareList(ownerID, listID int64, targetEmail string, role domain.Role) error {
	if err := s.base.ShareList(ownerID, listID, targetEmail, role); err != nil {
		return err
	}

	// Note: We don't know the sharedUserID here, so we can't invalidate their cache
	// This could be improved by returning the shared user ID from ShareList

	return nil
}

// AddItem adds an item and invalidates cache
func (s *CachedTodoService) AddItem(userID, listID int64, content string) (*domain.TodoItem, error) {
	item, err := s.base.AddItem(userID, listID, content)
	if err != nil {
		return nil, err
	}

	// Invalidate items cache
	if s.redis.IsAvailable() {
		s.redis.Del(s.ctx, itemsKey(listID))
	}

	return item, nil
}

// GetItems retrieves items for a list (with cache)
func (s *CachedTodoService) GetItems(userID, listID int64) ([]domain.TodoItem, error) {
	cacheKey := itemsKey(listID)

	// Try cache first
	if s.redis.IsAvailable() {
		cached, err := s.redis.Get(s.ctx, cacheKey)
		if err == nil {
			var items []domain.TodoItem
			if json.Unmarshal([]byte(cached), &items) == nil {
				log.Printf("🎯 Cache HIT: %s", cacheKey)
				return items, nil
			}
		}
		log.Printf("📭 Cache MISS: %s", cacheKey)
	}

	// Fallback to database
	items, err := s.base.GetItems(userID, listID)
	if err != nil {
		return nil, err
	}

	// Populate cache
	if s.redis.IsAvailable() {
		if data, err := json.Marshal(items); err == nil {
			s.redis.Set(s.ctx, cacheKey, data, s.ttl)
		}
	}

	return items, nil
}

// UpdateItem updates an item and invalidates cache
func (s *CachedTodoService) UpdateItem(userID, listID, itemID int64, isDone bool) (*domain.TodoItem, error) {
	item, err := s.base.UpdateItem(userID, listID, itemID, isDone)
	if err != nil {
		return nil, err
	}

	// Invalidate items cache
	if s.redis.IsAvailable() {
		s.redis.Del(s.ctx, itemsKey(listID))
	}

	return item, nil
}

// DeleteItem deletes an item and invalidates cache
func (s *CachedTodoService) DeleteItem(userID, listID, itemID int64) error {
	if err := s.base.DeleteItem(userID, listID, itemID); err != nil {
		return err
	}

	// Invalidate items cache
	if s.redis.IsAvailable() {
		s.redis.Del(s.ctx, itemsKey(listID))
	}

	return nil
}


