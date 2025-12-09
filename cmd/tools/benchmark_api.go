package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	apiURL      = flag.String("url", "http://localhost:8080", "API base URL")
	duration    = flag.Int("duration", 60, "Test duration in seconds")
	concurrency = flag.Int("concurrency", 100, "Number of concurrent requests")
	testType    = flag.String("test", "all", "Test type: login, register, query, create, update, delete, share, all")
)

type Stats struct {
	Total    int64
	Success  int64
	Failure  int64
	TotalMs  int64
}

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags)
	log.Printf("🚀 Starting API Benchmark")
	log.Printf("🎯 Target: %s", *apiURL)
	log.Printf("⏱️  Duration: %d seconds, Concurrency: %d", *duration, *concurrency)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var stats Stats

	switch *testType {
	case "register":
		benchmarkRegister(client, &stats)
	case "login":
		benchmarkLogin(client, &stats)
	case "query":
		benchmarkQuery(client, &stats)
	case "create":
		benchmarkCreate(client, &stats)
	case "update":
		benchmarkUpdate(client, &stats)
	case "delete":
		benchmarkDelete(client, &stats)
	case "share":
		benchmarkShare(client, &stats)
	case "all":
		benchmarkAll(client, &stats)
	default:
		log.Fatal("❌ Unknown test type:", *testType)
	}

	printStats(&stats)
}

func benchmarkAll(client *http.Client, stats *Stats) {
	log.Println("🔄 Running all benchmarks...")
	benchmarkRegister(client, stats)
	benchmarkLogin(client, stats)
	benchmarkQuery(client, stats)
	benchmarkCreate(client, stats)
	benchmarkUpdate(client, stats)
	benchmarkShare(client, stats)
}

func benchmarkRegister(client *http.Client, stats *Stats) {
	log.Println("📝 Benchmarking Register API...")
	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		body := map[string]string{
			"email":    fmt.Sprintf("bench_%d_%d@test.com", time.Now().Unix(), i),
			"password": "testpass123",
		}
		data, _ := json.Marshal(body)
		return http.NewRequest("POST", *apiURL+"/auth/register", bytes.NewBuffer(data))
	})
}

func benchmarkLogin(client *http.Client, stats *Stats) {
	log.Println("🔑 Benchmarking Login API...")
	// Pre-register a user
	email := fmt.Sprintf("bench_login_%d@test.com", time.Now().Unix())
	registerUser(client, email, "testpass123")

	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		body := map[string]string{
			"email":    email,
			"password": "testpass123",
		}
		data, _ := json.Marshal(body)
		return http.NewRequest("POST", *apiURL+"/auth/login", bytes.NewBuffer(data))
	})
}

func benchmarkQuery(client *http.Client, stats *Stats) {
	log.Println("🔍 Benchmarking Query API...")
	token := setupTestUser(client)

	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		req, err := http.NewRequest("GET", *apiURL+"/todo/lists", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return req, nil
	})
}

func benchmarkCreate(client *http.Client, stats *Stats) {
	log.Println("➕ Benchmarking Create API...")
	token := setupTestUser(client)

	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		body := map[string]string{
			"title": fmt.Sprintf("Benchmark List %d", i),
		}
		data, _ := json.Marshal(body)
		req, err := http.NewRequest("POST", *apiURL+"/todo/lists", bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return req, nil
	})
}

func benchmarkUpdate(client *http.Client, stats *Stats) {
	log.Println("✏️ Benchmarking Update API...")
	token, listID := setupTestUserWithList(client)

	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		body := map[string]string{
			"title": fmt.Sprintf("Updated List %d", i),
		}
		data, _ := json.Marshal(body)
		req, err := http.NewRequest("PUT", fmt.Sprintf("%s/todo/lists/%d", *apiURL, listID), bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return req, nil
	})
}

func benchmarkShare(client *http.Client, stats *Stats) {
	log.Println("🤝 Benchmarking Share API...")
	token, listID := setupTestUserWithList(client)

	runBenchmark(client, stats, func(i int) (*http.Request, error) {
		body := map[string]interface{}{
			"shared_user_id": 999999 + i, // Dummy user IDs
			"role":           "editor",
		}
		data, _ := json.Marshal(body)
		req, err := http.NewRequest("POST", fmt.Sprintf("%s/todo/lists/%d/share", *apiURL, listID), bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return req, nil
	})
}

func runBenchmark(client *http.Client, stats *Stats, reqFactory func(int) (*http.Request, error)) {
	var wg sync.WaitGroup
	stopChan := make(chan struct{})

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			counter := 0
			for {
				select {
				case <-stopChan:
					return
				default:
					start := time.Now()
					req, err := reqFactory(counter)
					if err != nil {
						atomic.AddInt64(&stats.Failure, 1)
						continue
					}

					req.Header.Set("Content-Type", "application/json")
					resp, err := client.Do(req)

					elapsed := time.Since(start).Milliseconds()
					atomic.AddInt64(&stats.TotalMs, elapsed)
					atomic.AddInt64(&stats.Total, 1)

					if err != nil || resp.StatusCode >= 400 {
						atomic.AddInt64(&stats.Failure, 1)
					} else {
						atomic.AddInt64(&stats.Success, 1)
					}

					if resp != nil {
						resp.Body.Close()
					}

					counter++
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(time.Duration(*duration) * time.Second)
	close(stopChan)
	wg.Wait()
}

func setupTestUser(client *http.Client) string {
	email := fmt.Sprintf("bench_user_%d@test.com", time.Now().Unix())
	password := "testpass123"

	// Register
	registerUser(client, email, password)

	// Login
	body := map[string]string{"email": email, "password": password}
	data, _ := json.Marshal(body)
	resp, _ := client.Post(*apiURL+"/auth/login", "application/json", bytes.NewBuffer(data))
	if resp == nil {
		return ""
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result["token"].(string)
}

func setupTestUserWithList(client *http.Client) (string, int64) {
	token := setupTestUser(client)

	// Create a list
	body := map[string]string{"title": "Benchmark List"}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", *apiURL+"/todo/lists", bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := client.Do(req)
	if resp == nil {
		return token, 0
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	listID := int64(result["list_id"].(float64))

	return token, listID
}

func registerUser(client *http.Client, email, password string) {
	body := map[string]string{"email": email, "password": password}
	data, _ := json.Marshal(body)
	client.Post(*apiURL+"/auth/register", "application/json", bytes.NewBuffer(data))
}

func printStats(stats *Stats) {
	if stats.Total == 0 {
		log.Println("❌ No requests completed")
		return
	}

	avgLatency := float64(stats.TotalMs) / float64(stats.Total)
	qps := float64(stats.Total) / float64(*duration)
	successRate := float64(stats.Success) / float64(stats.Total) * 100

	log.Println("\n📊 Benchmark Results:")
	log.Printf("   Total Requests: %d", stats.Total)
	log.Printf("   Success: %d (%.2f%%)", stats.Success, successRate)
	log.Printf("   Failure: %d", stats.Failure)
	log.Printf("   Avg Latency: %.2f ms", avgLatency)
	log.Printf("   QPS: %.2f", qps)
}


