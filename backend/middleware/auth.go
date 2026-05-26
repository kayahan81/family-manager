package middleware

import (
	"context"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
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

	return nil, err
}

// Middleware function for Gorilla Mux
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.SendJSONError(w, "No authorization token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := ValidateToken(tokenString)
		if err != nil {
			utils.SendJSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "family_id", claims.FamilyID)
		ctx = context.WithValue(ctx, "role", claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function to create admin-only middleware
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role").(string)
		if role != "admin" {
			utils.SendJSONError(w, "Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Helper function to create adult-only middleware
func AdultOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value("role").(string)
		if role == "child" {
			utils.SendJSONError(w, "Adult access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
