package handlers

import (
	"database/sql"
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func AddEvent(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	familyID := r.Context().Value("family_id").(int)

	var request struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		EventDate   string `json:"event_date"`
		EventTime   string `json:"event_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.SendJSONError(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if request.Title == "" {
		utils.SendJSONError(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Парсим дату из формата YYYY-MM-DD
	eventDate, err := time.Parse("2006-01-02", request.EventDate)
	if err != nil {
		utils.SendJSONError(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	var eventTime interface{}
	if request.EventTime != "" {
		eventTime = request.EventTime
	} else {
		eventTime = nil
	}

	var eventID int
	err = db.DB.QueryRow(`
        INSERT INTO calendar_events (family_id, user_id, title, description, event_date, event_time)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, familyID, userID, request.Title, request.Description, eventDate, eventTime).Scan(&eventID)

	if err != nil {
		log.Printf("Error adding event: %v", err)
		utils.SendJSONError(w, "Error adding event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.CalendarEvent{
		ID:          eventID,
		FamilyID:    familyID,
		UserID:      userID,
		Title:       request.Title,
		Description: request.Description,
		EventDate:   eventDate,
		CreatedAt:   time.Now(),
	}

	if request.EventTime != "" {
		response.EventTime = &request.EventTime
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	// Получаем параметры месяца и года
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	// Парсим параметры
	month := -1
	year := -1

	if monthStr != "" {
		m, err := strconv.Atoi(monthStr)
		if err == nil {
			month = m
		}
	}

	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err == nil {
			year = y
		}
	}

	// Строим запрос
	query := `SELECT id, user_id, title, description, event_date, event_time, created_at
              FROM calendar_events WHERE family_id = $1`
	args := []interface{}{familyID}
	argIndex := 2

	if month > 0 && year > 0 {
		query += " AND EXTRACT(MONTH FROM event_date) = $" + strconv.Itoa(argIndex)
		args = append(args, month)
		argIndex++
		query += " AND EXTRACT(YEAR FROM event_date) = $" + strconv.Itoa(argIndex)
		args = append(args, year)
	}

	query += " ORDER BY event_date ASC, event_time ASC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Database error: %v", err)
		utils.SendJSONError(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []models.CalendarEvent
	for rows.Next() {
		var e models.CalendarEvent
		var timePtr sql.NullString

		err := rows.Scan(&e.ID, &e.UserID, &e.Title, &e.Description, &e.EventDate, &timePtr, &e.CreatedAt)
		if err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}

		if timePtr.Valid && timePtr.String != "" {
			e.EventTime = &timePtr.String
		}

		events = append(events, e)
	}

	utils.SendJSONResponse(w, events, http.StatusOK)
}

func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]
	userID := r.Context().Value("user_id").(int)
	role := r.Context().Value("role").(string)

	// Проверяем существование события и права доступа
	var ownerID int
	var familyID int
	err := db.DB.QueryRow(`SELECT user_id, family_id FROM calendar_events WHERE id = $1`, eventID).Scan(&ownerID, &familyID)
	if err != nil {
		utils.SendJSONError(w, "Event not found", http.StatusNotFound)
		return
	}

	currentFamilyID := r.Context().Value("family_id").(int)
	if familyID != currentFamilyID {
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	if ownerID != userID && role != "admin" {
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	_, err = db.DB.Exec(`DELETE FROM calendar_events WHERE id = $1`, eventID)
	if err != nil {
		log.Printf("Error deleting event: %v", err)
		utils.SendJSONError(w, "Error deleting event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]string{"message": "Event deleted successfully"}, http.StatusOK)
}
