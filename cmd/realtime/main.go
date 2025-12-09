package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

const (
	defaultPort         = "8091"
	defaultMaxPerList   = 500
	defaultRedisAddr    = "localhost:6379"
	pingInterval        = 20 * time.Second
	pongWait            = 30 * time.Second
	writeWait           = 10 * time.Second
	redisSubscribeRetry = 3 * time.Second
)

type hub struct {
	mu           sync.RWMutex
	clients      map[int64]map[*client]struct{}
	subscribers  map[int64]context.CancelFunc
	maxPerList   int
	redis        *redis.Client
	upgrader     websocket.Upgrader
}

type client struct {
	h      *hub
	conn   *websocket.Conn
	listID int64
	userID int64
	send   chan []byte
}

func newHub(rdb *redis.Client, maxPerList int) *hub {
	return &hub{
		clients:     make(map[int64]map[*client]struct{}),
		subscribers: make(map[int64]context.CancelFunc),
		maxPerList:  maxPerList,
		redis:       rdb,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

func (h *hub) serveWS(w http.ResponseWriter, r *http.Request) {
	listIDStr := r.URL.Query().Get("list_id")
	userIDStr := r.URL.Query().Get("user_id")
	if listIDStr == "" || userIDStr == "" {
		http.Error(w, "list_id and user_id required", http.StatusBadRequest)
		return
	}
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid list_id", http.StatusBadRequest)
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	cl := &client{
		h:      h,
		conn:   conn,
		listID: listID,
		userID: userID,
		send:   make(chan []byte, 256),
	}

	if !h.addClient(cl) {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many editors"))
		conn.Close()
		return
	}

	go cl.writePump()
	go cl.readPump()
}

func (h *hub) addClient(c *client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	listClients := h.clients[c.listID]
	if listClients == nil {
		listClients = make(map[*client]struct{})
		h.clients[c.listID] = listClients
	}
	if len(listClients) >= h.maxPerList {
		return false
	}
	listClients[c] = struct{}{}

	if _, ok := h.subscribers[c.listID]; !ok {
		ctx, cancel := context.WithCancel(context.Background())
		h.subscribers[c.listID] = cancel
		go h.subscribe(ctx, c.listID)
	}
	return true
}

func (h *hub) removeClient(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if listClients, ok := h.clients[c.listID]; ok {
		delete(listClients, c)
		if len(listClients) == 0 {
			delete(h.clients, c.listID)
			if cancel, ok := h.subscribers[c.listID]; ok {
				cancel()
				delete(h.subscribers, c.listID)
			}
		}
	}
	c.conn.Close()
}

func (h *hub) broadcast(listID int64, msg []byte, exclude *client) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for cl := range h.clients[listID] {
		if cl == exclude {
			continue
		}
		select {
		case cl.send <- msg:
		default:
			// drop if backpressure
		}
	}
}

func (h *hub) publish(listID int64, payload []byte) {
	channel := redisChannel(listID)
	if err := h.redis.Publish(context.Background(), channel, payload).Err(); err != nil {
		log.Printf("⚠️ redis publish failed: list=%d err=%v", listID, err)
	}
}

func (h *hub) subscribe(ctx context.Context, listID int64) {
	channel := redisChannel(listID)
	for {
		pubsub := h.redis.Subscribe(ctx, channel)
		_, err := pubsub.Receive(ctx)
		if err != nil {
			log.Printf("⚠️ redis subscribe failed: list=%d err=%v", listID, err)
			select {
			case <-time.After(redisSubscribeRetry):
				continue
			case <-ctx.Done():
				return
			}
		}
		ch := pubsub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					pubsub.Close()
					goto retry
				}
				h.broadcast(listID, []byte(msg.Payload), nil)
			case <-ctx.Done():
				pubsub.Close()
				return
			}
		}
	retry:
		select {
		case <-time.After(redisSubscribeRetry):
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (c *client) readPump() {
	defer c.h.removeClient(c)
	c.conn.SetReadLimit(64 * 1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// publish to Redis for fanout across instances
		c.h.publish(c.listID, message)
		// broadcast locally (exclude sender)
		c.h.broadcast(c.listID, message, c)
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.h.removeClient(c)
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func redisChannel(listID int64) string {
	return "list:" + strconv.FormatInt(listID, 10)
}

func main() {
	port := os.Getenv("REALTIME_PORT")
	if port == "" {
		port = defaultPort
	}
	maxPerList := defaultMaxPerList
	if v := os.Getenv("REALTIME_MAX_PER_LIST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxPerList = n
		}
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = defaultRedisAddr
	}
	redisPass := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis connect failed: %v", err)
	}
	log.Printf("✅ realtime connected to redis %s", redisAddr)

	h := newHub(rdb, maxPerList)

	router := chi.NewRouter()
	router.Get("/ws", h.serveWS)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Printf("🚀 realtime server listening on :%s (max per list: %d)", port, maxPerList)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("⏳ shutting down realtime server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	_ = rdb.Close()
	log.Println("✅ realtime server stopped")
}

