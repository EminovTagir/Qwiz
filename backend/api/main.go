package main

import (
	"api/account"
	"api/assignment"
	"api/class"
	"api/config"
	"api/media"
	"api/question"
	"api/qwiz"
	"api/utils"
	"api/vote"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	address := viper.GetString("default.address")
	limitsJson := viper.GetString("default.limits.json")

	// Преобразование "10MiB" в 10
	limitValueStr := strings.TrimSuffix(limitsJson, "MiB")
	limitValue, err := strconv.Atoi(limitValueStr)
	if err != nil {
		// Handle error: limitValueStr was not an integer
		fmt.Printf("Error parsing limit value: %s\n", err)
		return
	}
	// Преобразование 10 в байты (10 MiB)
	byteLimit := int64(limitValue) << 20

	// Загрузка переменных окружения
	// (аналог dotenv() в Rust)
	// (предполагается, что вы используете пакет github.com/joho/godotenv)
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found")
	}

	// Настройка соединения с базой данных
	databaseURL, exists := os.LookupEnv("DATABASE_URL")
	if !exists {
		log.Fatal("Please set DATABASE_URL environment variable")
	}

	database, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Fatalf("Failed to close database connection: %v", err)
		}
	}()

	account.DB = database
	assignment.DB = database
	class.DB = database
	media.DB = database
	question.DB = database
	vote.DB = database
	qwiz.DB = database

	r := gin.Default()
	r.Use(utils.LimitRequestBody(byteLimit))
	r.Use(utils.ErrorHandlingMiddleware())

	if err := account.LoadCache(); err != nil {
		log.Fatalf("Failed to load account cache: %v", err)
	}

	// Маршруты
	api := r.Group(config.BaseURL)
	api.GET(config.BaseURL+"/", rootInfo) // корневой маршрут
	assignment.RegisterRoutes(r)          // маршруты для ассигнаций
	class.RegisterRoutes(r)               // маршруты для классов
	vote.RegisterRoutes(r)                // маршруты для голосования
	media.RegisterRoutes(r)               // маршруты для медиа
	account.RegisterRoutes(r)             // маршруты для аккаунтов и заданий
	question.RegisterRoutes(r)            // маршруты для вопросов
	qwiz.RegisterRoutes(r)                // маршруты для викторины

	err = r.Run(address)
	if err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func rootInfo(c *gin.Context) {
	c.String(http.StatusOK, `
/account
/qwiz
/question
/class
/vote
/media
`)
}
