package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "host=localhost port=5432 user=postgres password=yourpassword dbname=family_manager sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Очищаем все таблицы (но не удаляем их)
	tables := []string{
		"event_photos",
		"family_events",
		"messages",
		"calendar_events",
		"devices",
		"transactions",
		"files",
		"users",
		"families",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			log.Printf("Error truncating %s: %v", table, err)
		} else {
			fmt.Printf("Cleared table: %s\n", table)
		}
	}

	fmt.Println("Database reset complete!")
}
