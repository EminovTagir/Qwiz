package question

import (
	"api/account"
	"api/config"
	"api/media"
	"api/utils"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"time"
)

func questionInfo(c *gin.Context) {
	c.String(http.StatusOK, `
GET /question/<qwiz_id>/<index> - get question data by qwiz id and index

POST /question/<qwiz_id> - add a question to an existing qwiz
creator_password: String - required
question: {
	body: String - required,
	answer1: String - required,
	answer2: String - required,
	answer3: String - optional,
	answer4: String - optional,
	correct: 1/2/3/4 - required,
	embed: {
		data: String - required
		media_type: MediaType - required
	} - optional
} - required

PATCH /question/<qwiz_id>/<index> - update question data
creator_password: String - required
new_index: i32 - optional
new_body: String - optional
new_answers: Vector of {
	index: 1/2/3/4 - required,
	content: String - optional (null to delete)
} - optional
new_embed: {
	data: String - required
	media_type: MediaType - required
} - optional

DELETE /question/<qwiz_id> - delete question
creator_password: String - required
`)
}

type Qwiz struct {
	ID            int32     `db:"id"`
	Name          string    `db:"name"`
	CreatorID     int32     `db:"creator_id"`
	ThumbnailUUID uuid.UUID `db:"thumbnail_uuid"`
	Public        bool      `db:"public"`
	CreateTime    time.Time `db:"create_time"`
}

func GetQwizByID(id int32) (*Qwiz, error) {
	var qwiz Qwiz
	// Use the global DB variable directly without a context
	err := DB.Get(&qwiz, "SELECT * FROM qwiz WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &qwiz, nil
}

func getQuestionByQwizIDIndex(c *gin.Context) {
	qwizID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid qwiz ID"})
		return
	}

	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}

	// Retrieve question from the database
	question, err := GetQuestionByQwizIDIndex(int32(qwizID), int32(index))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.DbErrToStatus(err, http.StatusNotFound)
			c.Status(http.StatusNotFound)
		} else {
			utils.DbErrToStatus(err, http.StatusNotFound)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Convert your question to the GetQuestionData struct
	questionData, err := GetQuestionDataFromQuestion(*question)

	// Return the result
	c.JSON(http.StatusOK, questionData)
}

type GetQuestionData struct {
	Index   int32               `json:"index"`
	Body    string              `json:"body"`
	Answer1 string              `json:"answer1"`
	Answer2 string              `json:"answer2"`
	Answer3 *string             `json:"answer3"`
	Answer4 *string             `json:"answer4"`
	Embed   *media.GetMediaData `json:"embed"`
}

func GetQuestionDataFromQuestion(question Question) (*GetQuestionData, error) {
	var mediaData *media.GetMediaData
	if question.EmbedUUID != nil {
		med, err := media.GetByUUID(question.EmbedUUID) // Assuming this returns (*Media, error)
		if err != nil {
			return nil, err
		}
		// Convert *Media to *GetMediaData
		mediaData = &media.GetMediaData{
			URI:       med.URI,
			MediaType: media.Type(med.MediaType),
		}
	}

	return &GetQuestionData{
		Index:   question.Index,
		Body:    question.Body,
		Answer1: question.Answer1,
		Answer2: question.Answer2,
		Answer3: question.Answer3,
		Answer4: question.Answer4,
		Embed:   mediaData,
	}, nil
}

type PostQuestionData struct {
	CreatorPassword string          `json:"creator_password"`
	Question        NewQuestionData `json:"question"`
}

