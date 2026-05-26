package handlers

import (
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"net/http"

	"github.com/gorilla/mux"
)

func AddDevice(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		utils.SendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	err := db.DB.QueryRow(`
        INSERT INTO devices (family_id, name, type, status, settings)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `, familyID, device.Name, device.Type, device.Status, device.Settings).Scan(&device.ID)

	if err != nil {
		utils.SendJSONError(w, "Error adding device", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, device, http.StatusOK)
}

func GetDevices(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	rows, err := db.DB.Query(`
        SELECT id, name, type, status, settings, created_at
        FROM devices WHERE family_id = $1
    `, familyID)
	if err != nil {
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		rows.Scan(&d.ID, &d.Name, &d.Type, &d.Status, &d.Settings, &d.CreatedAt)
		devices = append(devices, d)
	}

	utils.SendJSONResponse(w, devices, http.StatusOK)
}

func UpdateDeviceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]
	familyID := r.Context().Value("family_id").(int)

	var req struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	_, err := db.DB.Exec(`
        UPDATE devices SET status = $1
        WHERE id = $2 AND family_id = $3
    `, req.Status, deviceID, familyID)

	if err != nil {
		utils.SendJSONError(w, "Error updating device", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]string{"message": "Device updated"}, http.StatusOK)
}

func DeleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]
	familyID := r.Context().Value("family_id").(int)
	role := r.Context().Value("role").(string)

	if role != "admin" {
		utils.SendJSONError(w, "Admin access required", http.StatusForbidden)
		return
	}

	_, err := db.DB.Exec(`DELETE FROM devices WHERE id = $1 AND family_id = $2`, deviceID, familyID)
	if err != nil {
		utils.SendJSONError(w, "Error deleting device", http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, map[string]string{"message": "Device deleted"}, http.StatusOK)
}
