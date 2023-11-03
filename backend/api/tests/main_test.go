package tests

import (
	"api/account"
	"api/assignment"
	"api/class"
	"api/media"
	"api/question"
	"api/qwiz"
	"api/utils"
	"api/vote"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"log"
	"os"
	"strconv"
	"strings"
)

var db *sqlx.DB

func setup() {
	// Загружаем переменные окружения из файла .env для тестовой среды
	if err := godotenv.Load("config.env"); err != nil {
		log.Fatal("Error loading .env.test file")
	}

	// Настраиваем соединение с базой данных
	databaseURL, exists := os.LookupEnv("DATABASE_URL")
	if !exists {
		log.Fatal("TEST_DATABASE_URL environment variable required but not set")
	}

	var err error
	db, err = sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}
}

func tearDown() {
	// Закрываем соединение с базой данных
	if err := db.Close(); err != nil {
		log.Fatalf("Failed to close test database connection: %v", err)
	}
}

// setupRouter функция настройки роутера для тестирования
func setupRouter() *gin.Engine {
	// Этот блок кода повторяет инициализацию из вашего основного файла main
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	limitsJson := viper.GetString("default.limits.json")
	limitValueStr := strings.TrimSuffix(limitsJson, "MiB")
	limitValue, err := strconv.Atoi(limitValueStr)
	if err != nil {
		log.Fatal(err)
	}
	byteLimit := int64(limitValue) << 20

	account.DB = db
	assignment.DB = db
	class.DB = db
	media.DB = db
	question.DB = db
	vote.DB = db
	qwiz.DB = db

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(utils.LimitRequestBody(byteLimit))
	r.Use(utils.ErrorHandlingMiddleware())
	assignment.RegisterRoutes(r) // маршруты для ассигнаций
	class.RegisterRoutes(r)      // маршруты для классов
	vote.RegisterRoutes(r)       // маршруты для голосования
	media.RegisterRoutes(r)      // маршруты для медиа
	account.RegisterRoutes(r)    // маршруты для аккаунтов и заданий
	question.RegisterRoutes(r)   // маршруты для вопросов
	qwiz.RegisterRoutes(r)       // маршруты для викторины

	return r
}
