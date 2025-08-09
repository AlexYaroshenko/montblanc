package store

import (
	"errors"
	"time"
)

// Subscriber represents a Telegram subscriber stored in the DB
type Subscriber struct {
	ChatID        string    `json:"chat_id"`
    Username      string    `json:"username"`
    FirstName     string    `json:"first_name"`
    LastName      string    `json:"last_name"`
	Language      string    `json:"language"`
	Plan          string    `json:"plan"` // free, pro (future use)
	CreatedAt     time.Time `json:"created_at"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
	IsActive      bool      `json:"is_active"`
}

// Query represents a user's monitoring request/filters
type Query struct {
	ID            string    `json:"id"`
	ChatID        string    `json:"chat_id"`
	Refuge        string    `json:"refuge"`    // "Tête Rousse" | "du Goûter" | "*"
	DateFrom      string    `json:"date_from"` // YYYY-MM-DD
	DateTo        string    `json:"date_to"`   // YYYY-MM-DD
	CreatedAt     time.Time `json:"created_at"`
	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// Store abstracts persistent storage operations
type Store interface {
	Close() error

	// Subscribers
	UpsertSubscriber(sub Subscriber) error
	GetSubscriber(chatID string) (Subscriber, error)
	ListSubscribers() ([]Subscriber, error)
	DeactivateSubscriber(chatID string) error

	// Queries
	AddQuery(q Query) (string, error)
	ListQueriesByChat(chatID string) ([]Query, error)
}

var ErrNotFound = errors.New("not found")
