package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"todolist-app/internal/infrastructure"
)

// MediaHandler handles media upload requests
type MediaHandler struct {
	kafka     *infrastructure.KafkaProducer
	uploadDir string
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(kafka *infrastructure.KafkaProducer) *MediaHandler {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	// Create upload directory if it doesn't exist
	os.MkdirAll(uploadDir, 0755)

	return &MediaHandler{
		kafka:     kafka,
		uploadDir: uploadDir,
	}
}

// UploadMedia handles media file uploads
// POST /media/upload
// Form data: user_id, list_id, item_id, media_type, file
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "File too large"})
		return
	}

	// Parse form fields
	userID, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	listID, _ := strconv.ParseInt(r.FormValue("list_id"), 10, 64)
	itemID, _ := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	mediaType := r.FormValue("media_type") // "image" or "video"

	if userID == 0 || listID == 0 || itemID == 0 || mediaType == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required fields"})
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Generate unique filename
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d_%d_%d_%s", userID, listID, timestamp, header.Filename)
	filePath := filepath.Join(h.uploadDir, fileName)

	// Save file temporarily
	dst, err := os.Create(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to save file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write file"})
		return
	}

	// Send event to Kafka for async S3 upload
	s3Key := fmt.Sprintf("media/%d/%d/%s", userID, listID, fileName)
	event := infrastructure.MediaUploadEvent{
		UserID:     userID,
		ListID:     listID,
		ItemID:     itemID,
		MediaType:  mediaType,
		FileName:   fileName,
		FilePath:   filePath,
		S3Bucket:   os.Getenv("S3_BUCKET"),
		S3Key:      s3Key,
		UploadedAt: time.Now(),
	}

	if err := h.kafka.SendMediaUploadEvent(r.Context(), event); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to queue upload"})
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "File uploaded successfully and queued for S3 upload",
		"s3_key":  s3Key,
	})
}


