package models

import "time"

type FamilyEvent struct {
	ID          int          `json:"id"`
	FamilyID    int          `json:"family_id"`
	UserID      int          `json:"user_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	EventDate   time.Time    `json:"event_date"`
	Location    string       `json:"location"`
	CreatedAt   time.Time    `json:"created_at"`
	Photos      []EventPhoto `json:"photos,omitempty"`
}

type EventPhoto struct {
	ID        int       `json:"id"`
	EventID   int       `json:"event_id"`
	UserID    int       `json:"user_id"`
	PhotoPath string    `json:"photo_path"`
	PhotoURL  string    `json:"photo_url"`
	Caption   string    `json:"caption"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type PresentationRequest struct {
	EventIDs          []int `json:"event_ids"`
	MaxPhotosPerEvent int   `json:"max_photos_per_event"`
}
