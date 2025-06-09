package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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
	subscriberNames := make([]string, 0)
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

	// Start web server
	web.StartServer()

	// Update web interface with initial results
	if refuges != nil {
		web.UpdateState(refuges, time.Now())
	}

	// Set up ticker for regular checks
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Main loop
	for {
		select {
		case <-ticker.C:
			log.Printf("Checking availability...")
			refuges, err := parser.ParseRefugeAvailability(refugeURL, targetDate)
			if err != nil {
				log.Printf("Warning: Failed to check availability: %v", err)
				continue
			}

			// Update web interface with current time
			web.UpdateState(refuges, time.Now())

			// Check for new available dates
			var newDates []string
			for _, refuge := range refuges {
				for date, status := range refuge.Dates {
					if status != "Full" && !notifiedDates[date] {
						newDates = append(newDates, fmt.Sprintf("%s: %s places", date, status))
						notifiedDates[date] = true
					}
				}
			}

			// Send notification only if there are new dates
			if len(newDates) > 0 {
				notification := fmt.Sprintf("🎉 New availability found!\n%s", strings.Join(newDates, "\n"))
				if err := telegram.SendMessage(notification); err != nil {
					log.Printf("Warning: Failed to send notification: %v", err)
				}
			}

		case <-sigChan:
			log.Println("Received shutdown signal, stopping...")
			shutdownMsg := fmt.Sprintf("🛑 Monitoring stopped for %s", targetDate.Format("2006-01-02"))
			if err := telegram.SendMessage(shutdownMsg); err != nil {
				log.Printf("Warning: Failed to send shutdown message: %v", err)
			}
			return
		}
	}
}
