# Hx-H.A.W.K.S
## High Accuracy Web Keywords Scanner

> **Ultra-fast, concurrent, customizable vulnerability scanner written in Go**  
> _“Scan. Detect. Dominate.”_

---

## 📌 Overview

**Hx-H.A.W.K.S** is a high-performance CLI and API-based tool built for security researchers, red teamers, and bug bounty hunters. It scans thousands of URLs, searches for specified keywords in the HTTP responses, and detects possible vulnerability footprints. All with color-coded terminal output, powerful concurrency options, and customizable formats.

---

## ⚙️ Features

- ✅ **Keyword-based response scanning** (e.g., `admin`, `password`, `flag{`, etc.)
- ⚡ **Super-fast concurrency** with goroutines (unlimited URLs)
- 🎯 **Multiple output formats**: plain, JSON, full reports
- 🌈 **Color-coded output** for quick terminal scanning:
  - 🟩 Green: Safe URLs  
  - 🟥 Red: Vulnerable URLs  
  - ⚪ White: Safe responses  
  - 🔵 Blue: Vulnerable responses  
  - 💗 Pink: Matched keywords
- 🧠 Smart filters, retries, timeouts, custom headers
- 🌐 **Built-in API server** (SSE + RESTful) for real-time results
- 🛠️ Ready for integration into future tools like **Fruttry**, **Hx-Bunny**, or custom dashboards

---

## 📦 Installation

## Go-Lang Installer (Self-Install)

```bash
go install github.com/hxbunny/hx-hawks@latest
```

### Manual

```bash
git clone https://github.com/nxneeraj/hx-hawks.git
cd hx-hawks
go build -o hx-hawks main.go
```

Now you're ready to fly 🦅

---

## 🧪 CLI Usage

```bash
./hx-hawks -f targets.txt -o vulnerable.txt --ck "admin,password,login"
```

### 🔧 Key Flags

| Flag                | Description |
|---------------------|-------------|
| `-f <file>`         | Input file of URLs (one per line) |
| `--ck "<k1>,<k2>"`  | Comma-separated keywords |
| `-o <file>`         | Plain text output (vulnerable URLs only) |
| `-o-json <file>`    | Save vulnerable data as JSON |
| `-o-response <file>`| Save response with each vulnerable URL |
| `-o-all <file>`     | Save all data (safe + vulnerable) |
| `-o-all-json <file>`| JSON output with metadata, IP, status |
| `--threads <num>`   | Goroutines to use (default 10) |
| `--timeout <s>`     | Timeout per URL (default 5s) |
| `--delay <ms>`      | Delay between requests |
| `--api`             | Enable API server mode |
| `--port <num>`      | Set custom API port (default 8080) |
| `--verbose`         | Print all scanning details |

---

## 📤 Output Formats

#### 📝 -o (Plain Vulnerable URLs)

```text
https://target.com/login
https://admin.site.com
```

#### 🧾 -o-json (Matched Results)

```json
[
  {
    "url": "https://target.com/login",
    "matched_keywords": ["login", "admin"],
    "response": "<html>Welcome admin</html>"
  }
]
```

#### 📊 -o-all-json (Full Metadata)

```json
{
  "url": "https://target.com/login",
  "status_code": 200,
  "ip": "93.184.216.34",
  "matched_keywords": ["admin"],
  "response": "<html>Admin panel</html>",
  "is_vulnerable": true,
  "timestamp": "2025-05-02T14:33:22Z"
}
```

---

## 🌐 API Mode

Start server:

```bash
./hx-hawks --api -f targets.txt --ck "password,login" --port 7171
```

### 📡 API Endpoints

| Endpoint                  | Method | Description |
|---------------------------|--------|-------------|
| `/scan/start`             | POST   | Start new scan (JSON payload) |
| `/scan/status/{jobID}`    | GET    | Get scan progress |
| `/scan/result/{jobID}`    | GET    | Get full results |
| `/scan/stream/{jobID}`    | GET    | Real-time events via SSE |

---

## 🚀 Example Use Cases

```bash
# Basic keyword scan
hx-hawks -f urls.txt --ck "admin,password"

# Save matched responses
hx-hawks -f urls.txt -o-response match.txt --ck "error,flag{"

# API mode on port 9000
hx-hawks --api -f urls.txt --ck "sql,injection" --port 9000
```

---

## 🗂️ Project Structure

```Project Structure
hx-hawks/
├── main.go                 # Entry point, CLI flag parsing, mode switching (CLI/API)
├── go.mod                  # Go module definition
├── go.sum                  # Go module checksums
│
├── pkg/                    # Internal packages
│   ├── config/             # Configuration handling
│   │   └── config.go
│   ├── scanner/            # Core scanning logic
│   │   └── scanner.go
│   │   └── worker.go       # Individual worker logic
│   ├── httpclient/         # Customized HTTP client
│   │   └── client.go
│   ├── output/             # Output formatting (terminal & file)
│   │   └── terminal.go
│   │   └── file.go
│   │   └── colors.go       # Color definitions
│   ├── types/              # Shared data structures
│   │   └── types.go
│   ├── utils/              # Utility functions (e.g., file reading)
│   │   └── utils.go
│   └── api/                # API server logic (if --api is enabled)
│       ├── server.go       # API server setup and routing
│       ├── handlers.go     # HTTP request handlers
│       └── manager.go      # Scan job management
│
├── examples/               # Example usage files
│   └── targets.txt
│
└── README.md               # Project documentation

```

---

## 🧠 Future Enhancements

- [x] Smart scan mode (auto keyword discovery)
- [x] Web GUI (real-time dashboard)
- [x] Cookie + auth support
- [x] Proxy and Tor integration
- [x] Headless browser / JS execution
- [x] Plugin support (LFI, SSRF, XSS modules)

---

## 🙌 Contribution Guide

PRs, feedback, and stars 🌟 are always welcome!

1. Fork & clone  
2. Create a feature branch  
3. Commit & push  
4. Open a pull request  
5. Discuss with Bunny 😏

---

## 📜 License

MIT License – Use it, fork it, break it, fix it 🔓  
See [LICENSE](./LICENSE)

---

Made with 💖 by Bunny & Hx-Crew 🐰🦅  
“Sniff the weak, strike with precision.”

------

> Build faster. Test smarter. Hack ethically.  
> With 💥 from Team HyperGod-X 👾
<p align="center"><strong> Keep Moving Forward </strong></p>


