package handlers

import (
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"net/http"
	"strconv"
	"time"
)

func AddTransaction(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	familyID := r.Context().Value("family_id").(int)

	var trans models.Transaction
	if err := json.NewDecoder(r.Body).Decode(&trans); err != nil {
		utils.SendJSONError(w, "Invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if trans.Amount <= 0 {
		utils.SendJSONError(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	if trans.Type != "income" && trans.Type != "expense" {
		utils.SendJSONError(w, "Type must be 'income' or 'expense'", http.StatusBadRequest)
		return
	}

	// Если дата не указана, используем текущую
	if trans.Date.IsZero() {
		trans.Date = time.Now()
	}

	var id int
	err := db.DB.QueryRow(`
        INSERT INTO transactions (family_id, user_id, amount, type, category, description, date)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `, familyID, userID, trans.Amount, trans.Type, trans.Category, trans.Description, trans.Date).Scan(&id)

	if err != nil {
		utils.SendJSONError(w, "Error adding transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}

	trans.ID = id
	utils.SendJSONResponse(w, trans, http.StatusOK)
}

func GetFamilyTransactions(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")

	query := `SELECT id, user_id, amount, type, category, description, date 
              FROM transactions WHERE family_id = $1`
	args := []interface{}{familyID}
	argIndex := 2

	if month != "" && year != "" {
		query += " AND EXTRACT(MONTH FROM date) = $" + strconv.Itoa(argIndex)
		args = append(args, month)
		argIndex++
		query += " AND EXTRACT(YEAR FROM date) = $" + strconv.Itoa(argIndex)
		args = append(args, year)
	}

	query += " ORDER BY date DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.Type, &t.Category, &t.Description, &t.Date)
		transactions = append(transactions, t)
	}

	// Calculate statistics
	var totalIncome, totalExpense float64
	db.DB.QueryRow(`
        SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0),
               COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0)
        FROM transactions WHERE family_id = $1
    `, familyID).Scan(&totalIncome, &totalExpense)

	// Compare with previous month
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()

	var lastMonthExpense float64
	db.DB.QueryRow(`
        SELECT COALESCE(SUM(amount), 0) FROM transactions 
        WHERE family_id = $1 AND type = 'expense' 
        AND EXTRACT(MONTH FROM date) = $2 AND EXTRACT(YEAR FROM date) = $3
    `, familyID, currentMonth-1, currentYear).Scan(&lastMonthExpense)

	comparison := "same"
	difference := totalExpense - lastMonthExpense
	if totalExpense > lastMonthExpense {
		comparison = "higher"
	} else if totalExpense < lastMonthExpense {
		comparison = "lower"
	}

	utils.SendJSONResponse(w, map[string]interface{}{
		"transactions": transactions,
		"statistics": map[string]interface{}{
			"total_income":             totalIncome,
			"total_expense":            totalExpense,
			"balance":                  totalIncome - totalExpense,
			"comparison_to_last_month": comparison,
			"difference":               difference,
		},
	}, http.StatusOK)
}
