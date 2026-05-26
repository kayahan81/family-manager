package handlers

import (
	"database/sql"
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/middleware"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your-secret-key-change-this-in-production")

func GenerateToken(userID, familyID int, role string) (string, error) {
	claims := &models.Claims{
		UserID:           userID,
		FamilyID:         familyID,
		Role:             role,
		RegisteredClaims: jwt.RegisteredClaims{},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Экспортируемая функция для валидации токена
func ValidateToken(tokenString string) (*models.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &models.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*models.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.SendJSONError(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	var userID int
	err = db.DB.QueryRow(`
        INSERT INTO users (family_id, username, email, password_hash, role)
        VALUES ($1, $2, $3, $4, 'child')
        RETURNING id
    `, req.FamilyID, req.Username, req.Email, hashedPassword).Scan(&userID)

	if err != nil {
		utils.SendJSONError(w, "User already exists", http.StatusConflict)
		return
	}

	token, err := middleware.GenerateToken(userID, req.FamilyID, "child")
	if err != nil {
		utils.SendJSONError(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"token":   token,
		"user_id": userID,
		"role":    "child",
	}, http.StatusOK)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	err := db.DB.QueryRow(`
        SELECT id, family_id, username, email, password_hash, role
        FROM users WHERE email = $1
    `, req.Email).Scan(&user.ID, &user.FamilyID, &user.Username, &user.Email, &user.PasswordHash, &user.Role)

	if err == sql.ErrNoRows {
		utils.SendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		utils.SendJSONError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.FamilyID, user.Role)
	if err != nil {
		utils.SendJSONError(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"token":     token,
		"user_id":   user.ID,
		"family_id": user.FamilyID,
		"username":  user.Username,
		"role":      user.Role,
	}, http.StatusOK)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	var user models.User
	err := db.DB.QueryRow(`
        SELECT id, family_id, username, email, role, created_at
        FROM users WHERE id = $1
    `, userID).Scan(&user.ID, &user.FamilyID, &user.Username, &user.Email, &user.Role, &user.CreatedAt)

	if err != nil {
		utils.SendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	utils.SendJSONResponse(w, user, http.StatusOK)
}