func createQuestion(c *gin.Context) {
	qwizID := c.Param("id")

	// Convert qwizID to int32 after validating
	intQwizID, err := strconv.Atoi(qwizID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID"})
		return
	}

	var questionData PostQuestionData
	if err := c.BindJSON(&questionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	quiz, err := GetQwizByID(int32(intQwizID))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Qwiz not found"})
		return
	}

	acct, err := account.GetByID(quiz.CreatorID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	isValid, err := acct.VerifyPassword(questionData.CreatorPassword)
	if err != nil {
		// Handle the error, could be a database error or something similar
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	question, err := FromQuestionData(int32(intQwizID), &questionData.Question)
	if err != nil {
		utils.DbErrToStatus(err, http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"location": fmt.Sprintf("/question/%d/%d", intQwizID, question.Index)})
}

type NewAnswer struct {
	Index   uint8   `json:"index"`
	Content *string `json:"content"`
}

type PatchQuestionData struct {
	CreatorPassword string              `json:"creator_password"`
	NewIndex        *int32              `json:"new_index"`
	NewBody         *string             `json:"new_body"`
	NewAnswers      []NewAnswer         `json:"new_answers"`
	NewCorrect      *uint8              `json:"new_correct"`
	NewEmbed        *media.NewMediaData `json:"new_embed"`
}

// updateQuestion handles PATCH requests to update a question.
func updateQuestion(c *gin.Context) {
	qwizID := c.Param("id")
	index := c.Param("index")

	var newQuestionData PatchQuestionData
	if err := c.BindJSON(&newQuestionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Convert qwizID from string to int32
	intQwizID, err := strconv.Atoi(qwizID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID"})
		return
	}

	// Convert index from string to int32
	intIndex, err := strconv.Atoi(index)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}

	question, err := GetQuestionByQwizIDIndex(int32(intQwizID), int32(intIndex))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	quiz, err := GetQwizByID(int32(intQwizID))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Qwiz not found"})
		return
	}

	acct, err := account.GetByID(quiz.CreatorID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	isValid, err := acct.VerifyPassword(newQuestionData.CreatorPassword)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if newQuestionData.NewIndex != nil {
		if _, err := question.UpdateIndex(*newQuestionData.NewIndex); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new index"})
			return
		}
	}

	if newQuestionData.NewBody != nil {
		if err := question.UpdateBody(*newQuestionData.NewBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new body"})
			return
		}
	}

	for _, newAnswer := range newQuestionData.NewAnswers {
		if _, err := question.UpdateAnswer(newAnswer.Index, newAnswer.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new answer"})
			return
		}
	}

	if newQuestionData.NewCorrect != nil {
		if err := question.UpdateCorrect(int16(*newQuestionData.NewCorrect)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new correct"})
			return
		}
	}

	if newQuestionData.NewEmbed != nil {
		if err := question.UpdateEmbed(newQuestionData.NewEmbed); err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid new embed"})
			return
		}
	}

	c.Status(http.StatusOK)
}

type DeleteQuestionData struct {
	CreatorPassword string `json:"creator_password"`
}

// deleteQuestion handles DELETE requests to delete a question.
func deleteQuestion(c *gin.Context) {
	qwizID := c.Param("id")
	index := c.Param("index")

	var deleteQuestionData DeleteQuestionData
	if err := c.BindJSON(&deleteQuestionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Convert qwizID from string to int32
	intQwizID, err := strconv.Atoi(qwizID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID"})
		return
	}

	// Convert index from string to int32
	intIndex, err := strconv.Atoi(index)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index"})
		return
	}

	question, err := GetQuestionByQwizIDIndex(int32(intQwizID), int32(intIndex))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Question not found"})
		return
	}

	quiz, err := GetQwizByID(int32(intQwizID))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	acct, err := account.GetByID(quiz.CreatorID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	isValid, err := acct.VerifyPassword(deleteQuestionData.CreatorPassword)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := question.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete question"})
		return
	}

	c.Status(http.StatusOK)
}

// RegisterRoutes добавляет маршруты модуля question к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	questionGroup := r.Group(config.BaseURL + "/question")
	{
		questionGroup.GET("", questionInfo)
		questionGroup.GET("/:id/:index", getQuestionByQwizIDIndex)
		questionGroup.POST("/:id", createQuestion)
		questionGroup.PATCH("/:id/:index", updateQuestion)
		questionGroup.DELETE("/:id/:index", deleteQuestion)
	}
}
