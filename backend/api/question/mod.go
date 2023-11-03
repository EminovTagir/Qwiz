package question

import (
	"api/media"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"log"
)

type NewQuestionData struct {
	Index     *int32
	Body      string
	Answer1   string
	Answer2   string
	Answer3   *string
	Answer4   *string
	Correct   int16
	EmbedData *media.NewMediaData
}

type Error struct {
	Kind      errorType
	SqlxErr   error
	Base64Err base64.CorruptInputError
	IOErr     error
}

var DB *sqlx.DB

func GetQuestionByQwizIDIndex(qwizID int32, index int32) (*Question, error) {
	var q Question
	err := DB.Get(&q, "SELECT * FROM question WHERE qwiz_id=$1 AND index=$2", qwizID, index)
	return &q, err
}

func GetAllQuestionsByQwizID(qwizID int32) ([]Question, error) {
	var questions []Question
	err := DB.Select(&questions, "SELECT * FROM question WHERE qwiz_id=$1 ORDER BY index", qwizID)
	return questions, err
}

type errorType int

const (
	SqlxErrorType errorType = iota
	Base64ErrorType
	IOErrorType
)

func (e Error) Error() string {
	switch e.Kind {
	case SqlxErrorType:
		return e.SqlxErr.Error()
	case Base64ErrorType:
		return e.Base64Err.Error()
	case IOErrorType:
		return e.IOErr.Error()
	default:
		return "Unknown error"
	}
}

type Question struct {
	QwizID    int32
	Index     int32
	Body      string
	Answer1   string
	Answer2   string
	Answer3   *string
	Answer4   *string
	Correct   int16
	EmbedUUID *uuid.UUID
}

