package parser

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Refuge struct {
	Name  string
	Dates map[string]string // date -> status
}

func ParseRefugeAvailability(baseURL string, targetDate time.Time) ([]Refuge, error) {
	log.Printf("Fetching refuge availability from %s for date %s", baseURL, targetDate.Format("2006-01-02"))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get session ID from environment
	sessionID := os.Getenv("PHPSESSID")
	if sessionID == "" {
		return nil, fmt.Errorf("PHPSESSID environment variable is not set")
	}

	// Create form data for the booking system
	formData := url.Values{}
	formData.Set("action", "availability")
	formData.Set("parent_url", "https://montblanc.ffcam.fr/GB_reservation-tout-public.html")
	formData.Set("mode", "FORM_PREBOOK")
	formData.Set("productCategory", "nomatter")
	formData.Set("pax", "1")
	formData.Set("date", targetDate.Format("2006-01-02"))

	refuges := make([]Refuge, 0)
	totalDates := 0

	// Process both refuges
	refugeIDs := map[string]string{
		"Tête Rousse": "BK_STRUCTURE:29",
		"du Goûter":   "BK_STRUCTURE:30",
	}

	for refugeName, refugeID := range refugeIDs {
		// Update structure ID for each refuge
		formData.Set("structure", refugeID)

		// Create request for availability
		req, err := http.NewRequest("POST", "https://centrale.ffcam.fr/index.php?_lang=GB", strings.NewReader(formData.Encode()))
		if err != nil {
			return nil, fmt.Errorf("error creating availability request: %v", err)
		}

		// Set essential headers
		req.Header.Set("content-type", "application/x-www-form-urlencoded")

		// Add session cookie
		req.AddCookie(&http.Cookie{
			Name:  "PHPSESSID",
			Value: sessionID,
		})

		// Send request
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s page: %v", refugeName, err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, refugeName)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s response body: %v", refugeName, err)
		}

		// Parse the HTML content
		content := string(body)
		log.Printf("Received %s response of length %d bytes at %v", refugeName, len(content), time.Now().Format("2006-01-02 15:04:05"))

		// Create refuge
		refuge := Refuge{
			Name:  refugeName,
			Dates: make(map[string]string),
		}

		// Parse HTML content
		if err := parseRefugeContent(content, &refuge); err != nil {
			log.Printf("Warning: Failed to parse HTML for %s: %v", refugeName, err)
			continue
		}

		// Check if we got any dates for this refuge
		if len(refuge.Dates) == 0 {
			log.Printf("Warning: No dates found for %s", refugeName)
			continue
		}

		// Log available dates summary
		availableDates := make([]string, 0)
		for date, status := range refuge.Dates {
			if status != "Full" {
				availableDates = append(availableDates, fmt.Sprintf("%s (%s places)", date, status))
			}
		}
		if len(availableDates) > 0 {
			log.Printf("📅 %s available dates at %v: %s", refugeName, time.Now().Format("2006-01-02 15:04:05"), strings.Join(availableDates, ", "))
		} else {
			log.Printf("❌ %s has no available dates at %v", refugeName, time.Now().Format("2006-01-02 15:04:05"))
		}

		totalDates += len(refuge.Dates)
		refuges = append(refuges, refuge)
	}

	// Check if we got any dates at all
	if totalDates == 0 {
		return nil, fmt.Errorf("no dates found for any refuge")
	}

	log.Printf("Successfully parsed %d refuges with %d total dates", len(refuges), totalDates)
	return refuges, nil
}

// parseRefugeContent parses HTML content and extracts available and full dates
func parseRefugeContent(content string, refuge *Refuge) error {
	// Parse HTML using goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %v", err)
	}

	// if content contains "Your Rank in the waiting room"
	// try again in 1 minute
	if strings.Contains(content, "Your Rank in the waiting room") {
		log.Printf("Your Rank in the waiting room, retrying in 1 minute")
		time.Sleep(1 * time.Minute)
		return parseRefugeContent(content, refuge)
	}

	// Find all available dates
	doc.Find(".day.dispo").Each(func(i int, s *goquery.Selection) {
		dateSpan := s.Find("span.date").First()
		placeSpan := s.Find("span.place").First()

		if dateSpan.Length() > 0 && placeSpan.Length() > 0 {
			date := strings.TrimSpace(dateSpan.Text())
			places := strings.TrimSpace(placeSpan.Text())

			if date != "" && places != "" {
				parts := strings.Split(date, "/")
				if len(parts) == 2 {
					month := parts[0]
					day := parts[1]
					formattedDate := fmt.Sprintf("2025-%02s-%02s", month, day)
					refuge.Dates[formattedDate] = places
					log.Printf("🎉 %s - Date %s: %s places available", refuge.Name, formattedDate, places)
				}
			}
		}
	})

	// Find all full dates
	doc.Find(".day.complet").Each(func(i int, s *goquery.Selection) {
		date := strings.TrimSpace(s.Text())
		if date != "" {
			// Convert MM/DD to YYYY-MM-DD
			parts := strings.Split(date, "/")
			if len(parts) == 2 {
				month := parts[0]
				day := parts[1]
				formattedDate := fmt.Sprintf("2025-%02s-%02s", month, day)
				refuge.Dates[formattedDate] = "Full"
			}
		}
	})

	return nil
}

func CheckAvailability(refuges []Refuge, targetDate time.Time) (bool, string) {
	log.Printf("Checking availability across all dates")

	var availableDates []string
	totalPlaces := 0

	for _, refuge := range refuges {
		for date, status := range refuge.Dates {
			if status != "Full" {
				// Convert places string to integer
				places, err := strconv.Atoi(status)
				if err != nil {
					log.Printf("Warning: Failed to parse places for %s on %s: %v", refuge.Name, date, err)
					continue
				}
				totalPlaces += places
				availableDates = append(availableDates, fmt.Sprintf("%s on %s has %d places", refuge.Name, date, places))
			}
		}
	}

	if len(availableDates) > 0 {
		return true, fmt.Sprintf("Total %d places available across all dates: %s", totalPlaces, strings.Join(availableDates, ", "))
	}

	return false, "No availability found for any date"
}
