package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
   "log"
	"github.com/gofiber/fiber/v2"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"github.com/PragaL15/go_newBackend/go_backend/db"
	Masterhandlers "github.com/PragaL15/go_newBackend/handlers/master"
)

func setupMockDB(t *testing.T) pgxmock.PgxPoolIface {
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	db.Pool = mockPool
	return mockPool
}

func setupTestApp() *fiber.App {
	app := fiber.New()
	app.Post("/violation", Masterhandlers.InsertMasterViolation)
	app.Put("/violation", Masterhandlers.UpdateMasterViolation)
	app.Delete("/violation/:id", Masterhandlers.DeleteMasterViolation)
	app.Get("/violation", Masterhandlers.GetViolations)
	return app
}

func TestInsertMasterViolation(t *testing.T) {
	app := setupTestApp()
	mockPool := setupMockDB(t)
	defer mockPool.Close()

	mockPool.ExpectExec(`CALL insert_master_violation\(\$1, \$2, \$3\)`).
		WithArgs("Violation A", "High", 1).
		WillReturnResult(pgxmock.NewResult("CALL", 1))

	payload := map[string]interface{}{
		"violation_name":   "Violation A",
		"level_of_serious": "High",
		"status":           1,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/violation", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var jsonResponse map[string]string
	json.Unmarshal(respBody, &jsonResponse)
	assert.Equal(t, "Violation added successfully", jsonResponse["message"])

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet database expectations: %v", err)
	}
}

func TestUpdateMasterViolation(t *testing.T) {
	app := setupTestApp()
	mockPool, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("Failed to create mock pool: %v", err)
	}
	defer mockPool.Close()

	log.Println("Setting up mock database expectation...")
	mockPool.ExpectExec(`CALL update_master_violation\(\$1, \$2, \$3, \$4\)`).
		WithArgs(1, "Updated Violation", "Medium", 0). 
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	payload := map[string]interface{}{
		"id":               1,
		"violation_name":   "Updated Violation",
		"level_of_serious": "Medium", 
		"status":           0, 
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/violation", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	log.Println("Sending request...")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	log.Printf("Response status: %d", resp.StatusCode)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected status code 200 but got %d", resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Response body: %s", string(respBody))

	var jsonResponse map[string]string
	if err := json.Unmarshal(respBody, &jsonResponse); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	assert.Equal(t, "Violation updated successfully", jsonResponse["message"], "Response message mismatch")

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet database expectations: %v", err)
	}
}



func TestDeleteMasterViolation(t *testing.T) {
	app := setupTestApp()
	mockPool := setupMockDB(t)
	defer mockPool.Close()

	mockPool.ExpectExec(`CALL delete_master_violation\(\$1\)`).
		WithArgs(1).
		WillReturnResult(pgxmock.NewResult("CALL", 1))

	req := httptest.NewRequest(http.MethodDelete, "/violation/1", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var jsonResponse map[string]string
	json.Unmarshal(respBody, &jsonResponse)
	assert.Equal(t, "Violation deleted successfully", jsonResponse["message"])

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet database expectations: %v", err)
	}
}

func TestGetViolations(t *testing.T) {
	app := setupTestApp()
	mockPool := setupMockDB(t)
	defer mockPool.Close()

	columns := []string{"id", "violation_name", "level_of_serious", "status"}
	mockPool.ExpectQuery(`SELECT \* FROM get_master_violations\(\)`).
		WillReturnRows(pgxmock.NewRows(columns).
			AddRow(1, "Violation A", "High", 1).
			AddRow(2, "Violation B", "Medium", 0))

	req := httptest.NewRequest(http.MethodGet, "/violation", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	respBody, _ := io.ReadAll(resp.Body)
	var jsonResponse []map[string]interface{}
	json.Unmarshal(respBody, &jsonResponse)

	assert.Len(t, jsonResponse, 2)
	assert.Equal(t, float64(1), jsonResponse[0]["id"]) 
	assert.Equal(t, "Violation A", jsonResponse[0]["violation_name"])
	assert.Equal(t, "High", jsonResponse[0]["level_of_serious"])
	assert.Equal(t, float64(1), jsonResponse[0]["status"])

	assert.Equal(t, float64(2), jsonResponse[1]["id"])
	assert.Equal(t, "Violation B", jsonResponse[1]["violation_name"])
	assert.Equal(t, "Medium", jsonResponse[1]["level_of_serious"])
	assert.Equal(t, float64(0), jsonResponse[1]["status"])

	if err := mockPool.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet database expectations: %v", err)
	}
}
