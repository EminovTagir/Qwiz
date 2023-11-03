package qwiz

import (
	"api/account"
	"api/assignment"
	"api/config"
	"api/media"
	"api/question"
	"api/utils"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
	"time"
)

func qwizInfo(c *gin.Context) {
	c.String(http.StatusOK, `
GET /qwiz/<id> - get qwiz data by id

GET /qwiz/best?<page>&<search> - get 50 best qwizes by name, rated by votes

GET /qwiz/recent?<page> - get 50 best qwizes created in the last 2 weeks, rated by votes

POST /qwiz - create a qwiz
creator_password: String - required
qwiz: {
	name: String - required
	creator_id: i32 - required
	thumbnail_uri: String - optional
	public: bool - optional
} - required
questions: Vector of {
	body: String - required,
	answer1: String - required,
	answer2: String - required,
	answer3: String - optional,
	answer4: String - optional,
	correct: 1/2/3/4 - required,
	embed: {
		data: String - required
		media_type: MediaType - required
	} - optional,
} - required

PATCH /qwiz/<id> - update qwiz data
creator_password: String - required
new_name: String - optional
new_thumbnail: String - optional

DELETE /qwiz/<id> - delete qwiz
creator_password: String - required

POST /qwiz/<id>/solve - solve qwiz

answers: Vec<1/2/3/4> - required
`)
}

// GetFullQwizData represents a complete qwiz object in Go.
type GetFullQwizData struct {
	ID         int32                      `json:"id"`
	Name       string                     `json:"name"`
	CreatorID  int32                      `json:"creator_id"`
	Thumbnail  *media.GetMediaData        `json:"thumbnail,omitempty"`
	Questions  []question.GetQuestionData `json:"questions"`
	Public     bool                       `json:"public"`
	CreateTime int64                      `json:"create_time"`
}

// NewGetFullQwizData creates a new GetFullQwizData instance from a Qwiz struct.
func NewGetFullQwizData(qwiz Qwiz) (*GetFullQwizData, error) {
	questions, err := question.GetAllQuestionsByQwizID(qwiz.ID)
	if err != nil {
		return nil, err
	}

	var getQuestionsData []question.GetQuestionData
	for _, quest := range questions {
		getQuestionData, err := question.GetQuestionDataFromQuestion(quest)
		if err != nil {
			return nil, err // Handle error appropriately
		}
		getQuestionsData = append(getQuestionsData, *getQuestionData)
	}

	var thumbnail *media.GetMediaData
	if qwiz.ThumbnailUUID != uuid.Nil {
		mediaData, err := media.GetByUUID(&qwiz.ThumbnailUUID)
		if err == nil {
			// Convert *Media to *GetMediaData
			thumbnail = &media.GetMediaData{
				URI:       mediaData.URI,
				MediaType: media.Type(mediaData.MediaType),
			}
		}
	}

	return &GetFullQwizData{
		ID:         qwiz.ID,
		Name:       qwiz.Name,
		CreatorID:  qwiz.CreatorID,
		Thumbnail:  thumbnail,
		Questions:  getQuestionsData,
		Public:     qwiz.Public,
		CreateTime: qwiz.CreateTime.UnixNano() / int64(time.Millisecond),
	}, nil
}

func getQwizByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		// The ID is not an integer
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID format"})
		return
	}

	qwiz, err := GetByID(int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve quiz"})
		}
		utils.DbErrToStatus(err, http.StatusNotFound)
		return
	}

	// Assuming fromQwiz is a function that converts a Qwiz to GetFullQwizData
	qwizData, err := NewGetFullQwizData(*qwiz)
	if err != nil {
		// Log the error, then return a 500 internal server error to the client
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not convert quiz data"})
		return
	}

	c.JSON(http.StatusOK, qwizData)
}

// GetShortQwizData mirrors the Rust structure for serialization.
type GetShortQwizData struct {
	ID                       int32   `json:"id"`
	Name                     string  `json:"name"`
	ThumbnailURI             *string `json:"thumbnail_uri,omitempty"`
	Votes                    *int64  `json:"votes,omitempty"`
	CreatorName              *string `json:"creator_name,omitempty"`
	CreatorProfilePictureURI *string `json:"creator_profile_picture_uri,omitempty"`
	CreateTime               *int64  `json:"create_time,omitempty"`
}

