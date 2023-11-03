package media

import (
	"api/config"
	"api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
)

type GetMediaData struct {
	URI       string `json:"uri"`
	MediaType Type   `json:"media_type"`
}

func mediaInfo(c *gin.Context) {
	c.String(http.StatusOK, `
enum MediaType ( "Image", "Video", "Audio", "Youtube", "Gif" )
GET /media/<uuid> - get media data by uuid
`)
}

func getMediaByUUID(c *gin.Context) {
	uuidParam := c.Param("uuid")

	// Parse the UUID.
	uuidValue, err := uuid.Parse(uuidParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid UUID"})
		return
	}

	media, err := GetByUUID(&uuidValue) // Assuming you have this function
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		utils.DbErrToStatus(err, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, GetMediaData{
		URI:       media.URI,
		MediaType: Type(media.MediaType),
	})
}

// RegisterRoutes добавляет маршруты модуля media к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	mediaGroup := r.Group(config.BaseURL + "/media")
	{
		mediaGroup.GET("", mediaInfo)
		mediaGroup.GET("/:uuid", getMediaByUUID)
	}
}
