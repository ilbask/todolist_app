package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"todolist-app/internal/infrastructure"
)

// CaptchaHandler handles CAPTCHA-related requests
type CaptchaHandler struct {
	captcha *infrastructure.CaptchaService
}

// NewCaptchaHandler creates a new CAPTCHA handler
func NewCaptchaHandler(captcha *infrastructure.CaptchaService) *CaptchaHandler {
	return &CaptchaHandler{captcha: captcha}
}

// Generate generates a new CAPTCHA
// GET /captcha/generate
func (h *CaptchaHandler) Generate(w http.ResponseWriter, r *http.Request) {
	captchaID := h.captcha.Generate()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"captcha_id": captchaID,
	})
}

// GetImage serves the CAPTCHA image
// GET /captcha/image/{captchaID}
func (h *CaptchaHandler) GetImage(w http.ResponseWriter, r *http.Request) {
	captchaID := chi.URLParam(r, "captchaID")
	if captchaID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "captcha_id is required"})
		return
	}

	if err := h.captcha.ServeImage(w, r, captchaID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate image"})
	}
}

// Verify verifies a CAPTCHA solution
// POST /captcha/verify
// Body: { "captcha_id": "...", "solution": "..." }
func (h *CaptchaHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CaptchaID string `json:"captcha_id"`
		Solution  string `json:"solution"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	isValid := h.captcha.Verify(req.CaptchaID, req.Solution)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": isValid,
	})
}


