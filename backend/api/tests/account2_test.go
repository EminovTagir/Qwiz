package tests

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAccountTeacher(t *testing.T) {
	setup()
	router := setupRouter() // Эта функция должна быть определена, чтобы настроить ваш маршрутизатор и все зависимости

	// Структура данных для запроса создания аккаунта
	accountData := map[string]string{
		"username":     "test_acc_2",
		"password":     "pAssword1234&",
		"account_type": "Teacher",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(accountData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/account", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusCreated, w.Code) // Убедитесь, что статус-код ответа - 201 (Created)
	assert.NotContains(t, w.Body.String(), "error")

	defer tearDown()
}

func TestGetAccountTeacherInfo(t *testing.T) {
	setup()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)          // Мы ожидаем, что статус будет OK
	assert.NotContains(t, w.Body.String(), "error") // Также ожидаем, что в теле ответа не будет слово "error"
	defer tearDown()
}

func TestGetAccountInfoByUsernameTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/test_acc_2", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)          // Мы ожидаем, что статус будет OK
	assert.NotContains(t, w.Body.String(), "error") // Также ожидаем, что в теле ответа не будет слово "error"
	defer tearDown()
}

func TestInvalidGetClassesTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "not_the_password",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/10/classes", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidGetClassesTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "pAssword1234&",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/account/10/classes", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidPatchPasswordTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password":     "1",
		"new_password": "2",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/account/10", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidPatchTypeTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password":         "pAssword1234&",
		"new_account_type": "lol",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/account/10", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestInvalidDeletePasswordTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "1",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/account/10", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.NotEqual(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
	defer tearDown()
}

func TestValidDeleteTeacher(t *testing.T) {
	setup()
	router := setupRouter()

	// Структура данных для запроса изменения пароля
	passwordData := map[string]string{
		"password": "pAssword1234&",
	}

	// Преобразование структуры данных в JSON
	data, _ := json.Marshal(passwordData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/account/10", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json") // Задаем заголовок Content-Type

	router.ServeHTTP(w, req)

	// Проверка ответа
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "error")
	defer tearDown()
}
