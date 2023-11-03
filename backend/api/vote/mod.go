package vote

import (
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

type NewVoteData struct {
	VoterID int32 `db:"voter_id"`
	QwizID  int32 `db:"qwiz_id"`
}

type Vote struct {
	VoterID int32 `db:"voter_id"`
	QwizID  int32 `db:"qwiz_id"`
}

// Standard errors
var (
	ErrQwizNotFound = errors.New("qwiz not found")
	ErrSelfVote     = errors.New("cannot vote for own qwiz")
)

// Exists checks if a vote already exists in the database.
func Exists(voterID, qwizID int32) (bool, error) {
	var exists bool
	err := DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM vote WHERE voter_id = $1 AND qwiz_id = $2)", voterID, qwizID)
	return exists, err
}

// GetByVoterIDQwizID fetches a vote from the database based on voter ID and qwiz ID.
func GetByVoterIDQwizID(voterID, qwizID int32) (*Vote, error) {
	var vote Vote
	err := DB.Get(&vote, "SELECT * FROM vote WHERE voter_id = $1 AND qwiz_id = $2", voterID, qwizID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No vote found is not an error
		}
		return nil, err
	}
	return &vote, nil
}

// GetAllByQwizID fetches all votes for a qwiz.
func GetAllByQwizID(qwizID int32) ([]Vote, error) {
	var votes []Vote
	err := DB.Select(&votes, "SELECT * FROM vote WHERE qwiz_id = $1", qwizID)
	return votes, err
}

// FromVoteData creates a vote in the database.
func FromVoteData(data NewVoteData) (*Vote, error) {
	var vote Vote
	err := DB.QueryRowx("INSERT INTO vote (voter_id, qwiz_id) VALUES ($1, $2) ON CONFLICT (voter_id, qwiz_id) DO NOTHING RETURNING *",
		data.VoterID, data.QwizID).StructScan(&vote)
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

// Delete removes a vote from the database.
func (v *Vote) Delete() error {
	_, err := DB.Exec("DELETE FROM vote WHERE voter_id = $1 AND qwiz_id = $2", v.VoterID, v.QwizID)
	return err
}
