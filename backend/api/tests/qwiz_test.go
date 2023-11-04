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

func TestSolveQwiz(t *testing.T) {
	setup()
	router := setupRouter()

	// Подготовка данных для отправки решения
	solveQwizData := map[string]interface{}{
		"answers": []int{1},
	}

	// Преобразование данных в JSON
	data, err := json.Marshal(solveQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal solve qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/qwiz/18/solve", bytes.NewBuffer(data))
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

func TestInvalidSolveQwizMany(t *testing.T) {
	setup()
	router := setupRouter()

	// Подготовка данных для отправки решения
	solveQwizData := map[string]interface{}{
		"answers": []int{1, 2},
	}

	// Преобразование данных в JSON
	data, err := json.Marshal(solveQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal solve qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/qwiz/18/solve", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка кода состояния ответа
	assert.NotEqual(t, http.StatusOK, w.Code, "Expected status code 200")

	// Проверка тела ответа на наличие ошибок
	assert.Contains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}

func TestInvalidSolveQwizNotEnough(t *testing.T) {
	setup()
	router := setupRouter()

	// Подготовка данных для отправки решения
	solveQwizData := map[string]interface{}{
		"answers": []int{},
	}

	// Преобразование данных в JSON
	data, err := json.Marshal(solveQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal solve qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/qwiz/18/solve", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка кода состояния ответа
	assert.NotEqual(t, http.StatusOK, w.Code, "Expected status code 200")

	// Проверка тела ответа на наличие ошибок
	assert.Contains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}

func TestInvalidSolveQwizAns(t *testing.T) {
	setup()
	router := setupRouter()

	// Подготовка данных для отправки решения
	solveQwizData := map[string]interface{}{
		"answers": []int{5},
	}

	// Преобразование данных в JSON
	data, err := json.Marshal(solveQwizData)
	if err != nil {
		t.Fatalf("Failed to marshal solve qwiz data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/qwiz/18/solve", bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Отправка запроса через маршрутизатор
	router.ServeHTTP(w, req)

	// Проверка кода состояния ответа
	assert.NotEqual(t, http.StatusOK, w.Code, "Expected status code 200")

	// Проверка тела ответа на наличие ошибок
	assert.Contains(t, w.Body.String(), "error", "Response body should not contain 'error'")

	defer tearDown()
}

func TestGetQwiz(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/qwiz/18", nil)
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

func TestGetBest(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/qwiz/best", nil)
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

func TestGetBestSearch(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/qwiz/best?search=test", nil)
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

func TestGetRecent(t *testing.T) {
	setup()
	router := setupRouter()

	// Выполнение запроса GET
	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/api/qwiz/recent", nil)
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

func TestInvalidQwizPatchID(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "1",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/qwiz/-1", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidQwizPatchPassword(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "1",
		"new_name":         "test qwiz 2",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/qwiz/18", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidQwizPatch(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "Password123!",
		"new_name":         "test qwiz 2",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/qwiz/19", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidQwizPatchThumbnail(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	patchQwizData := map[string]interface{}{
		"creator_password": "Password123!",
		"new_thumbnail": map[string]interface{}{
			"data":       "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",
			"media_type": "Image",
		},
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(patchQwizData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/qwiz/19", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidQwizDeleteID(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "1",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/qwiz/-1", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidQwizDeletePassword(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "1",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/qwiz/18", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidQwizDelete(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"creator_password": "Password123!",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/qwiz/18", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}
