package account

import (
	"api/config"
	"api/media"
	"api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strconv"
)

func MediaToGetMediaData(mediaInstance *media.Media) *media.GetMediaData {
	if mediaInstance == nil {
		return nil
	}
	return &media.GetMediaData{
		URI:       mediaInstance.URI,
		MediaType: media.Type(mediaInstance.MediaType),
	}
}

func accountInfo(c *gin.Context) {
	c.String(200, `
enum AccountType ( "Student", "Parent", "Teacher" )

GET /account/<id> - get account data by id
GET /account/<username> - get account data by username

POST /account - create an account
username: String - required
password: String - required
account_type: AccountType - required
profile_picture: {
	data: String - required
	media_type: MediaType - required
} - optional

PATCH /account/<id> - update account data
password: String - required
new_password: String - optional
new_account_type: AccountType - optional
new_profile_picture: {
	data: String - required
	media_type: MediaType - required
} - optional

DELETE /account/<id> - delete account
password: String - required

GET /account/<id>/classes - get student/teacher account classes
password: String - required

GET /account/<id>/assignments - get student assignments
password: String - required
`)
}

type GetAccountData struct {
	ID             int32               `json:"id"`
	Username       string              `json:"username"`
	ProfilePicture *media.GetMediaData `json:"profile_picture,omitempty"` // Значение может быть nil, если нет картинки профиля.
	AccountType    string              `json:"account_type"`
}

func fromAccount(account Account) GetAccountData {
	var profilePic *media.Media

	if account.ProfilePictureUUID != nil && *account.ProfilePictureUUID != uuid.Nil {
		uuidValue, err := uuid.Parse(account.ProfilePictureUUID.String())
		if err != nil {
			// Можно обработать ошибку или просто продолжить выполнение.
			// Здесь мы просто продолжаем выполнение, предполагая, что profilePic останется nil.
		} else {
			profilePic, _ = media.GetByUUID(&uuidValue)
			// Здесь мы также игнорируем ошибку. Если вы хотите обработать ее, можете добавить код обработки.
		}
	}

	data := GetAccountData{
		ID:             account.ID,
		Username:       account.Username,
		ProfilePicture: MediaToGetMediaData(profilePic), // Этот код предполагает, что ProfilePicture может быть типа *media.Media. Если это не так, вам может потребоваться дополнительное преобразование.
		AccountType:    account.AccountType,
	}
	return data
}

func getAccount(c *gin.Context) {
	param := c.Param("accountParam") // Получение параметра из URL
	var account *Account
	var err error

	log.Printf("Параметр для поиска: %v\n", param)
	// Пытаемся преобразовать параметр в число для поиска по ID
	if id, err := strconv.Atoi(param); err == nil {
		log.Printf("ID для поиска: %d\n", id)
		account, err = GetByID(int32(id))
	} else {
		log.Printf("Имя пользователя для поиска: %s\n", param)
		account, err = GetByUsername(param)
	}

	if account == nil {
		// Это дополнительная проверка на всякий случай.
		c.JSON(http.StatusNotFound, gin.H{"error": "Аккаунт не найден"})
		return
	}
	// Обрабатываем ошибку, если аккаунт не найден или другие ошибки базы данных
	if err != nil {
		c.JSON(utils.DbErrToStatus(err, http.StatusNotFound), gin.H{"error": err.Error()})
		return
	}

	// Проверяем и загружаем профильное изображение, если оно есть
	var mediaResult *media.Media
	if account.ProfilePictureUUID != nil && *account.ProfilePictureUUID != uuid.Nil {
		uuidValue, err := uuid.Parse(account.ProfilePictureUUID.String())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse UUID"})
			return
		}
		mediaResult, err = media.GetByUUID(&uuidValue)
		if err != nil {
			c.JSON(utils.DbErrToStatus(err, http.StatusNotFound), gin.H{"error": "Media not found"})
			return
		}
	}

	// Формируем и отправляем ответ
	accountData := GetAccountData{
		ID:             account.ID,
		Username:       account.Username,
		ProfilePicture: MediaToGetMediaData(mediaResult),
		AccountType:    account.AccountType,
	}
	c.JSON(http.StatusOK, accountData)
}

