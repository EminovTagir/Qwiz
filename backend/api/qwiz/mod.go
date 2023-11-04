package qwiz

import (
	"api/media"
	"api/question"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"time"
)

type NewQwizData struct {
	Name      string              `json:"name"`
	CreatorID int32               `json:"creator_id"`
	Thumbnail *media.NewMediaData `json:"thumbnail,omitempty"`
	Public    bool                `json:"public"`
}

type Qwiz struct {
	ID            int32     `db:"id"`
	Name          string    `db:"name"`
	CreatorID     int32     `db:"creator_id"`
	ThumbnailUUID uuid.UUID `db:"thumbnail_uuid"`
	Public        bool      `db:"public"`
	CreateTime    time.Time `db:"create_time"`
}

type Error struct {
	Err error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

type SolveError struct {
	Message string
}

func (e *SolveError) Error() string {
	return e.Message
}

var DB *sqlx.DB

func GetByID(id int32) (*Qwiz, error) {
	var qwiz Qwiz
	// Use the global DB variable directly without a context
	err := DB.Get(&qwiz, "SELECT * FROM qwiz WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &qwiz, nil
}

func FromQwizData(data NewQwizData) (*Qwiz, error) {
	// Check if creator ID exists
	var accountID int32
	err := DB.Get(&accountID, "SELECT id FROM account WHERE id=$1", data.CreatorID)
	if err != nil {
		return nil, err
	}

	var thumbnailUUID *uuid.UUID // using the uuid package
	if data.Thumbnail != nil {
		mediaData, err := media.FromMediaData(data.Thumbnail) // Assuming this function does not require context
		if err != nil {
			return nil, err
		}
		thumbnailUUID = &mediaData.UUID
	}

	var qwiz Qwiz
	err = DB.Get(&qwiz, "INSERT INTO qwiz (name, creator_id, thumbnail_uuid, public) VALUES ($1, $2, $3, $4) RETURNING *",
		data.Name, data.CreatorID, thumbnailUUID, data.Public)
	if err != nil {
		return nil, err
	}
	return &qwiz, nil
}

func (qwiz *Qwiz) Delete() error {
	_, err := DB.Exec("DELETE FROM qwiz WHERE id=$1", qwiz.ID)
	return err
}

func (qwiz *Qwiz) UpdateName(newName string) error {
	row := DB.QueryRow("UPDATE qwiz SET name=$1 WHERE id=$2 RETURNING name", newName, qwiz.ID)
	err := row.Scan(&qwiz.Name)
	return err
}

// UpdateThumbnail updates or sets a new thumbnail for the Qwiz.
func (qwiz *Qwiz) UpdateThumbnail(newThumbnail media.NewMediaData) error {
	if qwiz.ThumbnailUUID != uuid.Nil {
		// Thumbnail already exists; update it
		med, err := media.GetByUUID(&qwiz.ThumbnailUUID)
		if err != nil {
			return err
		}
		err = med.Update(&newThumbnail)
		if err != nil {
			return err
		}
	} else {
		// Thumbnail does not exist; create a new one
		med, err := media.FromMediaData(&newThumbnail)
		if err != nil {
			return err
		}
		_, err = DB.Exec("UPDATE qwiz SET thumbnail_uuid=$1 WHERE id=$2", med.UUID, qwiz.ID)
		if err != nil {
			return err
		}
		qwiz.ThumbnailUUID = med.UUID
	}
	return nil
}

// GetBest retrieves the best scoring Qwizes for the given page.
func GetBest(page int64) ([]GetShortQwizData, error) {
	var qwizes []GetShortQwizData
	err := DB.Select(&qwizes, `SELECT id, name,
		(SELECT uri FROM media WHERE uuid=thumbnail_uuid) AS thumbnail_uri,
		(SELECT COUNT(*) FROM vote WHERE qwiz_id=id) AS votes,
		(SELECT username FROM account WHERE id=creator_id) AS creator_name,
		(SELECT uri FROM media WHERE uuid=(SELECT profile_picture_uuid FROM account WHERE id=creator_id)) AS creator_profile_picture_uri,
		CAST(EXTRACT(EPOCH FROM create_time) * 1000 AS BIGINT) AS create_time
		FROM qwiz WHERE public
		ORDER BY votes LIMIT 50 OFFSET $1`, page*50)
	return qwizes, err
}

// getByName retrieves qwizzes by name with pagination.
func getByName(name string, page int64) ([]GetShortQwizData, error) {
	var qwizzes []GetShortQwizData
	query := `SELECT id, name,
		(SELECT uri FROM media WHERE uuid=thumbnail_uuid) AS thumbnail_uri,
		(SELECT COUNT(*) FROM vote WHERE qwiz_id=id) AS votes,
		(SELECT username FROM account WHERE id=creator_id) AS creator_name,
		(SELECT uri FROM media WHERE uuid=(SELECT profile_picture_uuid FROM account WHERE id=creator_id)) as creator_profile_picture_uri,
		CAST(EXTRACT(EPOCH FROM create_time) * 1000 AS BIGINT) AS create_time
		FROM qwiz WHERE public AND name LIKE $1
		ORDER BY votes DESC LIMIT 50 OFFSET $2`
	err := DB.Select(&qwizzes, query, name+"%", page*50)
	if err != nil {
		return nil, err
	}
	return qwizzes, nil
}

// GetRecent retrieves recent qwizzes within a specified number of days with pagination.
func GetRecent(days uint16, page int64) ([]GetShortQwizData, error) {
	var qwizzes []GetShortQwizData
	query := `SELECT id, name,
		(SELECT uri FROM media WHERE uuid=thumbnail_uuid) AS thumbnail_uri,
		(SELECT COUNT(*) FROM vote WHERE qwiz_id=id) AS votes,
		(SELECT username FROM account WHERE id=creator_id) AS creator_name,
		(SELECT uri FROM media WHERE uuid=(SELECT profile_picture_uuid FROM account WHERE id=creator_id)) as creator_profile_picture_uri,
		CAST(EXTRACT(EPOCH FROM create_time AT TIME ZONE 'UTC') * 1000 AS BIGINT) AS create_time
		FROM qwiz WHERE public AND create_time >= (NOW() - INTERVAL '1 DAY' * $1)
		ORDER BY votes DESC LIMIT 50 OFFSET $2`
	err := DB.Select(&qwizzes, query, days, page*50)
	if err != nil {
		return nil, err
	}
	return qwizzes, nil
}

// Solve checks if the provided answers are correct for a qwiz.
func Solve(qwizID int32, answers []uint8) ([]bool, error) {
	var questions []question.Question
	err := DB.Select(&questions, "SELECT correct FROM question WHERE qwiz_id=$1 ORDER BY index", qwizID)
	if err != nil {
		return nil, err
	}

	if len(answers) > len(questions) {
		return nil, fmt.Errorf("too many answers")
	} else if len(answers) < len(questions) {
		return nil, fmt.Errorf("not enough answers")
	}

	results := make([]bool, len(answers))
	for i, answer := range answers {
		// Проверяем, что ответ входит в допустимый диапазон значений.
		if answer < 1 || answer > 4 {
			return nil, fmt.Errorf("answer %d is out of range", answer)
		}
		results[i] = answer == uint8(questions[i].Correct)
	}

	return results, nil
}
