package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/AlexYaroshenko/montblanc/internal/parser"
	"github.com/AlexYaroshenko/montblanc/internal/store"
	"github.com/AlexYaroshenko/montblanc/internal/telegram"
	"github.com/AlexYaroshenko/montblanc/internal/web"
	"github.com/joho/godotenv"
)

const (
	refugeURL     = "https://montblanc.ffcam.fr/GB_reservation-tout-public.html"
	checkInterval = 1 * time.Minute
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Require Google Analytics measurement ID
	if os.Getenv("GA_MEASUREMENT_ID") == "" {
		log.Fatal("GA_MEASUREMENT_ID is not set")
	}

	// Rolling window: from today to two months ahead (fetch month views)
	now := time.Now().UTC()
	// Normalize to first day of current month
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	// Build list of 3 month anchors: current, +1, +2
	monthAnchors := []time.Time{
		monthStart,
		monthStart.AddDate(0, 1, 0),
		monthStart.AddDate(0, 2, 0),
	}

	// Open store: require Postgres
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	st, err := store.OpenPostgres(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to open postgres: %v", err)
	}
	defer st.Close()

	// Track previously notified dates
	notifiedDates := make(map[string]bool)

	// Perform initial availability check
	log.Printf("Performing initial availability check for 3-month window starting %s...", monthStart.Format("2006-01-02"))
	refuges, err := fetchRefugesWindow(refugeURL, monthAnchors)
	if err != nil {
		log.Printf("Warning: Initial availability check failed: %v", err)
	}

	// Get subscriber names
	var subscriberNames []string
	if chatIDs := os.Getenv("TELEGRAM_CHAT_IDS"); chatIDs != "" {
		for _, chatID := range telegram.ParseChatIDs(chatIDs) {
			if name, err := telegram.GetUserInfo(chatID); err == nil {
				subscriberNames = append(subscriberNames, name)
			} else {
				subscriberNames = append(subscriberNames, chatID)
			}
		}
	}

	// Send start message
	windowEnd := monthStart.AddDate(0, 3, -1)
	startMsg := fmt.Sprintf("🚀 Monitoring started for window %s – %s\nCheck interval: %v", monthStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"), checkInterval)
	if err := sendToSubscribersOrEnv(st, startMsg); err != nil {
		log.Printf("Warning: Failed to send start message: %v", err)
	}

	// Start web server in a goroutine
	go func() {
		log.Printf("🌐 Starting web server...")
		web.StartServer()
	}()

	// Update web interface with initial results
	if refuges != nil {
		web.UpdateState(refuges, time.Now())
	}

	// Set up ticker for regular checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Printf("⏰ Starting main loop with check interval: %v", checkInterval)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main loop
	for {
		log.Printf("⏳ Waiting for next tick...")
		select {
		case <-ticker.C:
			log.Printf("🔔 Ticker triggered at %v - Starting availability check...", time.Now().Format("2006-01-02 15:04:05"))
			// refresh month anchors on each tick to keep rolling window
			now = time.Now().UTC()
			monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			monthAnchors = []time.Time{monthStart, monthStart.AddDate(0, 1, 0), monthStart.AddDate(0, 2, 0)}

			refuges, err := fetchRefugesWindow(refugeURL, monthAnchors)
			if err != nil {
				log.Printf("❌ Failed to check availability: %v", err)
				continue
			}

			// Update web interface with current time
			web.UpdateState(refuges, time.Now())
			log.Printf("✅ Web interface updated at %v", time.Now().Format("2006-01-02 15:04:05"))

			// Check for new available dates
			type availability struct {
				refuge string
				date   string
				status string
			}
			var newAvailabilities []availability

			// Check if we got any dates at all
			totalDates := 0
			for _, refuge := range refuges {
				totalDates += len(refuge.Dates)
				for date, status := range refuge.Dates {
					if status != "Full" && !notifiedDates[date] {
						newAvailabilities = append(newAvailabilities, availability{
							refuge: refuge.Name,
							date:   date,
							status: status,
						})
						notifiedDates[date] = true
					}
				}
			}

			// Notify if no dates were parsed
			if totalDates == 0 {
				notification := "⚠️ Warning: No dates were parsed from the response. This might indicate an issue with the website or session."
				if err := sendToSubscribersOrEnv(st, notification); err != nil {
					log.Printf("❌ Failed to send warning notification: %v", err)
				} else {
					log.Printf("✅ Warning notification sent successfully")
				}
				continue
			}

			// Per-subscriber filtered notifications based on saved queries
			if len(newAvailabilities) > 0 {
				subs, err := st.ListSubscribers()
				if err != nil {
					log.Printf("❌ Failed to list subscribers: %v", err)
				} else {
					for _, sub := range subs {
						qs, err := st.ListQueriesByChat(sub.ChatID)
						if err != nil {
							log.Printf("❌ Failed to list queries for %s: %v", sub.ChatID, err)
							continue
						}
						if len(qs) == 0 {
							continue
						}

						// Build matches for this subscriber
						type line struct {
							refuge string
							date   string
							status string
						}
						var lines []line
						for _, avail := range newAvailabilities {
							for _, q := range qs {
								if queryMatches(avail.refuge, avail.date, q) {
									lines = append(lines, line{refuge: avail.refuge, date: avail.date, status: avail.status})
									// mark date as notified globally to avoid repeats
									notifiedDates[avail.date] = true
									break
								}
							}
						}
						if len(lines) == 0 {
							continue
						}

						// Sort by date and group by refuge
						sort.Slice(lines, func(i, j int) bool { return lines[i].date < lines[j].date })
						groups := map[string][]string{}
						for _, l := range lines {
							groups[l.refuge] = append(groups[l.refuge], fmt.Sprintf("%s: %s places", l.date, l.status))
						}

						var b strings.Builder
						b.WriteString("🎉 New availability found for your subscription!\n\n")
						for _, r := range []string{"Tête Rousse", "du Goûter"} {
							if dates, ok := groups[r]; ok {
								b.WriteString(fmt.Sprintf("🏔️ %s:\n", r))
								for _, d := range dates {
									b.WriteString("  • " + d + "\n")
								}
								b.WriteString("\n")
							}
						}
						// any other refuges
						for r, dates := range groups {
							if r == "Tête Rousse" || r == "du Goûter" {
								continue
							}
							b.WriteString(fmt.Sprintf("🏔️ %s:\n", r))
							for _, d := range dates {
								b.WriteString("  • " + d + "\n")
							}
							b.WriteString("\n")
						}
						_ = telegram.SendMessageTo(sub.ChatID, b.String())
					}
				}
			} else {
				log.Printf("ℹ️ No new availability found at %v", time.Now().Format("2006-01-02 15:04:05"))
			}
			log.Printf("✅ Check completed at %v", time.Now().Format("2006-01-02 15:04:05"))

		case <-sigChan:
			log.Println("🛑 Received shutdown signal, stopping...")
			shutdownMsg := "🛑 Monitoring stopped"
			if err := sendToSubscribersOrEnv(st, shutdownMsg); err != nil {
				log.Printf("❌ Failed to send shutdown message: %v", err)
			}
			return
		}
	}
}

// fetchRefugesWindow fetches availability for multiple month anchors and merges the results
func fetchRefugesWindow(refugeURL string, monthAnchors []time.Time) ([]parser.Refuge, error) {
	merged := make(map[string]parser.Refuge)
	for _, anchor := range monthAnchors {
		res, err := parser.ParseRefugeAvailability(refugeURL, anchor)
		if err != nil {
			return nil, err
		}
		for _, rf := range res {
			if existing, ok := merged[rf.Name]; ok {
				// merge dates
				for d, s := range rf.Dates {
					existing.Dates[d] = s
				}
				merged[rf.Name] = existing
			} else {
				// copy to avoid aliasing
				copyDates := make(map[string]string, len(rf.Dates))
				for d, s := range rf.Dates {
					copyDates[d] = s
				}
				merged[rf.Name] = parser.Refuge{Name: rf.Name, Dates: copyDates}
			}
		}
	}
	// flatten
	out := make([]parser.Refuge, 0, len(merged))
	for _, rf := range merged {
		out = append(out, rf)
	}
	return out, nil
}

// sendToSubscribersOrEnv sends to DB/bolt subscribers if available; otherwise falls back to TELEGRAM_CHAT_IDS
func sendToSubscribersOrEnv(st store.Store, msg string) error {
	if st != nil {
		subs, err := st.ListSubscribers()
		if err == nil && len(subs) > 0 {
			for _, s := range subs {
				_ = telegram.SendMessageTo(s.ChatID, msg)
			}
			return nil
		}
	}
	return telegram.SendMessage(msg)
}

// queryMatches checks if an availability line matches a saved query
func queryMatches(refuge string, date string, q store.Query) bool {
	if q.Refuge != "*" && q.Refuge != refuge {
		return false
	}
	// date window
	if q.DateFrom == "" && q.DateTo == "" {
		return true
	}
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	if q.DateFrom != "" {
		if from, err := time.Parse("2006-01-02", q.DateFrom); err == nil {
			if d.Before(from) {
				return false
			}
		}
	}
	if q.DateTo != "" {
		if to, err := time.Parse("2006-01-02", q.DateTo); err == nil {
			if d.After(to) {
				return false
			}
		}
	}
	return true
}
