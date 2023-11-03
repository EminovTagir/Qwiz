package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassInfo(t *testing.T) {
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/class", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/class")
}

func TestValidCreateClass(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса создания класса
	createClassData := map[string]interface{}{
		"teacher_password": "pAssword1234&",
		"class": map[string]interface{}{
			"teacher_id": 11,
			"name":       "test class",
		},
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(createClassData)
	if err != nil {
		t.Fatalf("Failed to marshal create class data: %v", err)
	}

	// Выполнение запроса POST
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/api/class", bytes.NewBuffer(data))
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

func TestGetClass(t *testing.T) {
	setup()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/class/5", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)          // Мы ожидаем, что статус будет OK
	assert.NotContains(t, w.Body.String(), "error") // Также ожидаем, что в теле ответа не будет слово "error"
	defer tearDown()
}

func TestAddStudentsToClass(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса добавления студентов в класс
	addStudentsData := map[string]interface{}{
		"teacher_password": "pAssword1234&", // Замените на реальный пароль
		"student_ids":      []int32{13},     // Замените на реальные ID студентов
	}

	// Преобразование структуры данных в JSON
	data, err := json.Marshal(addStudentsData)
	if err != nil {
		t.Fatalf("Failed to marshal add students data: %v", err)
	}

	// Выполнение запроса PUT
	w := httptest.NewRecorder()
	req, err := http.NewRequest("PUT", fmt.Sprintf("/api/class/5"), bytes.NewBuffer(data))
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

func TestGetStudentClasses(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "Password123!",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/13/classes", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestGetTeacherClasses(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "pAssword1234&",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/11/classes", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}
