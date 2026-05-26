package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=yourpassword dbname=family_manager sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Удаляем фото
	_, err = db.Exec("TRUNCATE TABLE event_photos CASCADE")
	if err != nil {
		log.Printf("Error clearing photos: %v", err)
	}

	// Удаляем события
	_, err = db.Exec("TRUNCATE TABLE family_events CASCADE")
	if err != nil {
		log.Printf("Error clearing events: %v", err)
	}

	// Удаляем папки с картинками
	os.RemoveAll("uploads/events")
	os.MkdirAll("uploads/events", 0755)

	fmt.Println("✅ All events and photos cleared!")
}
