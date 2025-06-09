package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	envBotToken = "MONTBLANC_BOT_TOKEN"
	envChatIDs  = "MONTBLANC_CHAT_IDS"
)

type Refuge struct {
	Name  string
	ID    string
	Dates map[string]string // date -> availability info
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func sendTelegramMessage(botToken string, chatIDs []string, message string) error {
	for _, chatID := range chatIDs {
		msg := TelegramMessage{
			ChatID:    chatID,
			Text:      message,
			ParseMode: "HTML",
		}

		jsonData, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("error marshaling message: %v", err)
		}

		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("error sending message to chat %s: %v", chatID, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("telegram API error for chat %s: %s", chatID, string(body))
		}
	}
	return nil
}

func checkRefuge(client *http.Client, refuge Refuge, date string, pax string) error {
	formData := url.Values{}
	formData.Set("action", "availability")
	formData.Set("parent_url", "https://montblanc.ffcam.fr/GB_reservation-tout-public.html")
	formData.Set("widgetHostCss", "https://centrale.ffcam.fr/css/widget-resa/ffcam/widget-resa.css")
	formData.Set("apporigin", "MONTBLANC")
	formData.Set("structures", "")
	formData.Set("faqurl", "")
	formData.Set("faqtitle", "")
	formData.Set("mode", "FORM_PREBOOK")
	formData.Set("structure", refuge.ID)
	formData.Set("productCategory", "nomatter")
	formData.Set("pax", pax)
	formData.Set("date", date)

	req, err := http.NewRequest("POST", "https://centrale.ffcam.fr/index.php?_lang=GB&", strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("accept-language", "en-US,en;q=0.9,uk;q=0.8,ru;q=0.7")
	req.Header.Set("cache-control", "max-age=0")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("origin", "https://centrale.ffcam.fr")
	req.Header.Set("referer", "https://centrale.ffcam.fr/index.php?_lang=GB&")
	req.Header.Set("sec-ch-ua", `"Chromium";v="136", "Google Chrome";v="136", "Not.A/Brand";v="99"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", "macOS")
	req.Header.Set("sec-fetch-dest", "iframe")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("sec-fetch-user", "?1")
	req.Header.Set("upgrade-insecure-requests", "1")
	req.Header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")

	// Add cookies
	cookies := []http.Cookie{
		{Name: "_ga", Value: "GA1.1.1973794123.1748930074"},
		{Name: "PHPSESSID", Value: "f0aa1c1498ee78b14bfa46a3313e124c"},
		{Name: "_ga_ZZX4DJ3EHR", Value: "GS2.1.s1749462950$o4$g1$t1749462950$j60$l0$h0"},
	}
	for _, cookie := range cookies {
		req.AddCookie(&cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}

	// Check for session expiration
	bodyStr := string(body)
	if strings.Contains(bodyStr, "session expired") || strings.Contains(bodyStr, "Votre session a expiré") {
		return fmt.Errorf("session expired - please update the PHPSESSID cookie")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return fmt.Errorf("error parsing HTML: %v", err)
	}

	// Initialize dates map if not exists
	if refuge.Dates == nil {
		refuge.Dates = make(map[string]string)
	}

	// Find all available dates
	doc.Find(".day.dispo").Each(func(i int, s *goquery.Selection) {
		date := s.Find("a").AttrOr("data-date", "")
		places := s.Find(".place").Text()
		if date != "" {
			refuge.Dates[date] = fmt.Sprintf("📅 %s places available", places)
		}
	})

	// Find all full dates
	doc.Find(".day.complet").Each(func(i int, s *goquery.Selection) {
		date := s.Find("a").AttrOr("data-date", "")
		if date != "" {
			refuge.Dates[date] = "❌ Full"
		}
	})

	return nil
}

func formatAvailabilityMessage(refuges []Refuge, month time.Time) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("<b>Refuge Availability for %s:</b>\n", month.Format("January 2006")))
	msg.WriteString("===============================\n\n")

	// Get all unique dates
	allDates := make(map[string]bool)
	for _, refuge := range refuges {
		for date := range refuge.Dates {
			allDates[date] = true
		}
	}

	// Format results by date
	for date := range allDates {
		msg.WriteString(fmt.Sprintf("📅 <b>%s</b>:\n", date))
		for _, refuge := range refuges {
			if status, exists := refuge.Dates[date]; exists {
				msg.WriteString(fmt.Sprintf("  %s: %s\n", refuge.Name, status))
			} else {
				msg.WriteString(fmt.Sprintf("  %s: Not available\n", refuge.Name))
			}
		}
		msg.WriteString("\n")
	}

	return msg.String()
}

func formatConsoleOutput(refuges []Refuge, month time.Time) string {
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("\nRefuge Availability for %s:\n", month.Format("January 2006")))
	msg.WriteString("===============================\n\n")

	// Get all unique dates
	allDates := make(map[string]bool)
	for _, refuge := range refuges {
		for date := range refuge.Dates {
			allDates[date] = true
		}
	}

	// Format results by date
	for date := range allDates {
		msg.WriteString(fmt.Sprintf("📅 %s:\n", date))
		for _, refuge := range refuges {
			if status, exists := refuge.Dates[date]; exists {
				msg.WriteString(fmt.Sprintf("  %s: %s\n", refuge.Name, status))
			} else {
				msg.WriteString(fmt.Sprintf("  %s: Not available\n", refuge.Name))
			}
		}
		msg.WriteString("\n")
	}

	return msg.String()
}

func main() {
	date := flag.String("date", "", "Booking date in YYYY-MM-DD format (will check the entire month)")
	pax := flag.String("pax", "1", "Number of people")
	botToken := flag.String("bot-token", os.Getenv(envBotToken), "Telegram bot token (can be set via MONTBLANC_BOT_TOKEN env var)")
	chatIDs := flag.String("chat-ids", os.Getenv(envChatIDs), "Comma-separated list of Telegram chat IDs (can be set via MONTBLANC_CHAT_IDS env var)")
	frequency := flag.Int("frequency", 1, "Check frequency in minutes (default: 1)")
	flag.Parse()

	if *date == "" {
		fmt.Println("Error: date is required")
		fmt.Println("Usage: check-booking -date YYYY-MM-DD [-pax NUMBER] [-chat-ids ID1,ID2,...] [-frequency MINUTES]")
		fmt.Println("\nEnvironment variables:")
		fmt.Printf("  %s: Telegram bot token\n", envBotToken)
		fmt.Printf("  %s: Comma-separated list of Telegram chat IDs\n", envChatIDs)
		os.Exit(1)
	}

	if *botToken == "" {
		fmt.Printf("Error: Telegram bot token is required. Set it via %s environment variable or -bot-token flag\n", envBotToken)
		os.Exit(1)
	}

	if *chatIDs == "" {
		fmt.Printf("Error: At least one Telegram chat ID is required. Set it via %s environment variable or -chat-ids flag\n", envChatIDs)
		os.Exit(1)
	}

	if *frequency < 1 {
		fmt.Println("Error: frequency must be at least 1 minute")
		os.Exit(1)
	}

	// Split chat IDs
	chatIDList := strings.Split(*chatIDs, ",")
	for i, id := range chatIDList {
		chatIDList[i] = strings.TrimSpace(id)
	}

	// Parse the input date to get the first day of the month
	inputDate, err := time.Parse("2006-01-02", *date)
	if err != nil {
		fmt.Printf("Error parsing date: %v\n", err)
		os.Exit(1)
	}

	// Get the first day of the month
	firstDayOfMonth := time.Date(inputDate.Year(), inputDate.Month(), 1, 0, 0, 0, 0, inputDate.Location())
	monthStr := firstDayOfMonth.Format("2006-01-02")

	refuges := []Refuge{
		{Name: "Tête Rousse", ID: "BK_STRUCTURE:29"},
		{Name: "du Goûter", ID: "BK_STRUCTURE:30"},
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Store previous results for comparison
	previousResults := make(map[string]map[string]string)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Send startup notification
	startupMsg := fmt.Sprintf("🚀 Starting monitoring for %s\nChecking refuges: %s\nCheck frequency: every %d minute(s)",
		firstDayOfMonth.Format("January 2006"),
		strings.Join([]string{refuges[0].Name, refuges[1].Name}, ", "),
		*frequency)
	if err := sendTelegramMessage(*botToken, chatIDList, startupMsg); err != nil {
		fmt.Printf("Error sending startup notification: %v\n", err)
	}

	fmt.Printf("Starting continuous monitoring for %s...\n", firstDayOfMonth.Format("January 2006"))
	fmt.Printf("Check frequency: every %d minute(s)\n", *frequency)
	fmt.Println("Press Ctrl+C to stop")

	// Create a channel to handle program termination
	done := make(chan bool)

	// Start monitoring in a goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				// Check availability for each refuge
				for i := range refuges {
					if err := checkRefuge(client, refuges[i], monthStr, *pax); err != nil {
						if strings.Contains(err.Error(), "session expired") {
							errorMsg := fmt.Sprintf("⚠️ Session expired!\nPlease update the PHPSESSID cookie in the code.")
							if err := sendTelegramMessage(*botToken, chatIDList, errorMsg); err != nil {
								fmt.Printf("Error sending error notification: %v\n", err)
							}
							fmt.Println("\n⚠️  Session expired!")
							fmt.Println("Please update the PHPSESSID cookie in the code with a new value from your browser.")
							fmt.Println("You can get it by:")
							fmt.Println("1. Opening https://centrale.ffcam.fr/ in your browser")
							fmt.Println("2. Opening Developer Tools (F12)")
							fmt.Println("3. Going to Application/Storage tab")
							fmt.Println("4. Looking for PHPSESSID cookie")
							done <- true
							return
						}
						errorMsg := fmt.Sprintf("⚠️ Error checking %s: %v", refuges[i].Name, err)
						if err := sendTelegramMessage(*botToken, chatIDList, errorMsg); err != nil {
							fmt.Printf("Error sending error notification: %v\n", err)
						}
						fmt.Printf("Error checking %s: %v\n", refuges[i].Name, err)
						continue
					}
				}

				// Check for changes
				hasChanges := false
				for _, refuge := range refuges {
					if prevDates, exists := previousResults[refuge.Name]; exists {
						for date, status := range refuge.Dates {
							if prevStatus, exists := prevDates[date]; !exists || prevStatus != status {
								hasChanges = true
								break
							}
						}
					} else {
						hasChanges = true
					}
				}

				// Always print to console, but only send to Telegram if there are changes
				consoleMsg := formatConsoleOutput(refuges, firstDayOfMonth)
				fmt.Println(consoleMsg)

				// If there are changes, send notification to Telegram
				if hasChanges {
					telegramMsg := formatAvailabilityMessage(refuges, firstDayOfMonth)
					if *botToken != "" && len(chatIDList) > 0 {
						if err := sendTelegramMessage(*botToken, chatIDList, telegramMsg); err != nil {
							fmt.Printf("Error sending Telegram message: %v\n", err)
						} else {
							fmt.Printf("Notification sent to %d Telegram chats\n", len(chatIDList))
						}
					}

					// Update previous results
					previousResults = make(map[string]map[string]string)
					for _, refuge := range refuges {
						previousResults[refuge.Name] = make(map[string]string)
						for date, status := range refuge.Dates {
							previousResults[refuge.Name][date] = status
						}
					}
				}

				// Wait for the specified frequency before next check
				time.Sleep(time.Duration(*frequency) * time.Minute)
			}
		}
	}()

	// Wait for termination signal
	<-sigChan
	fmt.Println("\nShutting down...")

	// Send shutdown notification
	shutdownMsg := fmt.Sprintf("🛑 Monitoring stopped for %s", firstDayOfMonth.Format("January 2006"))
	if err := sendTelegramMessage(*botToken, chatIDList, shutdownMsg); err != nil {
		fmt.Printf("Error sending shutdown notification: %v\n", err)
	}

	// Signal the monitoring goroutine to stop
	done <- true
}
