package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/AlexYaroshenko/montblanc/internal/parser"
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

	// Set hardcoded target date to July 1st, 2025
	targetDate := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)

	// Track previously notified dates
	notifiedDates := make(map[string]bool)

	// Perform initial availability check
	log.Printf("Performing initial availability check...")
	refuges, err := parser.ParseRefugeAvailability(refugeURL, targetDate)
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
	startMsg := fmt.Sprintf("🚀 Monitoring started for %s\nCheck interval: %v", targetDate.Format("2006-01-02"), checkInterval)
	if err := telegram.SendMessage(startMsg); err != nil {
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
			refuges, err := parser.ParseRefugeAvailability(refugeURL, targetDate)
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
				if err := telegram.SendMessage(notification); err != nil {
					log.Printf("❌ Failed to send warning notification: %v", err)
				} else {
					log.Printf("✅ Warning notification sent successfully")
				}
				continue
			}

			// Sort by date
			sort.Slice(newAvailabilities, func(i, j int) bool {
				return newAvailabilities[i].date < newAvailabilities[j].date
			})

			// Group by refuge
			refugeGroups := make(map[string][]string)
			for _, avail := range newAvailabilities {
				refugeGroups[avail.refuge] = append(refugeGroups[avail.refuge],
					fmt.Sprintf("%s: %s places", avail.date, avail.status))
			}

			// Format notification
			if len(newAvailabilities) > 0 {
				var notification strings.Builder
				notification.WriteString("🎉 New availability found!\n\n")

				for _, refuge := range []string{"Tête Rousse", "du Goûter"} {
					if dates, exists := refugeGroups[refuge]; exists {
						notification.WriteString(fmt.Sprintf("🏔️ %s:\n", refuge))
						for _, date := range dates {
							notification.WriteString(fmt.Sprintf("  • %s\n", date))
						}
						notification.WriteString("\n")
					}
				}

				if err := telegram.SendMessage(notification.String()); err != nil {
					log.Printf("❌ Failed to send notification: %v", err)
				} else {
					log.Printf("✅ Notification sent successfully")
				}
			} else {
				log.Printf("ℹ️ No new availability found at %v", time.Now().Format("2006-01-02 15:04:05"))
			}
			log.Printf("✅ Check completed at %v", time.Now().Format("2006-01-02 15:04:05"))

		case <-sigChan:
			log.Println("🛑 Received shutdown signal, stopping...")
			shutdownMsg := fmt.Sprintf("🛑 Monitoring stopped for %s", targetDate.Format("2006-01-02"))
			if err := telegram.SendMessage(shutdownMsg); err != nil {
				log.Printf("❌ Failed to send shutdown message: %v", err)
			}
			return
		}
	}
}
