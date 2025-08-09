package telegram

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type Response struct {
	OK     bool `json:"ok"`
	Result User `json:"result"`
}

type Message struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendMessageTo sends a message to a specific chat id
func SendMessageTo(chatID string, message string) error {
    botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
    if botToken == "" { return fmt.Errorf("TELEGRAM_BOT_TOKEN not set") }
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    resp, err := http.PostForm(apiURL, url.Values{
        "chat_id":    {chatID},
        "text":       {message},
        "parse_mode": {"HTML"},
    })
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("telegram send failed %d: %s", resp.StatusCode, string(body))
    }
    return nil
}

func SendMessage(message string) error {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	chatIDs := os.Getenv("TELEGRAM_CHAT_IDS")
	if chatIDs == "" {
		return fmt.Errorf("TELEGRAM_CHAT_IDS not set")
	}

	log.Printf("Sending Telegram message to %d recipients", len(ParseChatIDs(chatIDs)))
	log.Printf("Message content: %s", message)

	ids := ParseChatIDs(chatIDs)
	for _, chatID := range ids {
		apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
		log.Printf("Sending to chat ID: %s", chatID)

		resp, err := http.PostForm(apiURL, url.Values{
			"chat_id":    {chatID},
			"text":       {message},
			"parse_mode": {"HTML"},
		})
		if err != nil {
			log.Printf("Error sending message to %s: %v", chatID, err)
			continue
		}

		// Read and log response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response for %s: %v", chatID, err)
		} else {
			log.Printf("Telegram API response for %s: %s", chatID, string(body))
		}
		resp.Body.Close()
	}
	return nil
}

func GetUserInfo(chatID string) (string, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		return chatID, fmt.Errorf("TELEGRAM_BOT_TOKEN not set")
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", botToken)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id": {chatID},
	})
	if err != nil {
		return chatID, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return chatID, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	// For now, just return the chat ID as we don't need to parse the full response
	return chatID, nil
}

func FormatUserName(user *User) string {
	if user == nil {
		return "Unknown"
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	if user.FirstName != "" {
		if user.LastName != "" {
			return fmt.Sprintf("%s %s", user.FirstName, user.LastName)
		}
		return user.FirstName
	}
	return "Unknown"
}

// ParseChatIDs splits a comma-separated string of chat IDs into a slice
func ParseChatIDs(chatIDs string) []string {
	ids := strings.Split(chatIDs, ",")
	for i, id := range ids {
		ids[i] = strings.TrimSpace(id)
	}
	return ids
}
