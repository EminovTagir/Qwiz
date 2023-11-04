package tests

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuestionInfo(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/question", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/question")
}

func TestGetFirstQuestion(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/question/18/0", nil)
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

func TestCreateQuestion(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса создания викторины
	createQwizData := map[string]interface{}{
		"creator_password": "Password123!",
		"question": map[string]interface{}{
			"body":    "q2",
			"answer1": "True",
			"answer2": "False",
			"answer3": "IDFK",
			"correct": 3,
		},
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(createQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal create qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/question/20", bytes.NewBuffer(data))
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

func TestCreatedFirstQuestion(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/question/18/1", nil)
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

func TestPatchQuestion(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	createQwizData := map[string]interface{}{
		"creator_password": "Password123!",
		"new_embed": map[string]interface{}{
			"data":       "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",
			"media_type": "Image",
		},
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(createQwizData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/question/18/0", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidQuestionDelete(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "Password123!",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/question/20/0", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}
