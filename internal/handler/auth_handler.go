package handler

import (
	"encoding/json"
	"net/http"
	"todolist-app/internal/domain"
)

type AuthHandler struct {
	svc domain.AuthService
}

func NewAuthHandler(svc domain.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Helper to send JSON error
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid body", 400)
		return
	}

	if req.Email == "" || req.Password == "" {
		jsonError(w, "Email and Password are required", 400)
		return
	}

	code, err := h.svc.Register(req.Email, req.Password)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Verification code sent! (DEMO CODE: " + code + ")",
		"code":    code,
	})
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid body", 400)
		return
	}

	if err := h.svc.Verify(req.Email, req.Code); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Verified!"})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid body", 400)
		return
	}

	token, user, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		jsonError(w, err.Error(), 401)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": token,
		"user":  user,
	})
}
