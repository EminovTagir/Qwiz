package assignment

import (
	"api/account"
	"api/config"
	"api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type GetAssignmentData struct {
	QwizID    int    `json:"qwiz_id"`
	ClassID   int    `json:"class_id"`
	OpenTime  *int64 `json:"open_time"`
	CloseTime *int64 `json:"close_time"`
	Completed bool   `json:"completed"`
}

type GetAssignmentsData struct {
	Password string `json:"password"`
}

func GetAccountAssignments(c *gin.Context) {
	var data GetAssignmentsData
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	id := c.Param("id")
	intID, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	acct, err := account.GetByID(int32(intID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		utils.DbErrToStatus(err, http.StatusNotFound)
		return
	}

	isValid, err := acct.VerifyPassword(data.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		utils.InternalErr(err)
		return
	}

	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if acct.AccountType != account.Student.String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not a student"})
		return
	}

	assignments, err := GetAllByStudentID(int(acct.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		utils.InternalErr(err)
		return
	}

	var result []GetAssignmentData
	for _, a := range assignments {
		var openTime, closeTime *int64

		if a.OpenTime != nil {
			t := a.OpenTime.Unix() * 1000 // Convert to milliseconds
			openTime = &t
		}

		if a.CloseTime != nil {
			t := a.CloseTime.Unix() * 1000 // Convert to milliseconds
			closeTime = &t
		}

		result = append(result, GetAssignmentData{
			QwizID:    a.QwizID,
			ClassID:   a.ClassID,
			OpenTime:  openTime,
			CloseTime: closeTime,
			Completed: a.Completed.Value,
		})
	}

	c.JSON(http.StatusOK, result)
}

// RegisterRoutes добавляет маршруты модуля assignment к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	accountGroup := r.Group(config.BaseURL + "/account")
	{
		accountGroup.GET("/:accountParam/assignments", GetAccountAssignments)
	}
}
