package main

import (
	"family-manager/backend/db"
	"family-manager/backend/handlers"
	"family-manager/backend/middleware"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	db.InitDB()
	defer db.CloseDB()

	go handlers.StartHub()

	r := mux.NewRouter()

	// Public routes
	r.HandleFunc("/api/register", handlers.Register).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/login", handlers.Login).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/public/file/{token}", handlers.GetPublicFile).Methods("GET", "OPTIONS")

	// WebSocket route
	r.HandleFunc("/api/chat/ws", handlers.HandleWebSocket)

	// Static files
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	// Protected routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(middleware.AuthMiddleware)

	// User routes
	api.HandleFunc("/user/info", handlers.GetUserInfo).Methods("GET", "OPTIONS")

	// File routes
	api.HandleFunc("/files/upload", handlers.UploadFile).Methods("POST", "OPTIONS")
	api.HandleFunc("/files", handlers.GetFamilyFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/{id}/download", handlers.DownloadFile).Methods("GET", "OPTIONS")
	api.HandleFunc("/files/{id}/access", handlers.UpdateFileAccess).Methods("PUT", "OPTIONS")
	api.HandleFunc("/files/{id}", handlers.DeleteFile).Methods("DELETE", "OPTIONS")

	// Finance routes
	api.HandleFunc("/finance/transactions", handlers.AddTransaction).Methods("POST", "OPTIONS")
	api.HandleFunc("/finance/transactions", handlers.GetFamilyTransactions).Methods("GET", "OPTIONS")

	// Chat routes
	api.HandleFunc("/chat/messages", handlers.GetMessages).Methods("GET", "OPTIONS")

	// Smart home routes
	api.HandleFunc("/smart-home/devices", handlers.AddDevice).Methods("POST", "OPTIONS")
	api.HandleFunc("/smart-home/devices", handlers.GetDevices).Methods("GET", "OPTIONS")
	api.HandleFunc("/smart-home/devices/{id}/status", handlers.UpdateDeviceStatus).Methods("PUT", "OPTIONS")
	api.HandleFunc("/smart-home/devices/{id}", handlers.DeleteDevice).Methods("DELETE", "OPTIONS")

	// Calendar routes
	api.HandleFunc("/calendar/events", handlers.GetCalendarEvents).Methods("GET", "OPTIONS")
	api.HandleFunc("/calendar/events", handlers.AddEvent).Methods("POST", "OPTIONS")
	api.HandleFunc("/calendar/events/{id}", handlers.DeleteEvent).Methods("DELETE", "OPTIONS")

	// Family Events routes
	api.HandleFunc("/events", handlers.CreateEvent).Methods("POST", "OPTIONS")
	api.HandleFunc("/events", handlers.GetFamilyEvents).Methods("GET", "OPTIONS")
	api.HandleFunc("/events/{eventId}/photos", handlers.UploadEventPhoto).Methods("POST", "OPTIONS")
	api.HandleFunc("/events/photos/{photoId}", handlers.DeleteEventPhoto).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/events/presentation", handlers.GeneratePresentation).Methods("POST", "OPTIONS")

	// Serve frontend files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend")))

	// CORS setup
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"http://localhost:5500",
			"http://127.0.0.1:5500",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Accept"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           86400,
	})

	handler := c.Handler(r)

	log.Println("🚀 Server starting on :8080")
	log.Println("📱 Open: http://localhost:8080")
	log.Println("📅 Events page: http://localhost:8080/events.html")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