type VerifyPasswordData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func verifyPassword(c *gin.Context) {
	var data VerifyPasswordData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": "Bad Request"})
		return
	}
	account, err := GetByUsername(data.Username)
	if err != nil {
		c.JSON(utils.DbErrToStatus(err, http.StatusNotFound), gin.H{"error": err.Error()})
		return
	}
	isValid, err := account.VerifyPassword(data.Password)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(500, gin.H{"error": "Internal Server Error"})
		return
	}
	if isValid {
		c.Status(http.StatusOK)
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

type PostAccountData struct {
	Username       string              `json:"username"`
	Password       string              `json:"password"`
	AccountType    string              `json:"account_type"`
	ProfilePicture *media.NewMediaData `json:"profile_picture"`
}

func createAccount(c *gin.Context) {
	var data PostAccountData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var accountType Type
	switch data.AccountType {
	case "Student":
		accountType = Student
	case "Parent":
		accountType = Parent
	case "Teacher":
		accountType = Teacher
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный тип аккаунта"})
		return
	}
	account, err := New(data.Username, data.Password, accountType, data.ProfilePicture)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{"location": fmt.Sprintf("%s/account/%d", config.BaseURL, account.ID)})
}

type PatchAccountData struct {
	Password          string              `json:"password"`
	NewPassword       *string             `json:"new_password"`
	NewAccountType    *Type               `json:"new_account_type"`
	NewProfilePicture *media.NewMediaData `json:"new_profile_picture"`
}

func updateAccount(c *gin.Context) {
	var data PatchAccountData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	account, err := GetByID(int32(id))
	if err != nil {
		utils.DbErrToStatus(err, http.StatusNotFound)
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}
	if account == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Аккаунт не найден"})
		return
	}

	isValid, err := account.VerifyPassword(data.Password)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(500, gin.H{"error": "Internal server error"}) // Or another appropriate error message
		return
	}
	if !isValid {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if data.NewPassword != nil {
		success, err := account.UpdatePassword(*data.NewPassword)
		if err != nil || !success {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Не удалось обновить пароль"})
			return
		}
	}

	if data.NewAccountType != nil {
		if err := account.UpdateAccountType(*data.NewAccountType); err != nil {
			c.JSON(500, gin.H{"error": "Failed to update account type"})
			return
		}
	}

	if data.NewProfilePicture != nil {
		log.Printf("Update Profile Picture Request: %+v\n", data.NewProfilePicture)
	}
	if data.NewProfilePicture != nil {
		err := account.UpdateProfilePicture(data.NewProfilePicture)
		if err != nil {
			utils.InternalErr(err)
			c.JSON(500, gin.H{"error": "Failed to update profile picture"})
			return
		}
	}

	c.JSON(200, gin.H{"message": "Account updated successfully"})
}

type DeleteAccountData struct {
	Password string `json:"password"`
}

func deleteAccount(c *gin.Context) {
	var data DeleteAccountData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	account, err := GetByID(int32(id))
	if err != nil {
		utils.InternalErr(err)
		c.JSON(404, gin.H{"error": "Account not found"})
		return
	}

	isValid, err := account.VerifyPassword(data.Password)
	if err != nil {
		utils.InternalErr(err)
		c.JSON(500, gin.H{"error": "Internal server error"}) // Or another appropriate error message
		return
	}
	if !isValid {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	if err := account.Delete(); err != nil {
		utils.InternalErr(err)
		c.JSON(500, gin.H{"error": "Failed to delete account"})
		return
	}

	c.JSON(200, gin.H{"message": "Account deleted successfully"})
}

// RegisterRoutes добавляет маршруты модуля account к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	accountGroup := r.Group(config.BaseURL + "/account")
	{
		accountGroup.GET("", accountInfo)              // Получение информации об аккаунтах
		accountGroup.GET("/:accountParam", getAccount) // Получение аккаунта
		accountGroup.POST("/verify", verifyPassword)
		accountGroup.POST("", createAccount)       // Создание аккаунта
		accountGroup.PATCH("/:id", updateAccount)  // Обновление аккаунта
		accountGroup.DELETE("/:id", deleteAccount) // Удаление аккаунта
	}
}
