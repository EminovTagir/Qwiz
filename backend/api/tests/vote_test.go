package tests

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVoteInfo(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/vote", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/vote")
}

func TestGetAllVotes(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/vote/20", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка кода состояния ответа
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")

	// Проверка тела ответа на наличие ошибок
	assert.NotContains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}

func TestAddVote(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса создания викторины
	createQwizData := map[string]interface{}{
		"voter_id":       "13",
		"voter_password": "Password123!",
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(createQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal create qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/vote/20", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка успешного ответа
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")
	assert.NotContains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}

func TestDeleteVote(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса создания викторины
	createQwizData := map[string]interface{}{
		"voter_id":       "13",
		"voter_password": "Password123!",
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(createQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal create qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("DELETE", "/api/vote/20", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка успешного ответа
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200")
	assert.NotContains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}
