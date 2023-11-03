package class

import (
	"api/account"
	"api/config"
	"api/utils"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strconv"
)

func ConvertToNewClassData(class Class) *NewClassData {
	return &NewClassData{
		TeacherID: class.TeacherID,
		Name:      class.Name,
		// We're setting StudentIDs to nil here since Class doesn't have a StudentIDs field
		// If you have a way to fetch StudentIDs based on a Class, you can use that here.
		StudentIDs: nil,
	}
}

func ConvertToGetClassData(class *Class) GetClassData {
	return GetClassData{
		ID:         class.ID,
		TeacherID:  class.TeacherID,
		Name:       class.Name,
		StudentIDs: nil, // Assuming you pass the student IDs when calling this function
	}
}

func ConvertToInt32Slice(ints []int) []int32 {
	result := make([]int32, len(ints))
	for i, v := range ints {
		result[i] = int32(v)
	}
	return result
}

type PutClassData struct {
	TeacherPassword string `json:"teacher_password"`
	StudentIDs      []int  `json:"student_ids"`
}

type DeleteClassData struct {
	TeacherPassword string `json:"teacher_password"`
	StudentIDs      *[]int `json:"student_ids,omitempty"`
}

type GetClassesData struct {
	Password string `json:"password"`
}

type PostClassData struct {
	TeacherPassword string       `json:"teacher_password"`
	Class           NewClassData `json:"class"`
}

func classInfo(c *gin.Context) {
	info := `
GET /class/<id> - get class by id

POST /class - create a new class
teacher_password: String - required
class: {
	teacher_id: i32 - required
	name: String - required
	student_ids: Vec<i32> - optional
}

PUT /class/<id> - add students to a class
teacher_password: String - required
student_ids: Vec<i32> - required

DELETE /class/<id> - delete a class
teacher_password: String - required

DELETE /class/<id> - remove students from class
teacher_password: String - required
student_ids: Vec<i32> - required
`
	c.String(http.StatusOK, info)
}

func getClassByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	classData, err := GetByID(int32(id))
	if err != nil {
		status := utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	newClassData := &NewClassData{
		TeacherID:  classData.TeacherID,
		Name:       classData.Name,
		StudentIDs: nil,
	}

	responseData, err := FromClassData(newClassData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}

	c.JSON(http.StatusOK, responseData)
}

func CreateClass(c *gin.Context) {
	var classData PostClassData
	if err := c.BindJSON(&classData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	acct, err := account.GetByID(classData.Class.TeacherID)
	if err != nil {
		log.Printf("Error getting account by ID: %v", classData.Class.TeacherID) // Логирование ошибки
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}

	if valid, err := acct.VerifyPassword(classData.TeacherPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	} else if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	classObj, err := FromClassData(&classData.Class)
	if err != nil {
		var customErr *Error
		if errors.As(err, &customErr) { // Проверяем, может ли ошибка быть приведена к типу *Error
			c.JSON(http.StatusBadRequest, gin.H{"error": customErr})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%s/class/%d", config.BaseURL, classObj.ID)})
}

func addStudents(c *gin.Context) {
	idStr := c.Param("id")
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	id := int32(idInt)

	classData, err := GetByID(id)
	if err != nil {
		c.JSON(utils.DbErrToStatus(err, http.StatusNotFound), gin.H{"error": err.Error()})
		return
	}

	acct, err := account.GetByID(classData.TeacherID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}

	var putClassData PutClassData
	if err := c.BindJSON(&putClassData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format"})
		return
	}

	isValid, err := acct.VerifyPassword(putClassData.TeacherPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	studentIDs32 := ConvertToInt32Slice(putClassData.StudentIDs)

	err = classData.AddStudents(&studentIDs32)
	var customErr *Error
	if errors.As(err, &customErr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": customErr.Error()})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": utils.InternalErr(err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Students added successfully"})
}

func DeleteClass(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid class ID"})
		return
	}

	classData, err := GetByID(int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}

	acct, err := account.GetByID(classData.TeacherID)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var deleteClassData DeleteClassData
	if err := c.BindJSON(&deleteClassData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if ok, err := acct.VerifyPassword(deleteClassData.TeacherPassword); err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	} else if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	if studentIDs := deleteClassData.StudentIDs; studentIDs != nil {
		studentIDs32 := ConvertToInt32Slice(*studentIDs)
		if err := classData.RemoveStudents(&studentIDs32); err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove students"})
			return
		}
	} else {
		if err := classData.Delete(); err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete class"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

type GetClassData struct {
	ID         int32
	TeacherID  int32
	Name       string
	StudentIDs []int32
}

func GetAccountClasses(c *gin.Context) {
	idStr := c.Param("accountParam")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var getClassesData GetClassesData
	if err := c.BindJSON(&getClassesData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	acct, err := account.GetByID(int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	isValid, err := acct.VerifyPassword(getClassesData.Password)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
		return
	}
	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	var classes []Class
	if acct.AccountType == account.Student.String() {
		classes, err = GetAllByStudentID(int32(id))
		if err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if acct.AccountType == account.Teacher.String() {
		classes, err = GetAllByTeacherID(int32(id))
		if err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Account with ID %d is neither a teacher nor a student", id)})
		return
	}

	var classDatas []GetClassData
	for _, class := range classes {
		newClassData := ConvertToNewClassData(class)
		data, err := FromClassData(newClassData)
		if err != nil {
			utils.InternalErr(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		getClassData := ConvertToGetClassData(data)
		classDatas = append(classDatas, getClassData)
	}

	c.JSON(http.StatusOK, classDatas)
}

// RegisterRoutes добавляет маршруты модуля class к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	classGroup := r.Group(config.BaseURL + "/class")
	{
		classGroup.GET("", classInfo)
		classGroup.POST("", CreateClass)
		classGroup.GET("/:id", getClassByID)
		classGroup.PUT("/:id", addStudents)
		classGroup.DELETE("/:id", DeleteClass)
	}
	accountGroup := r.Group(config.BaseURL + "/account")
	{
		accountGroup.GET("/:accountParam/classes", GetAccountClasses)
	}
}
