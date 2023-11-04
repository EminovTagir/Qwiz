package vote

import (
	"api/account"
	"api/config"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// Assuming Vote and other related types and vars like db have been defined earlier

// GetVoteData structure for encoding the JSON response
type GetVoteData struct {
	VoterIDs []int32 `json:"voter_ids"`
}

func voteInfo(c *gin.Context) {
	c.String(http.StatusOK, `
GET /vote/<qwiz_id> - get list of voter ids by qwiz id

PUT /vote/<qwiz_id> - vote for a qwiz
voter_id: i32 - required
voter_password: String - required

DELETE /vote/<qwiz_id> - delete vote
voter_id: i32 - required
voter_password: String - required
`)
}

func fromVotes(votes []Vote) GetVoteData {
	voterIDs := make([]int32, len(votes))
	for i, vote := range votes {
		voterIDs[i] = vote.VoterID
	}
	return GetVoteData{VoterIDs: voterIDs}
}

// getVotesHandler handles the GET request to fetch votes
func getVotesHandler(c *gin.Context) {
	qwizID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid qwiz ID"})
		return
	}

	votes, err := GetAllByQwizID(int32(qwizID))
	if err != nil {
		switch {
		case errors.Is(err, ErrQwizNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Qwiz not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, fromVotes(votes))
}

// PutVoteData structure to bind the PUT request body
type PutVoteData struct {
	VoterID       string `json:"voter_id"`
	VoterPassword string `json:"voter_password"`
}

// addVoteHandler handles the PUT request to add a vote
func addVoteHandler(c *gin.Context) {
	var data PutVoteData
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	voterIDInt, err := strconv.ParseInt(data.VoterID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Voter ID"})
		return
	}

	qwizID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID"})
		return
	}

	acct, err := account.GetByID(int32(voterIDInt))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if ok, err := acct.VerifyPassword(data.VoterPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	} else if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	exists, err := Exists(int32(voterIDInt), int32(qwizID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if exists {
		c.Status(http.StatusNoContent)
		return
	}

	// This will handle inserting a new vote, assuming NewVoteData is equivalent to PutVoteData
	_, err = FromVoteData(NewVoteData{VoterID: int32(voterIDInt), QwizID: int32(qwizID)})
	if err != nil {
		switch {
		case errors.Is(err, ErrQwizNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Qwiz not found"})
		case errors.Is(err, ErrSelfVote):
			c.JSON(http.StatusForbidden, gin.H{"error": "Self-voting is not allowed"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	c.Status(http.StatusOK)
}

// DeleteVoteData structure to bind the DELETE request body
type DeleteVoteData struct {
	VoterID       string `json:"voter_id"`
	VoterPassword string `json:"voter_password"`
}

// deleteVoteHandler handles the DELETE request to remove a vote
func deleteVoteHandler(c *gin.Context) {
	var data DeleteVoteData
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	qwizID, err := strconv.ParseInt(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Qwiz ID"})
		return
	}

	voterIDInt, err := strconv.ParseInt(data.VoterID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Voter ID"})
		return
	}

	acct, err := account.GetByID(int32(voterIDInt))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if ok, err := acct.VerifyPassword(data.VoterPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	} else if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	vote, err := GetByVoterIDQwizID(int32(voterIDInt), int32(qwizID))
	if err != nil {
		if errors.Is(err, ErrQwizNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	if vote == nil {
		c.Status(http.StatusNoContent)
		return
	}

	if err := vote.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Status(http.StatusOK)
}

// RegisterRoutes добавляет маршруты модуля vote к роутеру Gin.
func RegisterRoutes(r *gin.Engine) {
	voteGroup := r.Group(config.BaseURL + "/vote")
	{
		voteGroup.GET("", voteInfo)
		voteGroup.GET("/:id", getVotesHandler)
		voteGroup.POST("/:id", addVoteHandler)
		voteGroup.DELETE("/:id", deleteVoteHandler)
	}
}
