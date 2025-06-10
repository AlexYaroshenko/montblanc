package web

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/AlexYaroshenko/montblanc/internal/parser"
)

var (
	state struct {
		Refuges   []parser.Refuge
		LastCheck time.Time
		mu        sync.RWMutex
	}
)

func StartServer() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Initialize LastCheck time
	state.mu.Lock()
	state.LastCheck = time.Now()
	state.mu.Unlock()

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server with timeouts
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🌐 Starting web server on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("❌ Server error: %v", err)
		}
	}()

	// Start keep-alive goroutine
	go keepAlive()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}
}

func UpdateState(refuges []parser.Refuge, lastCheck time.Time) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.Refuges = refuges
	if !lastCheck.IsZero() {
		state.LastCheck = lastCheck
		log.Printf("Updated web state - Last check: %v, Refuges: %d", state.LastCheck, len(state.Refuges))
	} else {
		log.Printf("Warning: Attempted to update web state with zero time")
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Refuge Availability</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .refuge {
            background: white;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .refuge h2 {
            margin: 0 0 15px 0;
            color: #333;
        }
        .dates {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
            gap: 10px;
        }
        .date {
            padding: 10px;
            border-radius: 4px;
            text-align: center;
            font-size: 14px;
        }
        .date.full {
            background: #f0f0f0;
            color: #999;
        }
        .date.available {
            background: #e6ffe6;
            color: #2e7d32;
            font-weight: bold;
        }
        .places {
            display: block;
            font-size: 12px;
            margin-top: 4px;
            color: #1b5e20;
        }
        .last-check {
            color: #666;
            font-size: 14px;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <h1>Refuge Availability</h1>
    <div class="container">
        {{range .Refuges}}
        <div class="refuge">
            <h2>{{.Name}}</h2>
            <div class="dates">
                {{range $date, $status := .Dates}}
                    {{if eq $status "Full"}}
                        <div class="date full">{{$date}}</div>
                    {{else}}
                        <div class="date available">
                            {{$date}}
                            <span class="places">{{$status}} places</span>
                        </div>
                    {{end}}
                {{end}}
            </div>
        </div>
        {{end}}
    </div>
    <div class="last-check">
        Last updated: {{.LastCheck.Format "2006-01-02 15:04:05"}}
    </div>
</body>
</html>`

	t, err := template.New("home").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "refuges": ` + strconv.Itoa(len(state.Refuges)) + `, "last_check": "` + state.LastCheck.Format(time.RFC3339) + `"}`))
}

// keepAlive periodically pings the health check endpoint to keep the instance alive
func keepAlive() {
	// Hardcoded base URL for RENDER deployment
	baseURL := "https://montblanc.onrender.com"
	log.Printf("🌐 Keep-alive using base URL: %s", baseURL)

	// Create ticker for periodic pings
	ticker := time.NewTicker(14 * time.Minute) // Ping every 14 minutes to stay within free tier limits
	defer ticker.Stop()

	for range ticker.C {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			log.Printf("❌ Keep-alive ping failed: %v", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			log.Printf("✅ Keep-alive ping successful")
		} else {
			log.Printf("❌ Keep-alive ping returned status: %d", resp.StatusCode)
		}
	}
}
