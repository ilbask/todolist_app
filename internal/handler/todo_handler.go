package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
	"todolist-app/internal/domain"

	"github.com/go-chi/chi/v5"
)

type TodoHandler struct {
	svc domain.TodoService
}

func NewTodoHandler(svc domain.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

func (h *TodoHandler) GetLists(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	lists, err := h.svc.GetLists(userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(lists)
}

func (h *TodoHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	var req struct {
		Title string `json:"title"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	list, err := h.svc.CreateList(userID, req.Title)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(list)
}

func (h *TodoHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := h.svc.DeleteList(userID, listID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TodoHandler) ShareList(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.svc.ShareList(userID, listID, req.Email, domain.Role(req.Role)); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TodoHandler) GetItems(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	items, err := h.svc.GetItems(userID, listID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(items)
}

func (h *TodoHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	var req struct {
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	item, err := h.svc.AddItem(userID, listID, req.Content)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(item)
}

func (h *TodoHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	itemID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	// We need ListID for sharding routing.
	// In REST, best practice is /lists/{listID}/items/{itemID}
	// But our route is /items/{id}.
	// We need to fetch ListID from somewhere or change API.
	// Hack: Client passes list_id in body, or we change route.
	// Let's assume we change route later, but for now we extract from body if possible or error.
	// Wait, the UI code sends it? No.
	// Best fix: Require list_id in JSON body for update.
	
	var req struct {
		ListID int64 `json:"list_id"` // Added requirement
		IsDone bool  `json:"is_done"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.ListID == 0 {
		http.Error(w, "list_id required for sharding", 400)
		return
	}

	item, err := h.svc.UpdateItem(userID, req.ListID, itemID, req.IsDone)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(item)
}

func (h *TodoHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	itemID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	// Issue: DELETE typically no body.
	// We should use Query Param ?list_id=...
	listIDStr := r.URL.Query().Get("list_id")
	listID, _ := strconv.ParseInt(listIDStr, 10, 64)

	if listID == 0 {
		http.Error(w, "list_id query param required for sharding", 400)
		return
	}

	if err := h.svc.DeleteItem(userID, listID, itemID); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// CreateItemExtended 创建扩展Item（支持所有新字段）
func (h *TodoHandler) CreateItemExtended(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	var item domain.TodoItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body", 400)
		return
	}

	createdItem, err := h.svc.CreateItemExtended(userID, listID, &item)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdItem)
}

// UpdateItemExtended 更新扩展Item（支持所有新字段）
func (h *TodoHandler) UpdateItemExtended(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	itemID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	var req struct {
		ListID      int64   `json:"list_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Status      string  `json:"status"`
		Priority    string  `json:"priority"`
		DueDate     *string `json:"due_date"`
		Tags        string  `json:"tags"`
		IsDone      bool    `json:"is_done"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", 400)
		return
	}

	if req.ListID == 0 {
		http.Error(w, "list_id required for sharding", 400)
		return
	}

	item := &domain.TodoItem{
		ID:          itemID,
		Name:        req.Name,
		Description: req.Description,
		Status:      domain.ItemStatus(req.Status),
		Priority:    domain.Priority(req.Priority),
		DueDate:     parseDueDate(req.DueDate),
		Tags:        req.Tags,
		IsDone:      req.IsDone,
	}

	updatedItem, err := h.svc.UpdateItemExtended(userID, req.ListID, item)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedItem)
}

// GetItemsFiltered 获取带筛选和排序的Items
func (h *TodoHandler) GetItemsFiltered(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	listID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	
	// 解析筛选参数
	filter := &domain.ItemFilter{}
	if status := r.URL.Query().Get("status"); status != "" {
		s := domain.ItemStatus(status)
		filter.Status = &s
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		p := domain.Priority(priority)
		filter.Priority = &p
	}
	if dueBefore := r.URL.Query().Get("due_before"); dueBefore != "" {
		filter.DueBefore = parseDueDate(&dueBefore)
	}
	if dueAfter := r.URL.Query().Get("due_after"); dueAfter != "" {
		filter.DueAfter = parseDueDate(&dueAfter)
	}
	if tags := r.URL.Query()["tags"]; len(tags) > 0 {
		filter.Tags = tags
	}
	
	// 解析排序参数
	sort := &domain.ItemSort{}
	if sortField := r.URL.Query().Get("sort"); sortField != "" {
		sort.Field = sortField
		if r.URL.Query().Get("order") == "desc" {
			sort.Desc = true
		}
	}
	
	items, err := h.svc.GetItemsFiltered(userID, listID, filter, sort)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// parseDueDate 辅助函数，将字符串解析为time.Time指针
func parseDueDate(dateStr *string) *time.Time {
	if dateStr == nil || *dateStr == "" {
		return nil
	}
	
	// 尝试多种日期格式
	formats := []string{
		"2006-01-02T15:04:05Z07:00", // RFC3339
		"2006-01-02 15:04:05",        // MySQL datetime
		"2006-01-02",                 // Date only
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, *dateStr); err == nil {
			return &t
		}
	}
	
	return nil
}
