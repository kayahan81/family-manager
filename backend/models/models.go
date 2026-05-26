package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Family struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           int       `json:"id"`
	FamilyID     int       `json:"family_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type File struct {
	ID         int       `json:"id"`
	FamilyID   int       `json:"family_id"`
	UserID     int       `json:"user_id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	IsPrivate  bool      `json:"is_private"`
	IsPublic   bool      `json:"is_public"`
	AccessType string    `json:"access_type"`
	ShareToken *string   `json:"share_token,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Transaction struct {
	ID          int       `json:"id"`
	FamilyID    int       `json:"family_id"`
	UserID      int       `json:"user_id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

type Message struct {
	ID        int       `json:"id"`
	FamilyID  int       `json:"family_id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Device struct {
	ID        int            `json:"id"`
	FamilyID  int            `json:"family_id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Status    string         `json:"status"`
	Settings  map[string]any `json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
}

type CalendarEvent struct {
	ID          int       `json:"id"`
	FamilyID    int       `json:"family_id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	EventDate   time.Time `json:"event_date"`
	EventTime   *string   `json:"event_time"`
	CreatedAt   time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FamilyID int    `json:"family_id"`
}

type Claims struct {
	UserID   int    `json:"user_id"`
	FamilyID int    `json:"family_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
