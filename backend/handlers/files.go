package handlers

import (
	"database/sql"
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/middleware"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func UploadFile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	familyID := r.Context().Value("family_id").(int)

	// Максимальный размер файла 20 MB
	err := r.ParseMultipartForm(20 << 20)
	if err != nil {
		utils.SendJSONError(w, "File too large (max 20MB)", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		utils.SendJSONError(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	accessType := r.FormValue("access_type")
	if accessType == "" {
		accessType = "private"
	}

	// Валидация типа доступа
	if accessType != "private" && accessType != "family" && accessType != "public" {
		accessType = "private"
	}

	// Create unique filename
	ext := filepath.Ext(handler.Filename)
	uniqueName := uuid.New().String() + ext
	filePath := filepath.Join("uploads", fmt.Sprintf("family_%d", familyID), uniqueName)

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		utils.SendJSONError(w, "Error creating directory", http.StatusInternalServerError)
		return
	}

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		utils.SendJSONError(w, "Error saving file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		utils.SendJSONError(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	var shareToken interface{} = nil
	if accessType == "public" {
		token := uuid.New().String()
		shareToken = token
	}

	var fileID int
	err = db.DB.QueryRow(`
        INSERT INTO files (family_id, user_id, name, path, access_type, share_token)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `, familyID, userID, handler.Filename, filePath, accessType, shareToken).Scan(&fileID)

	if err != nil {
		os.Remove(filePath)
		utils.SendJSONError(w, "Error saving to database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":      fileID,
		"message": "File uploaded successfully",
	}
	if accessType == "public" && shareToken != nil {
		response["share_token"] = shareToken
	}

	utils.SendJSONResponse(w, response, http.StatusOK)
}

func GetFamilyFiles(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)
	userID := r.Context().Value("user_id").(int)
	role := r.Context().Value("role").(string)

	var rows *sql.Rows
	var err error

	if role == "admin" {
		rows, err = db.DB.Query(`
            SELECT id, user_id, name, access_type, share_token, created_at
            FROM files WHERE family_id = $1 ORDER BY created_at DESC
        `, familyID)
	} else {
		rows, err = db.DB.Query(`
            SELECT id, user_id, name, access_type, share_token, created_at
            FROM files WHERE family_id = $1 AND (
                access_type = 'family' OR user_id = $2 OR access_type = 'public'
            ) ORDER BY created_at DESC
        `, familyID, userID)
	}

	if err != nil {
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []map[string]interface{}
	for rows.Next() {
		var file models.File
		err := rows.Scan(&file.ID, &file.UserID, &file.Name, &file.AccessType, &file.ShareToken, &file.CreatedAt)
		if err != nil {
			continue
		}

		files = append(files, map[string]interface{}{
			"id":          file.ID,
			"user_id":     file.UserID,
			"name":        file.Name,
			"access_type": file.AccessType,
			"share_token": file.ShareToken,
			"created_at":  file.CreatedAt,
		})
	}

	utils.SendJSONResponse(w, files, http.StatusOK)
}

func DownloadFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	log.Printf("DownloadFile called for file ID: %s", fileID)
	log.Printf("Request URL: %s", r.URL.String())
	log.Printf("Query params: %v", r.URL.Query())

	// 1. Сначала пытаемся получить токен из query параметра
	token := r.URL.Query().Get("token")

	// 2. Если нет в query, пробуем из заголовка Authorization
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		log.Printf("Authorization header: %s", authHeader)
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// 3. Если токена нет, возвращаем ошибку
	if token == "" {
		log.Printf("No token found in query or header")
		utils.SendJSONError(w, "No authorization token", http.StatusUnauthorized)
		return
	}

	log.Printf("Token found: %s...", token[:min(20, len(token))])

	// Валидируем токен
	claims, err := middleware.ValidateToken(token)
	if err != nil {
		log.Printf("Token validation error: %v", err)
		utils.SendJSONError(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	log.Printf("Token validated - UserID: %d, FamilyID: %d", claims.UserID, claims.FamilyID)

	userID := claims.UserID
	familyID := claims.FamilyID

	var file models.File
	var ownerID int
	err = db.DB.QueryRow(`
        SELECT id, user_id, name, path, access_type
        FROM files WHERE id = $1 AND family_id = $2
    `, fileID, familyID).Scan(&file.ID, &ownerID, &file.Name, &file.Path, &file.AccessType)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("File not found: ID=%s, FamilyID=%d", fileID, familyID)
			utils.SendJSONError(w, "File not found", http.StatusNotFound)
		} else {
			log.Printf("Database error: %v", err)
			utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("File found: Name=%s, AccessType=%s, OwnerID=%d", file.Name, file.AccessType, ownerID)

	// Check access
	if file.AccessType == "private" && ownerID != userID {
		log.Printf("Access denied: private file owned by %d, requested by %d", ownerID, userID)
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	// Проверяем существование файла на диске
	if _, err := os.Stat(file.Path); os.IsNotExist(err) {
		log.Printf("File not found on disk: %s", file.Path)
		utils.SendJSONError(w, "File not found on server", http.StatusNotFound)
		return
	}

	log.Printf("Serving file: %s", file.Name)

	// Устанавливаем заголовки для скачивания
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")

	http.ServeFile(w, r, file.Path)
}

func UpdateFileAccess(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	role := r.Context().Value("role").(string)
	vars := mux.Vars(r)
	fileID := vars["id"]

	var req struct {
		AccessType string `json:"access_type"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Check if user owns file or is admin
	var ownerID int
	err := db.DB.QueryRow(`SELECT user_id FROM files WHERE id = $1`, fileID).Scan(&ownerID)
	if err != nil {
		utils.SendJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	if ownerID != userID && role != "admin" {
		utils.SendJSONError(w, "Access denied", http.StatusForbidden)
		return
	}

	if req.AccessType == "public" {
		token := uuid.New().String()
		_, err = db.DB.Exec(`UPDATE files SET access_type = $1, share_token = $2 WHERE id = $3`,
			req.AccessType, token, fileID)
	} else {
		_, err = db.DB.Exec(`UPDATE files SET access_type = $1, share_token = NULL WHERE id = $2`,
			req.AccessType, fileID)
	}

	if err != nil {
		utils.SendJSONError(w, "Error updating file", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]string{"message": "File access updated"}, http.StatusOK)
}

func GetPublicFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	var file models.File
	err := db.DB.QueryRow(`
        SELECT id, name, path FROM files WHERE share_token = $1 AND access_type = 'public'
    `, token).Scan(&file.ID, &file.Name, &file.Path)

	if err != nil {
		utils.SendJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Name))
	http.ServeFile(w, r, file.Path)
}
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]

	userID := r.Context().Value("user_id").(int)
	role := r.Context().Value("role").(string)
	familyID := r.Context().Value("family_id").(int)

	log.Printf("DeleteFile called - FileID: %s, UserID: %d, Role: %s", fileID, userID, role)

	// Получаем информацию о файле
	var file models.File
	var ownerID int
	var filePath string

	err := db.DB.QueryRow(`
        SELECT id, user_id, path, name, access_type
        FROM files WHERE id = $1 AND family_id = $2
    `, fileID, familyID).Scan(&file.ID, &ownerID, &filePath, &file.Name, &file.AccessType)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.SendJSONError(w, "File not found", http.StatusNotFound)
		} else {
			utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Проверяем права на удаление (только владелец или админ)
	if ownerID != userID && role != "admin" {
		utils.SendJSONError(w, "Access denied. Only owner or admin can delete this file", http.StatusForbidden)
		return
	}

	// Удаляем файл с диска
	if err := os.Remove(filePath); err != nil {
		log.Printf("Warning: Could not delete file from disk: %v", err)
		// Продолжаем, даже если файл не удалился с диска
	}

	// Удаляем запись из базы данных
	result, err := db.DB.Exec(`DELETE FROM files WHERE id = $1 AND family_id = $2`, fileID, familyID)
	if err != nil {
		utils.SendJSONError(w, "Error deleting file from database", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.SendJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	log.Printf("File deleted successfully: ID=%s, Name=%s", fileID, file.Name)
	utils.SendJSONResponse(w, map[string]interface{}{
		"message": "File deleted successfully",
		"file_id": fileID,
	}, http.StatusOK)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
