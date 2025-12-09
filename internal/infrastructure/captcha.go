package infrastructure

import (
	"bytes"
	"net/http"
	"time"

	"github.com/dchest/captcha"
)

// CaptchaService provides CAPTCHA generation and verification
type CaptchaService struct {
	store captcha.Store
}

// NewCaptchaService creates a new CAPTCHA service
func NewCaptchaService() *CaptchaService {
	// Use in-memory store with 10-minute expiration
	store := captcha.NewMemoryStore(1000, 10*time.Minute)
	captcha.SetCustomStore(store)

	return &CaptchaService{
		store: store,
	}
}

// Generate creates a new CAPTCHA and returns its ID
func (s *CaptchaService) Generate() string {
	return captcha.New()
}

// Verify checks if the provided solution is correct
func (s *CaptchaService) Verify(captchaID, solution string) bool {
	return captcha.VerifyString(captchaID, solution)
}

// ServeImage serves the CAPTCHA image via HTTP
func (s *CaptchaService) ServeImage(w http.ResponseWriter, r *http.Request, captchaID string) error {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	// Use captcha's built-in WriteImage function
	var buf bytes.Buffer
	if err := captcha.WriteImage(&buf, captchaID, captcha.StdWidth, captcha.StdHeight); err != nil {
		return err
	}
	
	_, err := w.Write(buf.Bytes())
	return err
}


