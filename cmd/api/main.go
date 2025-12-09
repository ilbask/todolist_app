package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"todolist-app/internal/handler"
	"todolist-app/internal/infrastructure"
	"todolist-app/internal/infrastructure/sharding"
	"todolist-app/internal/repository"
	"todolist-app/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-sql-driver/mysql"
)

// Constants from init script
const (
	UserLogicalShards = 1024
	TodoLogicalShards = 4096
	// Index colocated with User

	UserPhysicalDBs = 16
	TodoPhysicalDBs = 64
)

func main() {
	// Setup Logging
	logFile, err := os.OpenFile("log/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

	// 1. Initialize Sharding Router V2
	router := sharding.NewRouterV2(UserLogicalShards, TodoLogicalShards)

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASS")
	// dbPass = "115119_hH"

	// Connect to User DB Clusters (Contains Users + UserIndex tables)
	for i := 0; i < UserPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_user_db_%d", i)
		dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true", dbUser, dbPass, dbName)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("Failed to open %s: %v", dbName, err)
		}
		if err := db.Ping(); err != nil {
			log.Printf("⚠️ Warning: %s unreachable: %v", dbName, err)
		} else {
			clusterID := fmt.Sprintf("todo_user_db_%d", i)
			router.RegisterCluster(clusterID, db, true, false)
		}
	}

	// Connect to Todo Data DB Clusters
	for i := 0; i < TodoPhysicalDBs; i++ {
		dbName := fmt.Sprintf("todo_data_db_%d", i)
		dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?parseTime=true", dbUser, dbPass, dbName)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("Failed to open %s: %v", dbName, err)
		}
		if err := db.Ping(); err != nil {
			log.Printf("⚠️ Warning: %s unreachable: %v", dbName, err)
		} else {
			clusterID := fmt.Sprintf("todo_data_db_%d", i)
			router.RegisterCluster(clusterID, db, false, true)
		}
	}

	log.Println("✅ Sharding Router V2 Initialized")

	// Infrastructure Services
	redis := infrastructure.NewRedisClient()
	defer redis.Close()

	kafka := infrastructure.NewKafkaProducer()
	defer kafka.Close()

	emailSvc := infrastructure.NewEmailServiceFromEnv()
	captchaSvc := infrastructure.NewCaptchaService()

	// 2. Repositories (V2 with Sharding Router)
	userRepo, err := repository.NewShardedUserRepoV2(router)
	if err != nil {
		log.Fatal(err)
	}
	todoRepo, err := repository.NewShardedTodoRepoV2(router)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Services (with Redis Caching)
	authSvc := service.NewAuthService(userRepo, emailSvc)
	baseTodoSvc := service.NewTodoService(todoRepo, userRepo, kafka)
	todoSvc := service.NewCachedTodoService(baseTodoSvc, redis) // Wrap with cache

	// 4. Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	todoHandler := handler.NewTodoHandler(todoSvc)
	captchaHandler := handler.NewCaptchaHandler(captchaSvc)
	mediaHandler := handler.NewMediaHandler(kafka)

	// 5. Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-User-ID"},
	}))

	// API Routes
	r.Route("/api", func(r chi.Router) {
		// Auth Routes
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/verify", authHandler.Verify)
		r.Post("/auth/login", authHandler.Login)

		// CAPTCHA Routes (Public)
		r.Get("/captcha/generate", captchaHandler.Generate)
		r.Get("/captcha/image/{captchaID}", captchaHandler.GetImage)
		r.Post("/captcha/verify", captchaHandler.Verify)

		// Protected Routes (Require Authentication)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					token := r.Header.Get("Authorization")
					if len(token) > 7 {
						r.Header.Set("X-User-ID", token[7:])
						next.ServeHTTP(w, r)
					} else {
						http.Error(w, "Unauthorized", 401)
					}
				})
			})

			// Todo Routes
			r.Get("/lists", todoHandler.GetLists)
			r.Post("/lists", todoHandler.CreateList)
			r.Delete("/lists/{id}", todoHandler.DeleteList)
			r.Post("/lists/{id}/share", todoHandler.ShareList)

			// Todo Items - Basic (Backward Compatibility)
			r.Get("/lists/{id}/items", todoHandler.GetItems)
			r.Post("/lists/{id}/items", todoHandler.AddItem)
			r.Put("/items/{id}", todoHandler.UpdateItem)
			r.Delete("/items/{id}", todoHandler.DeleteItem)

			// Todo Items - Extended (v2)
			r.Post("/lists/{id}/items/extended", todoHandler.CreateItemExtended)
			r.Put("/items/{id}/extended", todoHandler.UpdateItemExtended)
			r.Get("/lists/{id}/items/filtered", todoHandler.GetItemsFiltered)

			// Media Upload Route
			r.Post("/media/upload", mediaHandler.UploadMedia)
		})
	})

	// Web Demo
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "web"))
	r.Handle("/*", http.FileServer(filesDir))

	log.Println("🚀 TodoList App running on :8080")
	log.Println("✨ Features: Sharding, Redis Cache, CAPTCHA, Media Upload (Kafka)")
	http.ListenAndServe(":8080", r)
}