// getBest handles the request for the best qwizes. It supports optional search and page parameters.
func getBestQwizes(c *gin.Context) {
	// Получаем параметры page и search из строки запроса
	page, pageErr := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	search := c.Query("search")

	// Если есть ошибка в параметре page, возвращаем ошибку клиента
	if pageErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	var qwizes []GetShortQwizData
	var err error

	// Если параметр search не пустой, ищем по имени
	if search != "" {
		qwizes, err = getByName(search, int64(page))
	} else {
		// Если параметр search пустой, получаем лучшие викторины
		qwizes, err = GetBest(int64(page))
	}

	// Обработка ошибок запроса к базе данных
	if err != nil {
		utils.DbErrToStatus(err, http.StatusBadRequest)
		// В реальном приложении стоит использовать более специфичный тип ошибки
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Отправляем данные обратно клиенту
	c.JSON(http.StatusOK, qwizes)
}

func getRecent(c *gin.Context) {
	page, err := strconv.ParseInt(c.DefaultQuery("page", "0"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	datas, err := GetRecent(14, page)
	if err != nil {
		utils.DbErrToStatus(err, http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, datas)
}

type PostQwizData struct {
	CreatorPassword string                     `json:"creator_password"`
	Qwiz            NewQwizData                `json:"qwiz"`
	Questions       []question.NewQuestionData `json:"questions"`
}

func createQwiz(c *gin.Context) {
	var qwizData PostQwizData
	if err := c.BindJSON(&qwizData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	log.Println("Before getting account by ID")
	acct, err := account.GetByID(qwizData.Qwiz.CreatorID)
	if err != nil {
		log.Printf("Error getting account by ID: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if acct == nil {
		log.Printf("Account object is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Account not found"})
		return
	}

	log.Println("Before verifying password")
	passwordIsValid, err := acct.VerifyPassword(qwizData.CreatorPassword)
	if err != nil {
		log.Printf("Error verifying password: %v", err)
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while verifying the password"})
		return
	}
	if !passwordIsValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	log.Println("Before from Qwiz Data")
	qwiz, err := FromQwizData(qwizData.Qwiz)
	if err != nil {
		utils.DbErrToStatus(err, http.StatusBadRequest)
		log.Printf("Error from Qwiz Data: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if qwiz == nil {
		log.Printf("Qwiz object is nil")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Qwiz data conversion failed"})
		return
	}

	log.Println("Before from Question Datas")
	if _, err := question.FromQuestionDatas(qwiz.ID, qwizData.Questions); err != nil {
		log.Printf("Error from Question Datas: %v", err)
		if delErr := qwiz.Delete(); delErr != nil {
			log.Printf("Error on delete Qwiz: %v", delErr)
			utils.InternalErr(err)
		}
		utils.InternalErr(err)
		utils.DbErrToStatus(err, http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Location", fmt.Sprintf("%s/qwiz/%d", config.BaseURL, qwiz.ID))
	c.Status(http.StatusCreated)
}

// PatchQwizData Define the struct for patching quiz data
type PatchQwizData struct {
	CreatorPassword string              `json:"creator_password"`
	NewName         *string             `json:"new_name"`
	NewThumbnail    *media.NewMediaData `json:"new_thumbnail"`
}

// Patch handler to update a quiz
func updateQwiz(c *gin.Context) {
	var newQwizData PatchQwizData
	if err := c.ShouldBindJSON(&newQwizData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quiz ID"})
		return
	}

	qwiz, err := GetByID(int32(id))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	acct, err := account.GetByID(qwiz.CreatorID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	passwordIsValid, err := acct.VerifyPassword(newQwizData.CreatorPassword)
	if err != nil {
		// Handle the error according to its type or log it
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred while verifying the password"})
		return
	}
	if !passwordIsValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if newQwizData.NewName != nil {
		if err := qwiz.UpdateName(*newQwizData.NewName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bad name"})
			return
		}
	}

	if newQwizData.NewThumbnail != nil {
		if err := qwiz.UpdateThumbnail(*newQwizData.NewThumbnail); err != nil {
			var mediaErr *media.Error
			if errors.As(err, &mediaErr) {
				switch *mediaErr {
				case media.SqlxError:
					utils.InternalErr(err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				case media.Base64Error:
					c.JSON(http.StatusBadRequest, gin.H{"error": "Bad thumbnail base64"})
				case media.IOError:
					utils.InternalErr(err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "IO error"})
				default:
					utils.InternalErr(err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Unknown error"})
				}
			} else {
				utils.InternalErr(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "An unknown error occurred"})
			}
			return
		}
	}

	c.Status(http.StatusOK)
}

// DeleteQwizData represents the expected JSON payload
type DeleteQwizData struct {
	CreatorPassword string `json:"creator_password"`
}

// deleteQwizHandler handles the DELETE request to remove a quiz
func deleteQwizHandler(c *gin.Context) {
	var data DeleteQwizData
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid qwiz ID"})
		return
	}

	qwiz, err := GetByID(int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.Status(http.StatusNotFound)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		utils.DbErrToStatus(err, http.StatusNotFound)
		return
	}

	acct, err := account.GetByID(qwiz.CreatorID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if ok, err := acct.VerifyPassword(data.CreatorPassword); !ok || err != nil {
		if err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		} else {
			c.Status(http.StatusUnauthorized)
		}
		return
	}

	if err := qwiz.Delete(); err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

// PostSolveQwizData Structs to bind and render data
type PostSolveQwizData struct {
	Answers  []uint8 `json:"answers"`
	Username *string `json:"username"`
}

type SolveQwizData struct {
	Correct            uint32 `json:"correct"`
	Total              uint32 `json:"total"`
	Results            []bool `json:"results"`
	AssignmentComplete *bool  `json:"assignment_complete"`
}

// solveQwiz handler function
func solveQwiz(c *gin.Context) {
	var solveQwizData PostSolveQwizData
	qwizIDStr := c.Param("qwiz_id") // Assumes qwiz_id is passed as a parameter
	assignmentID := c.Query("assignment_id")

	if err := c.ShouldBindJSON(&solveQwizData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert qwizID from string to int32
	qwizID, err := strconv.Atoi(qwizIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID format"})
		return
	}

	_, err = GetByID(int32(qwizID))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Qwiz not found"})
		return
	}

	results, err := Solve(int32(qwizID), solveQwizData.Answers)
	if err != nil {
		if err.Error() == "too many answers" || err.Error() == "not enough answers" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			// Handle other errors, possibly internal ones.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		utils.InternalErr(err)
		return
	}

	// Check if the assignment ID was provided along with a username
	if assignmentID != "" && solveQwizData.Username != nil {
		student, err := account.GetByUsername(*solveQwizData.Username)
		if err != nil {
			utils.DbErrToStatus(err, http.StatusNotFound)
			c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
			return
		}

		// Convert assignmentID from string to int
		assignID, err := strconv.Atoi(assignmentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment ID"})
			return
		}

		// Now you can use assignID as an int
		assign, err := assignment.GetByID(assignID)
		if err != nil {
			utils.DbErrToStatus(err, http.StatusNotFound)
			// Handle error, assignment not found or other errors
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		solved := utils.AllTrue(results) // A function to check if all values in the results slice are true

		if solved {
			if _, err := assign.CompleteByStudentID(int(student.ID)); err != nil {
				utils.InternalErr(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark assignment as complete"})
				return
			}
		}

		c.JSON(http.StatusOK, SolveQwizData{
			Correct:            uint32(utils.CountCorrect(results)), // A function to count true values in the results slice
			Total:              uint32(len(results)),
			Results:            results,
			AssignmentComplete: &solved,
		})
		return
	}

	// If no assignment_id is provided, or there's no username with a provided assignment_id
	c.JSON(http.StatusOK, SolveQwizData{
		Correct:            uint32(utils.CountCorrect(results)),
		Total:              uint32(len(results)),
		Results:            results,
		AssignmentComplete: nil,
	})
}

// RegisterRoutes добавляет маршруты модуля qwiz к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	qwizGroup := r.Group(config.BaseURL + "/qwiz")
	{
		qwizGroup.GET("", qwizInfo)
		qwizGroup.GET("/:id", getQwizByID)
		qwizGroup.POST("", createQwiz)
		qwizGroup.PATCH("/:id", updateQwiz)
		qwizGroup.DELETE("/:id", deleteQwizHandler)
		qwizGroup.POST("/:id/solve", solveQwiz)
		qwizGroup.GET("/best", getBestQwizes)
		qwizGroup.GET("/recent", getRecent)

	}
}
