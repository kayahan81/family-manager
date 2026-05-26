package handlers

import (
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// CreateEvent создает новое событие
func CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	familyID := r.Context().Value("family_id").(int)

	var request struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		EventDate   string `json:"event_date"`
		Location    string `json:"location"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utils.SendJSONError(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

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

	var eventID int
	err = db.DB.QueryRow(`
        INSERT INTO family_events (family_id, user_id, title, description, event_date, location)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, familyID, userID, request.Title, request.Description, eventDate, request.Location).Scan(&eventID)

	if err != nil {
		log.Printf("Error creating event: %v", err)
		utils.SendJSONError(w, "Error creating event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.FamilyEvent{
		ID:          eventID,
		FamilyID:    familyID,
		UserID:      userID,
		Title:       request.Title,
		Description: request.Description,
		EventDate:   eventDate,
		Location:    request.Location,
		CreatedAt:   time.Now(),
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

// GetFamilyEvents возвращает события семьи
func GetFamilyEvents(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")

	query := `SELECT id, user_id, title, description, event_date, location, created_at
              FROM family_events WHERE family_id = $1`
	args := []interface{}{familyID}
	argIndex := 2

	if year != "" {
		query += " AND EXTRACT(YEAR FROM event_date) = $" + strconv.Itoa(argIndex)
		args = append(args, year)
		argIndex++
	}

	if month != "" {
		query += " AND EXTRACT(MONTH FROM event_date) = $" + strconv.Itoa(argIndex)
		args = append(args, month)
		argIndex++
	}

	query += " ORDER BY event_date DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		utils.SendJSONError(w, "Database error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []models.FamilyEvent
	for rows.Next() {
		var e models.FamilyEvent
		err := rows.Scan(&e.ID, &e.UserID, &e.Title, &e.Description, &e.EventDate, &e.Location, &e.CreatedAt)
		if err != nil {
			continue
		}

		photos, _ := getEventPhotos(e.ID)
		e.Photos = photos
		events = append(events, e)
	}

	utils.SendJSONResponse(w, events, http.StatusOK)
}

// getEventPhotos возвращает фото для события
func getEventPhotos(eventID int) ([]models.EventPhoto, error) {
	rows, err := db.DB.Query(`
        SELECT id, user_id, photo_path, photo_url, caption, sort_order, created_at
        FROM event_photos WHERE event_id = $1 ORDER BY sort_order
    `, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []models.EventPhoto
	for rows.Next() {
		var p models.EventPhoto
		rows.Scan(&p.ID, &p.UserID, &p.PhotoPath, &p.PhotoURL, &p.Caption, &p.SortOrder, &p.CreatedAt)
		photos = append(photos, p)
	}
	return photos, nil
}

// UploadEventPhoto загружает фото для события
func UploadEventPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["eventId"]

	userID := r.Context().Value("user_id").(int)
	familyID := r.Context().Value("family_id").(int)

	var checkFamilyID int
	err := db.DB.QueryRow(`SELECT family_id FROM family_events WHERE id = $1`, eventID).Scan(&checkFamilyID)
	if err != nil {
		utils.SendJSONError(w, "Event not found", http.StatusNotFound)
		return
	}

	if checkFamilyID != familyID {
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	err = r.ParseMultipartForm(50 << 20)
	if err != nil {
		utils.SendJSONError(w, "File too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("photo")
	if err != nil {
		utils.SendJSONError(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(handler.Filename)
	uniqueName := uuid.New().String() + ext
	photoPath := filepath.Join("uploads", "events", fmt.Sprintf("event_%s", eventID), uniqueName)

	os.MkdirAll(filepath.Dir(photoPath), 0755)

	dst, err := os.Create(photoPath)
	if err != nil {
		utils.SendJSONError(w, "Error saving photo", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = dst.ReadFrom(file)
	if err != nil {
		utils.SendJSONError(w, "Error saving photo", http.StatusInternalServerError)
		return
	}

	caption := r.FormValue("caption")
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))

	var photoID int
	err = db.DB.QueryRow(`
        INSERT INTO event_photos (event_id, user_id, photo_path, caption, sort_order)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, eventID, userID, photoPath, caption, sortOrder).Scan(&photoID)

	if err != nil {
		os.Remove(photoPath)
		utils.SendJSONError(w, "Error saving to database", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"id":         photoID,
		"photo_path": photoPath,
		"message":    "Photo uploaded successfully",
	}, http.StatusOK)
}

// DeleteEventPhoto удаляет фото
func DeleteEventPhoto(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	photoID := vars["photoId"]

	userID := r.Context().Value("user_id").(int)
	role := r.Context().Value("role").(string)

	var photoPath string
	var ownerID int
	err := db.DB.QueryRow(`SELECT photo_path, user_id FROM event_photos WHERE id = $1`, photoID).Scan(&photoPath, &ownerID)
	if err != nil {
		utils.SendJSONError(w, "Photo not found", http.StatusNotFound)
		return
	}

	if ownerID != userID && role != "admin" {
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	os.Remove(photoPath)

	_, err = db.DB.Exec(`DELETE FROM event_photos WHERE id = $1`, photoID)
	if err != nil {
		utils.SendJSONError(w, "Error deleting photo", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]string{"message": "Photo deleted"}, http.StatusOK)
}

// GeneratePresentation создает презентацию
func GeneratePresentation(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	var req models.PresentationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) == 0 {
		utils.SendJSONError(w, "No events selected", http.StatusBadRequest)
		return
	}

	if len(req.EventIDs) > 12 {
		utils.SendJSONError(w, "Maximum 12 events allowed", http.StatusBadRequest)
		return
	}

	maxPhotos := req.MaxPhotosPerEvent
	if maxPhotos <= 0 || maxPhotos > 3 {
		maxPhotos = 3
	}

	events, err := getEventsWithPhotos(familyID, req.EventIDs, maxPhotos)
	if err != nil {
		utils.SendJSONError(w, "Error loading events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	html := generatePresentationHTML(events)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func getEventsWithPhotos(familyID int, eventIDs []int, maxPhotos int) ([]models.FamilyEvent, error) {
	query := `SELECT id, user_id, title, description, event_date, location, created_at
              FROM family_events 
              WHERE family_id = $1 AND id = ANY($2)
              ORDER BY event_date ASC`

	rows, err := db.DB.Query(query, familyID, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.FamilyEvent
	for rows.Next() {
		var e models.FamilyEvent
		err := rows.Scan(&e.ID, &e.UserID, &e.Title, &e.Description, &e.EventDate, &e.Location, &e.CreatedAt)
		if err != nil {
			continue
		}

		photoRows, err := db.DB.Query(`
            SELECT id, photo_path, caption, sort_order
            FROM event_photos 
            WHERE event_id = $1 
            ORDER BY sort_order 
            LIMIT $2
        `, e.ID, maxPhotos)
		if err == nil {
			var photos []models.EventPhoto
			for photoRows.Next() {
				var p models.EventPhoto
				photoRows.Scan(&p.ID, &p.PhotoPath, &p.Caption, &p.SortOrder)
				photos = append(photos, p)
			}
			photoRows.Close()
			e.Photos = photos
		}

		events = append(events, e)
	}

	return events, nil
}

func generatePresentationHTML(events []models.FamilyEvent) string {
	type SlideData struct {
		Type        string `json:"type"`
		Title       string `json:"title,omitempty"`
		Date        string `json:"date,omitempty"`
		Location    string `json:"location,omitempty"`
		Description string `json:"description,omitempty"`
		PhotoPath   string `json:"photoPath,omitempty"`
		Caption     string `json:"caption,omitempty"`
	}

	var slides []SlideData
	for _, event := range events {
		slides = append(slides, SlideData{
			Type:        "event",
			Title:       event.Title,
			Date:        event.EventDate.Format("02.01.2006"),
			Location:    event.Location,
			Description: event.Description,
		})

		for _, photo := range event.Photos {
			slides = append(slides, SlideData{
				Type:      "photo",
				PhotoPath: photo.PhotoPath,
				Caption:   photo.Caption,
			})
		}
	}

	slidesJSON, _ := json.Marshal(slides)

	return `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Семейная презентация</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            overflow: hidden;
        }
        .presentation-container { position: relative; width: 100vw; height: 100vh; overflow: hidden; }
        .slide {
            position: absolute; width: 100%; height: 100%; display: flex;
            flex-direction: column; justify-content: center; align-items: center;
            text-align: center; background: white; transition: transform 0.5s ease-in-out;
            transform: translateX(100%); overflow-y: auto; padding: 40px;
        }
        .slide.active { transform: translateX(0); }
        .slide.prev { transform: translateX(-100%); }
        .event-slide { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; }
        .event-slide h1 { font-size: 4rem; margin-bottom: 20px; }
        .event-slide .date { font-size: 1.5rem; margin-bottom: 20px; opacity: 0.9; }
        .event-slide .location { font-size: 1.2rem; opacity: 0.8; margin-bottom: 10px; }
        .event-slide .description { font-size: 1.2rem; max-width: 600px; margin-top: 30px; line-height: 1.6; }
        .photo-slide { background: #f5f5f5; }
        .photo-slide img { max-width: 90%; max-height: 70vh; object-fit: contain; border-radius: 10px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .photo-caption { margin-top: 20px; font-size: 1.2rem; color: #666; font-style: italic; }
        .navigation { position: fixed; bottom: 30px; left: 0; right: 0; display: flex; justify-content: center; gap: 20px; z-index: 100; }
        .nav-btn { background: rgba(255,255,255,0.9); border: none; padding: 12px 24px; border-radius: 30px; font-size: 16px; cursor: pointer; transition: transform 0.2s; }
        .nav-btn:hover { transform: scale(1.05); }
        .slide-counter { position: fixed; top: 20px; right: 20px; background: rgba(0,0,0,0.7); color: white; padding: 8px 16px; border-radius: 20px; font-size: 14px; z-index: 100; }
        .progress-bar { position: fixed; top: 0; left: 0; height: 4px; background: linear-gradient(90deg, #667eea, #764ba2); transition: width 0.3s; z-index: 100; }
        @media (max-width: 768px) {
            .event-slide h1 { font-size: 2rem; }
            .event-slide .date { font-size: 1rem; }
            .photo-caption { font-size: 0.9rem; }
            .nav-btn { padding: 8px 16px; font-size: 14px; }
        }
    </style>
</head>
<body>
    <div class="presentation-container">
        <div class="progress-bar" id="progressBar"></div>
        <div class="slide-counter" id="slideCounter">1 / 0</div>
        <div class="navigation">
            <button class="nav-btn" onclick="prevSlide()">◀ Назад</button>
            <button class="nav-btn" onclick="nextSlide()">Вперед ▶</button>
        </div>
        <div id="slides"></div>
    </div>
    <script>
        var slidesData = ` + string(slidesJSON) + `;
        var currentSlide = 0;
        var totalSlides = 0;
        
        console.log('Presentation slides:', slidesData);
        
        function initPresentation() {
            var slidesContainer = document.getElementById('slides');
            var slidesHtml = '';
            for (var i = 0; i < slidesData.length; i++) {
                var slide = slidesData[i];
                if (slide.type === 'event') {
                    slidesHtml += '<div class="slide event-slide" id="slide_' + i + '">' +
                        '<h1>📅 ' + escapeHtml(slide.title) + '</h1>' +
                        '<div class="date">' + slide.date + '</div>' +
                        (slide.location ? '<div class="location">📍 ' + escapeHtml(slide.location) + '</div>' : '') +
                        (slide.description ? '<div class="description">' + escapeHtml(slide.description) + '</div>' : '') +
                        '</div>';
                } else {
                    var photoPath = slide.photoPath;
                    var imageUrl = 'http://localhost:8080/' + photoPath;
                    console.log('Loading image:', imageUrl);
                    slidesHtml += '<div class="slide photo-slide" id="slide_' + i + '">' +
                        '<img src="' + imageUrl + '" alt="' + escapeHtml(slide.caption) + '" onerror="this.src=\'https://via.placeholder.com/800x600?text=Image+Not+Found\'">' +
                        (slide.caption ? '<div class="photo-caption">📷 ' + escapeHtml(slide.caption) + '</div>' : '') +
                        '</div>';
                }
            }
            slidesContainer.innerHTML = slidesHtml;
            totalSlides = slidesData.length;
            document.getElementById('slideCounter').innerHTML = '1 / ' + totalSlides;
            showSlide(0);
        }
        
        function showSlide(index) {
            var slides = document.querySelectorAll('.slide');
            if (slides.length === 0) return;
            if (index < 0) index = 0;
            if (index >= slides.length) index = slides.length - 1;
            for (var i = 0; i < slides.length; i++) {
                slides[i].classList.remove('active');
                slides[i].classList.remove('prev');
                if (i === index) {
                    slides[i].classList.add('active');
                } else if (i < index) {
                    slides[i].classList.add('prev');
                }
            }
            currentSlide = index;
            document.getElementById('slideCounter').innerHTML = (currentSlide + 1) + ' / ' + slides.length;
            var progress = ((currentSlide + 1) / slides.length) * 100;
            document.getElementById('progressBar').style.width = progress + '%';
        }
        
        function nextSlide() {
            var slides = document.querySelectorAll('.slide');
            if (currentSlide + 1 < slides.length) {
                showSlide(currentSlide + 1);
            }
        }
        
        function prevSlide() {
            if (currentSlide - 1 >= 0) {
                showSlide(currentSlide - 1);
            }
        }
        
        function escapeHtml(text) {
            if (!text) return '';
            var div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        
        document.addEventListener('keydown', function(e) {
            if (e.key === 'ArrowRight') nextSlide();
            if (e.key === 'ArrowLeft') prevSlide();
        });
        
        var touchStartX = 0;
        document.addEventListener('touchstart', function(e) {
            touchStartX = e.changedTouches[0].screenX;
        });
        
        document.addEventListener('touchend', function(e) {
            var touchEndX = e.changedTouches[0].screenX;
            var diff = touchEndX - touchStartX;
            if (Math.abs(diff) > 50) {
                if (diff > 0) {
                    prevSlide();
                } else {
                    nextSlide();
                }
            }
        });
        
        initPresentation();
    </script>
</body>
</html>`
}
