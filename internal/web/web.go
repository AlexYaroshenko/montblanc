package web

import (
	"html/template"
	"log"
	"net/http"
	"sync"
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

	port := "8080"
	log.Printf("Starting web server on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}

func UpdateState(refuges []parser.Refuge, lastCheck time.Time) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.Refuges = refuges
	state.LastCheck = lastCheck
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
	w.Write([]byte(`{"status": "ok", "refuges": ` + string(len(state.Refuges)) + `, "last_check": "` + state.LastCheck.Format(time.RFC3339) + `"}`))
}
