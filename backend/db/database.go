package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func InitDB() {
	connStr := "host=localhost port=5432 user=postgres password=81 dbname=family_manager sslmode=disable"
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to database")

	createTables()
	seedTestData()
}

func createTables() {
	fmt.Println("Creating tables if not exist...")

	queries := []string{

		`CREATE TABLE IF NOT EXISTS families (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100) NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            username VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            password_hash VARCHAR(255) NOT NULL,
            role VARCHAR(20) CHECK (role IN ('child', 'adult', 'admin')) DEFAULT 'child',
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS files (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            name VARCHAR(255) NOT NULL,
            path VARCHAR(500) NOT NULL,
            access_type VARCHAR(20) DEFAULT 'private',
            share_token VARCHAR(100) UNIQUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS transactions (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            amount DECIMAL(10,2) NOT NULL,
            type VARCHAR(20) CHECK (type IN ('income', 'expense')),
            category VARCHAR(50),
            description TEXT,
            date DATE DEFAULT CURRENT_DATE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS messages (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            username VARCHAR(50),
            message TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS devices (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            name VARCHAR(100) NOT NULL,
            type VARCHAR(50),
            status VARCHAR(20) DEFAULT 'off',
            settings JSONB,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,

		`CREATE TABLE IF NOT EXISTS calendar_events (
            id SERIAL PRIMARY KEY,
            family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
            user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
            title VARCHAR(200) NOT NULL,
            description TEXT,
            event_date DATE NOT NULL,
            event_time TIME,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS family_events (
		    id SERIAL PRIMARY KEY,
		    family_id INTEGER REFERENCES families(id) ON DELETE CASCADE,
		    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		    title VARCHAR(200) NOT NULL,
		    description TEXT,
		    event_date DATE NOT NULL,
		    location VARCHAR(200),
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS event_photos (
		    id SERIAL PRIMARY KEY,
		    event_id INTEGER REFERENCES family_events(id) ON DELETE CASCADE,
		    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
		    photo_path VARCHAR(500) NOT NULL,
		    photo_url VARCHAR(500),
		    caption VARCHAR(500),
		    sort_order INTEGER DEFAULT 0,
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_family_events_family_id ON family_events(family_id);
		CREATE INDEX IF NOT EXISTS idx_family_events_event_date ON family_events(event_date);
		CREATE INDEX IF NOT EXISTS idx_event_photos_event_id ON event_photos(event_id);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			log.Printf("Error executing query: %v\nQuery: %s", err, query)
		}
	}

	fmt.Println("Tables created successfully")
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return ""
	}
	return string(hash)
}

func seedTestData() {
	fmt.Println("Checking if test data exists...")

	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM families").Scan(&count)
	if err != nil {
		log.Printf("Error checking families count: %v", err)
		return
	}

	if count > 0 {
		fmt.Printf("Test data already exists (%d families), skipping seed\n", count)
		return
	}

	fmt.Println("Seeding test data...")

	passwordHash := hashPassword("password123")
	if passwordHash == "" {
		log.Fatal("Failed to generate password hash")
	}
	fmt.Printf("Generated hash: %s\n", passwordHash)

	families := []string{"Ивановы", "Петровы", "Сидоровы"}
	for _, name := range families {
		_, err := DB.Exec("INSERT INTO families (name) VALUES ($1)", name)
		if err != nil {
			log.Printf("Error inserting family %s: %v", name, err)
		}
	}
	fmt.Println("Families added")

	var familyIDs []int
	rows, err := DB.Query("SELECT id FROM families ORDER BY id")
	if err != nil {
		log.Printf("Error getting family IDs: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		rows.Scan(&id)
		familyIDs = append(familyIDs, id)
	}

	if len(familyIDs) < 3 {
		log.Printf("Warning: Only got %d family IDs", len(familyIDs))
		familyIDs = []int{1, 2, 3} // Используем ID по умолчанию
	}

	// 2. Добавляем пользователей
	users := []struct {
		familyID int
		username string
		email    string
		role     string
	}{
		{familyIDs[0], "admin_ivanov", "admin@ivanov.com", "admin"},
		{familyIDs[0], "adult_ivanov", "adult@ivanov.com", "adult"},
		{familyIDs[0], "child_ivanov", "child@ivanov.com", "child"},
		{familyIDs[0], "mom_ivanov", "mom@ivanov.com", "adult"},
		{familyIDs[0], "dad_ivanov", "dad@ivanov.com", "adult"},
		{familyIDs[1], "admin_petrov", "admin@petrov.com", "admin"},
		{familyIDs[1], "adult_petrov", "adult@petrov.com", "adult"},
		{familyIDs[1], "child_petrov", "child@petrov.com", "child"},
		{familyIDs[2], "admin_sidorov", "admin@sidorov.com", "admin"},
		{familyIDs[2], "adult_sidorov", "adult@sidorov.com", "adult"},
	}

	for _, u := range users {
		_, err := DB.Exec(`
            INSERT INTO users (family_id, username, email, password_hash, role)
            VALUES ($1, $2, $3, $4, $5)
        `, u.familyID, u.username, u.email, passwordHash, u.role)
		if err != nil {
			log.Printf("Error inserting user %s: %v", u.username, err)
		}
	}
	fmt.Println("✅ Users added")

	// Получаем ID пользователей для семьи Ивановы
	var userIDs map[string]int = make(map[string]int)
	userRows, err := DB.Query(`
        SELECT id, username FROM users WHERE family_id = $1
    `, familyIDs[0])
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var id int
			var username string
			userRows.Scan(&id, &username)
			userIDs[username] = id
		}
	}

	// 3. Добавляем транзакции
	if adultID, ok := userIDs["adult_ivanov"]; ok {
		transactions := []struct {
			amount      float64
			transType   string
			category    string
			description string
			daysAgo     int
		}{
			{50000, "income", "salary", "Зарплата", 30},
			{15000, "expense", "food", "Продукты", 25},
			{5000, "expense", "bills", "Коммунальные платежи", 20},
			{3000, "expense", "transport", "Бензин", 15},
			{2000, "expense", "entertainment", "Кино", 12},
			{55000, "income", "salary", "Зарплата", 2},
			{18000, "expense", "food", "Продукты", 3},
			{6000, "expense", "bills", "Коммунальные платежи", 5},
		}

		for _, t := range transactions {
			_, err := DB.Exec(`
                INSERT INTO transactions (family_id, user_id, amount, type, category, description, date)
                VALUES ($1, $2, $3, $4, $5, $6, CURRENT_DATE - ($7 || ' days')::INTERVAL)
            `, familyIDs[0], adultID, t.amount, t.transType, t.category, t.description, t.daysAgo)
			if err != nil {
				log.Printf("Error inserting transaction: %v", err)
			}
		}
		fmt.Println("✅ Transactions added")
	}

	// 4. Добавляем сообщения
	messages := []struct {
		username string
		message  string
	}{
		{"admin_ivanov", "Добро пожаловать в семейный чат!"},
		{"adult_ivanov", "Кто будет забирать детей из школы?"},
		{"child_ivanov", "Я пришел!"},
		{"mom_ivanov", "Не забудьте купить хлеб"},
		{"dad_ivanov", "Уже купил 👍"},
	}

	for _, m := range messages {
		if userID, ok := userIDs[m.username]; ok {
			_, err := DB.Exec(`
                INSERT INTO messages (family_id, user_id, username, message)
                VALUES ($1, $2, $3, $4)
            `, familyIDs[0], userID, m.username, m.message)
			if err != nil {
				log.Printf("Error inserting message: %v", err)
			}
		}
	}
	fmt.Println("Messages added")

	// 5. Добавляем устройства
	devices := []struct {
		name   string
		dtype  string
		status string
	}{
		{"Гостиная", "light", "off"},
		{"Кухня", "light", "on"},
		{"Спальня", "light", "off"},
		{"Термостат", "thermostat", "on"},
		{"Входная дверь", "lock", "on"},
	}

	for _, d := range devices {
		_, err := DB.Exec(`
            INSERT INTO devices (family_id, name, type, status, settings)
            VALUES ($1, $2, $3, $4, '{}'::jsonb)
        `, familyIDs[0], d.name, d.dtype, d.status)
		if err != nil {
			log.Printf("Error inserting device: %v", err)
		}
	}
	fmt.Println("Devices added")

	if adultID, ok := userIDs["adult_ivanov"]; ok {
		events := []struct {
			userID    int
			title     string
			desc      string
			daysAhead int
			eventTime string
		}{
			{adultID, "День рождения мамы", "Нужно купить торт и цветы", 10, "12:00"},
			{adultID, "Родительское собрание", "Школа №123", 5, "18:30"},
			{userIDs["child_ivanov"], "Контрольная по математике", "Подготовиться", 8, "10:00"},
			{userIDs["mom_ivanov"], "Встреча с друзьями", "Кафе 'Уют'", 6, "19:00"},
			{userIDs["admin_ivanov"], "Плановое собрание семьи", "Обсудить бюджет", 12, "20:00"},
		}

		for _, e := range events {
			_, err := DB.Exec(`
                INSERT INTO calendar_events (family_id, user_id, title, description, event_date, event_time)
                VALUES ($1, $2, $3, $4, CURRENT_DATE + ($5 || ' days')::INTERVAL, $6)
            `, familyIDs[0], e.userID, e.title, e.desc, e.daysAhead, e.eventTime)
			if err != nil {
				log.Printf("Error inserting event: %v", err)
			}
		}
		fmt.Println("✅ Calendar events added")
	}

	// Проверка результата
	var familyCount, userCount, transCount, msgCount, deviceCount, eventCount int
	DB.QueryRow("SELECT COUNT(*) FROM families").Scan(&familyCount)
	DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	DB.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&transCount)
	DB.QueryRow("SELECT COUNT(*) FROM messages").Scan(&msgCount)
	DB.QueryRow("SELECT COUNT(*) FROM devices").Scan(&deviceCount)
	DB.QueryRow("SELECT COUNT(*) FROM calendar_events").Scan(&eventCount)

	fmt.Println("\n📊 Database Statistics:")
	fmt.Printf("   Families: %d\n", familyCount)
	fmt.Printf("   Users: %d\n", userCount)
	fmt.Printf("   Transactions: %d\n", transCount)
	fmt.Printf("   Messages: %d\n", msgCount)
	fmt.Printf("   Devices: %d\n", deviceCount)
	fmt.Printf("   Calendar Events: %d\n", eventCount)

	if familyCount > 0 && userCount > 0 {
		fmt.Println("\nTest data seeded successfully!")
		fmt.Println("\nTest accounts (password: password123):")

		// Выводим список пользователей
		userRows, err := DB.Query("SELECT email, role FROM users ORDER BY family_id, id")
		if err == nil {
			defer userRows.Close()
			for userRows.Next() {
				var email, role string
				userRows.Scan(&email, &role)
				roleIcon := "User"
				if role == "admin" {
					roleIcon = "Admin"
				} else if role == "adult" {
					roleIcon = "Adult"
				} else {
					roleIcon = "Child"
				}
				fmt.Printf("   %s %s / %s\n", roleIcon, email, role)
			}
		}
	} else {
		fmt.Println("\nFailed to seed test data! Check errors above.")
	}
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
