package tests

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQwizInfo(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/qwiz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/qwiz")
}

func TestCreateQwiz(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса создания викторины
	createQwizData := map[string]interface{}{
		"creator_password": "Password123!",
		"qwiz": map[string]interface{}{
			"name":       "test quiz",
			"creator_id": 13,
			"public":     false,
		},
		"questions": []map[string]interface{}{
			{
				"body":    "q1",
				"answer1": "t",
				"answer2": "f",
				"correct": 1,
			},
		},
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(createQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal create qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/qwiz", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка успешного ответа
	assert.Equal(t, http.StatusCreated, w.Code, "Expected status code 201")
	assert.NotContains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}