func FromQuestionData(qwizID int32, data *NewQuestionData) (*Question, error) {
	var embedUUID *uuid.UUID
	if data.EmbedData != nil {
		med, err := media.FromMediaData(data.EmbedData)
		if err != nil {
			log.Println(err)
		} else {
			embedUUID = &med.UUID
		}
	}

	var realIndex int32
	if data.Index != nil {
		// shift all existing questions after current index by 1
		row := DB.QueryRow(`UPDATE question SET index=index+1 WHERE index>= $1 AND qwiz_id=$2 RETURNING index`, data.Index, qwizID)
		if err := row.Scan(&realIndex); err != nil {
			return nil, err
		}
	} else {
		row := DB.QueryRow(`SELECT COALESCE(MAX(index) + 1, 0) FROM question WHERE qwiz_id=$1`, qwizID)
		if err := row.Scan(&realIndex); err != nil {
			return nil, err
		}
	}

	q := &Question{}
	err := DB.QueryRow(`INSERT INTO question (qwiz_id, index, body, answer1, answer2, answer3, answer4, correct, embed_uuid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *`,
		qwizID, realIndex, data.Body, data.Answer1, data.Answer2, data.Answer3, data.Answer4, data.Correct, embedUUID).Scan(
		&q.QwizID, &q.Index, &q.Body, &q.Answer1, &q.Answer2, &q.Answer3, &q.Answer4, &q.Correct, &q.EmbedUUID)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func FromQuestionDatas(qwizID int32, datas []NewQuestionData) ([]Question, error) {
	var indexes []int32
	var bodies, answers1, answers2, answers3, answers4 []string
	var corrects []int16
	var medias []*media.NewMediaData

	for _, d := range datas {
		indexes = append(indexes, int32(len(indexes)))
		bodies = append(bodies, d.Body)
		answers1 = append(answers1, d.Answer1)
		answers2 = append(answers2, d.Answer2)

		if d.Answer3 != nil {
			answers3 = append(answers3, *d.Answer3)
		} else {
			answers3 = append(answers3, "")
		}

		if d.Answer4 != nil {
			answers4 = append(answers4, *d.Answer4)
		} else {
			answers4 = append(answers4, "")
		}

		corrects = append(corrects, d.Correct)
		medias = append(medias, d.EmbedData)
	}

	log.Printf("Indexes: %v", indexes)
	log.Printf("Bodies: %v", bodies)

	embedUUIDs, err := media.FromMediaDatas(medias)
	if err != nil {
		return nil, err
	}

	log.Printf("Executing query with qwizID: %d and data: %v", qwizID, indexes)

	rows, err := DB.Query(`INSERT INTO question (qwiz_id, index, body, answer1, answer2, answer3, answer4, correct, embed_uuid)
		SELECT $1, index, body, answer1, answer2, NULLIF(answer3, ''), NULLIF(answer4, ''), correct, NULLIF(embed_uuid, uuid_nil())
		FROM UNNEST($2, $3, $4, $5, $6, $7, $8, $9)
		AS t(index, body, answer1, answer2, answer3, answer4, correct, embed_uuid)
		RETURNING *`, qwizID, indexes, bodies, answers1, answers2, answers3, answers4, corrects, embedUUIDs)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var result []Question
	for rows.Next() {
		var q Question
		if err := rows.Scan(&q.QwizID, &q.Index, &q.Body, &q.Answer1, &q.Answer2, &q.Answer3, &q.Answer4, &q.Correct, &q.EmbedUUID); err != nil {
			log.Printf("Scan error: %v", err)
			return nil, err
		}
		log.Printf("Question retrieved: %v", q)
		result = append(result, q)
	}

	log.Printf("Questions inserted successfully: %v", result)
	return result, nil
}

func (q *Question) Delete() error {
	_, err := DB.Exec(`WITH deleted AS (
		DELETE FROM question WHERE qwiz_id=$1 AND index=$2 RETURNING qwiz_id, index
	) UPDATE question SET index=index-1 WHERE index>(SELECT index FROM deleted) AND qwiz_id=(SELECT qwiz_id FROM deleted)`, q.QwizID, q.Index)
	return err
}

func (q *Question) UpdateIndex(newIndex int32) (bool, error) {
	switch {
	case newIndex == q.Index:
		return false, nil

	case newIndex > q.Index:
		_, err := DB.Exec("DELETE FROM question WHERE qwiz_id=? AND index=?", q.QwizID, q.Index)
		if err != nil {
			return false, err
		}

		_, err = DB.Exec("UPDATE question SET index=index-1 WHERE index>? AND index<=? AND qwiz_id=?", q.Index, newIndex, q.QwizID)
		if err != nil {
			return false, err
		}

		err = DB.Get(&q.Index, `INSERT INTO question (qwiz_id, index, body, answer1, answer2, answer3, answer4, correct, embed_uuid)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING index`,
			q.QwizID, newIndex, q.Body, q.Answer1, q.Answer2, q.Answer3, q.Answer4, q.Correct, q.EmbedUUID)
		if err != nil {
			return false, err
		}

		return true, nil

	case newIndex < q.Index:
		_, err := DB.Exec("DELETE FROM question WHERE qwiz_id=? AND index=?", q.QwizID, q.Index)
		if err != nil {
			return false, err
		}

		_, err = DB.Exec("UPDATE question SET index=index+1 WHERE index>=? AND index<? AND qwiz_id=?", newIndex, q.Index, q.QwizID)
		if err != nil {
			return false, err
		}

		err = DB.Get(&q.Index, `INSERT INTO question (qwiz_id, index, body, answer1, answer2, answer3, answer4, embed_uuid, correct)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING index`,
			q.QwizID, newIndex, q.Body, q.Answer1, q.Answer2, q.Answer3, q.Answer4, q.EmbedUUID, q.Correct)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, fmt.Errorf("unexpected condition in UpdateIndex")
}

func (q *Question) UpdateBody(newBody string) error {

	err := DB.Get(&q.Body, "UPDATE question SET body=? WHERE qwiz_id=? AND index=? RETURNING body",
		newBody, q.QwizID, q.Index)

	return err
}

func (q *Question) UpdateAnswer(answerNumber uint8, newAnswer *string) (bool, error) {
	switch answerNumber {
	case 1:
		if newAnswer == nil {
			return false, nil
		}
		return true, DB.Get(&q.Answer1, "UPDATE question SET answer1=? WHERE qwiz_id=? AND index=? RETURNING answer1",
			*newAnswer, q.QwizID, q.Index)

	case 2:
		if newAnswer == nil {
			return false, nil
		}
		return true, DB.Get(&q.Answer2, "UPDATE question SET answer2=? WHERE qwiz_id=? AND index=? RETURNING answer2",
			*newAnswer, q.QwizID, q.Index)

	case 3:
		if newAnswer == nil {
			newAnswer = new(string) // Go's sqlx doesn't handle nil pointers well for nullable database fields
		}
		return true, DB.Get(&q.Answer3, "UPDATE question SET answer3=? WHERE qwiz_id=? AND index=? RETURNING answer3",
			*newAnswer, q.QwizID, q.Index)

	case 4:
		if newAnswer == nil {
			newAnswer = new(string) // Go's sqlx doesn't handle nil pointers well for nullable database fields
		}
		return true, DB.Get(&q.Answer4, "UPDATE question SET answer4=? WHERE qwiz_id=? AND index=? RETURNING answer4",
			*newAnswer, q.QwizID, q.Index)

	default:
		return false, errors.New("invalid answer number")
	}
}

func (q *Question) UpdateCorrect(newCorrect int16) error {
	err := DB.Get(&q.Correct, "UPDATE question SET correct=? WHERE qwiz_id=? AND index=? RETURNING correct",
		newCorrect, q.QwizID, q.Index)

	return err
}

func (q *Question) UpdateEmbed(newData *media.NewMediaData) error {
	switch {
	case q.EmbedUUID != nil:
		med, err := media.GetByUUID(q.EmbedUUID)
		if err != nil {
			return err
		}
		return med.Update(newData)

	default:
		med, err := media.FromMediaData(newData)
		if err != nil {
			return err
		}

		_, err = DB.Exec("UPDATE question SET embed_uuid=? WHERE qwiz_id=? AND index=?",
			med.UUID, q.QwizID, q.Index)

		if err != nil {
			return err
		}
		q.EmbedUUID = &med.UUID
	}

	return nil
}
