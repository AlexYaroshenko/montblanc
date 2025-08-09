package web

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/AlexYaroshenko/montblanc/internal/i18n"
	"github.com/AlexYaroshenko/montblanc/internal/store"
	"github.com/AlexYaroshenko/montblanc/internal/telegram"
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
    http.HandleFunc("/telegram/webhook", handleTelegramWebhook)
    http.HandleFunc("/subscribe", handleSubscribe)
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
    lang := i18n.DetectLang(r)
    // Copy state under lock into a lightweight view model (no mutex)
    state.mu.RLock()
    view := struct {
        Refuges   []parser.Refuge
        LastCheck time.Time
    }{
        Refuges:   state.Refuges,
        LastCheck: state.LastCheck,
    }
    state.mu.RUnlock()

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{T "title"}}</title>
    <style>
        :root { --bg: #0b1021; --text: #111; --muted: #666; --brand: #0f62fe; --card: #fff; --ok: #2e7d32; --full: #999; }
        * { box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; margin: 0; background: #f7f8fb; color: var(--text); }
        a { color: var(--brand); text-decoration: none; }
        .container { max-width: 1200px; margin: 0 auto; padding: 24px; }
        .nav { display: flex; justify-content: space-between; align-items: center; padding: 12px 24px; }
        .lang { font-size: 14px; color: var(--muted); }
        .lang a { margin-left: 8px; }
        .hero { position: relative; padding: 80px 24px; background: linear-gradient(180deg, rgba(6,16,36,.85), rgba(6,16,36,.85)), url('https://images.unsplash.com/photo-1520697222863-c66ef0b8d1b3?q=80&w=1600&auto=format&fit=crop'); background-size: cover; background-position: center; color: white; text-align: center; }
        .hero h1 { margin: 0 0 12px 0; font-size: 36px; letter-spacing: .2px; }
        .hero p { margin: 0 auto 20px; max-width: 760px; color: #dfe7ff; }
        .cta { display: inline-flex; gap: 12px; }
        .btn { display: inline-block; padding: 12px 18px; border-radius: 8px; font-weight: 600; }
        .btn.primary { background: var(--brand); color: white; }
        .btn.secondary { background: rgba(255,255,255,.1); color: white; border: 1px solid rgba(255,255,255,.25); }
        .section { padding: 40px 24px; }
        .section h2 { margin: 0 0 16px 0; font-size: 24px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; }
        .card { background: var(--card); border-radius: 12px; padding: 16px; box-shadow: 0 2px 8px rgba(0,0,0,.06); }
        .muted { color: var(--muted); }
        .refuge { background: white; border-radius: 8px; padding: 12px; }
        .dates { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 8px; }
        .date { padding: 10px; border-radius: 6px; text-align: center; font-size: 14px; background: #f3f5f8; }
        .date.full { color: var(--full); }
        .date.available { background: #e6ffe6; color: var(--ok); font-weight: 700; }
        .places { display: block; font-size: 12px; margin-top: 4px; color: #1b5e20; }
        .last-check { color: var(--muted); font-size: 13px; margin-top: 8px; }
    </style>
</head>
<body>
    <div class="nav">
      <div class="brand">Mont Blanc Alerts</div>
      <div class="lang">Lang:
        <a href="?lang=en">EN</a>
        <a href="?lang=de">DE</a>
        <a href="?lang=fr">FR</a>
        <a href="?lang=es">ES</a>
        <a href="?lang=it">IT</a>
      </div>
    </div>

    <section class="hero">
      <div class="container">
        <h1>{{T "hero_title"}}</h1>
        <p>{{T "hero_subtitle"}}</p>
        <div class="cta">
          <a class="btn primary" href="#demo">{{T "cta_check"}}</a>
          <a class="btn secondary" href="#subscribe">{{T "cta_subscribe"}}</a>
        </div>
      </div>
    </section>

    <section class="section">
      <div class="container">
        <h2>{{T "how_it_works_title"}}</h2>
        <div class="grid">
          <div class="card">🔁 {{T "step1"}}</div>
          <div class="card">📅 {{T "step2"}}</div>
          <div class="card">📲 {{T "step3"}}</div>
        </div>
      </div>
    </section>

    <section id="demo" class="section">
      <div class="container">
        <h2>{{T "demo_title"}}</h2>
        <div class="grid">
          {{range .Refuges}}
          <div class="card">
            <h3>{{.Name}}</h3>
            <div class="dates">
              {{range $date, $status := .Dates}}
                {{if eq $status "Full"}}
                  <div class="date full">{{$date}} · {{T "full"}}</div>
                {{else}}
                  <div class="date available">{{$date}} <span class="places">{{$status}} {{T "places"}}</span></div>
                {{end}}
              {{end}}
            </div>
          </div>
          {{end}}
        </div>
        <div class="last-check">{{T "last_updated"}}: {{.LastCheck.Format "2006-01-02 15:04:05"}}</div>
      </div>
    </section>

    <section class="section">
      <div class="container">
        <h2>{{T "refuges_title"}}</h2>
        <div class="grid">
          <div class="card">🏔️ Refuge du Goûter 🇫🇷</div>
          <div class="card">🏔️ Tête Rousse 🇫🇷</div>
          <div class="card">🏔️ Refuge des Cosmiques 🇫🇷</div>
          <div class="card">🏔️ Rifugio Torino 🇮🇹</div>
        </div>
      </div>
    </section>

    <section id="subscribe" class="section">
      <div class="container">
        <h2>{{T "cta_subscribe"}}</h2>
        <div class="grid">
          <div class="card">
            <form method="post" action="/subscribe">
              <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;">
                <div>
                  <label class="muted">{{T "chat_id"}}</label>
                  <input name="chat_id" required placeholder="123456789" style="width:100%;padding:10px;border-radius:8px;border:1px solid #e2e8f0;" />
                </div>
                <div>
                  <label class="muted">{{T "language"}}</label>
                  <select name="language" style="width:100%;padding:10px;border-radius:8px;border:1px solid #e2e8f0;">
                    <option value="en">EN</option>
                    <option value="de">DE</option>
                    <option value="fr">FR</option>
                    <option value="es">ES</option>
                    <option value="it">IT</option>
                  </select>
                </div>
                <div>
                  <label class="muted">{{T "refuge"}}</label>
                  <select name="refuge" style="width:100%;padding:10px;border-radius:8px;border:1px solid #e2e8f0;">
                    <option value="*">Any</option>
                    <option value="Tête Rousse">Tête Rousse</option>
                    <option value="du Goûter">Refuge du Goûter</option>
                  </select>
                </div>
                <div>
                  <label class="muted">{{T "date_from"}}</label>
                  <input type="date" name="date_from" style="width:100%;padding:10px;border-radius:8px;border:1px solid #e2e8f0;" />
                </div>
                <div>
                  <label class="muted">{{T "date_to"}}</label>
                  <input type="date" name="date_to" style="width:100%;padding:10px;border-radius:8px;border:1px solid #e2e8f0;" />
                </div>
              </div>
              <div style="margin-top:12px">
                <button class="btn primary" type="submit">{{T "submit"}}</button>
              </div>
            </form>
            <p class="muted" style="margin-top:8px">Or start a chat with the Telegram bot and send /start.</p>
          </div>
        </div>
      </div>
    </section>
</body>
</html>`

    t, err := template.New("home").Funcs(template.FuncMap{
        "T": func(key string) string { return i18n.T(lang, key) },
    }).Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    if err := t.Execute(w, view); err != nil {
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

// Telegram webhook: save chat and simple /start
func handleTelegramWebhook(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }
    var upd telegram.Update
    if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if upd.Message == nil || upd.Message.Chat == nil { w.WriteHeader(http.StatusOK); return }
    chatID := fmt.Sprintf("%d", upd.Message.Chat.ID)

    // persist subscriber using Bolt as a local store
    // On Render we'll swap to Postgres impl, interface stays the same
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" { log.Printf("store open error: DATABASE_URL is empty"); w.WriteHeader(http.StatusOK); return }
    ps, err := store.OpenPostgres(context.Background(), dbURL)
    if err != nil { log.Printf("store open error: %v", err); w.WriteHeader(http.StatusOK); return }
    defer ps.Close()

    lang := "en"
    if upd.Message.From != nil && upd.Message.From.LanguageCode != "" { lang = upd.Message.From.LanguageCode }
    _ = ps.UpsertSubscriber(store.Subscriber{ChatID: chatID, Language: lang})

    // greet
    _ = telegram.SendMessageTo(chatID, "✅ Subscribed. We'll notify you about new dates. Send /stop to unsubscribe.")
    w.WriteHeader(http.StatusOK)
}

// handleSubscribe saves subscriber and a single query
func handleSubscribe(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Redirect(w, r, "/#subscribe", http.StatusSeeOther)
        return
    }
    if err := r.ParseForm(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    chatID := r.FormValue("chat_id")
    language := r.FormValue("language")
    refuge := r.FormValue("refuge")
    dateFrom := r.FormValue("date_from")
    dateTo := r.FormValue("date_to")

    if chatID == "" { http.Error(w, "chat_id required", http.StatusBadRequest); return }

    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" { http.Error(w, "DATABASE_URL is empty", http.StatusInternalServerError); return }
    ps, err := store.OpenPostgres(context.Background(), dbURL)
    if err != nil { http.Error(w, "store open error", http.StatusInternalServerError); return }
    defer ps.Close()

    if err := ps.UpsertSubscriber(store.Subscriber{ChatID: chatID, Language: language, IsActive: true}); err != nil {
        http.Error(w, "save subscriber error", http.StatusInternalServerError)
        return
    }
    _, _ = ps.AddQuery(store.Query{ChatID: chatID, Refuge: refuge, DateFrom: dateFrom, DateTo: dateTo})

    // optional: confirm in Telegram
    _ = telegram.SendMessageTo(chatID, "✅ Subscription saved. We'll notify you when matching dates appear.")
    http.Redirect(w, r, "/#subscribe", http.StatusSeeOther)
}
