package assignment

import (
	"api/optbool"
	"github.com/jmoiron/sqlx"
	"time"
)

type Assignment struct {
	ID        int             `db:"id"`
	QwizID    int             `db:"qwiz_id"`
	ClassID   int             `db:"class_id"`
	OpenTime  *time.Time      `db:"open_time"`
	CloseTime *time.Time      `db:"close_time"`
	Completed optbool.OptBool `db:"completed"`
}

func GetByID(id int) (*Assignment, error) {
	const query = `
		SELECT *, 
		EXISTS(SELECT * FROM completed_assignment WHERE assignment_id=id AND student_id=$1) AS completed 
		FROM assignment WHERE id=$1
	`
	a := &Assignment{}
	err := DB.Get(a, query, id)
	return a, err
}

func GetAllByStudentID(studentID int) ([]*Assignment, error) {
	const query = `
		SELECT *, 
		EXISTS(SELECT * FROM completed_assignment WHERE assignment_id=id AND student_id=$1) AS completed 
		FROM assignment 
		WHERE class_id IN (SELECT class_id FROM student WHERE student_id=$1)
	`
	var assignments []*Assignment
	err := DB.Select(&assignments, query, studentID)
	return assignments, err
}

func (a *Assignment) CompleteByStudentID(studentID int) (bool, error) {
	now := time.Now()

	if a.CloseTime != nil && a.CloseTime.Before(now) {
		return false, nil
	}
	if a.OpenTime != nil && a.OpenTime.After(now) {
		return false, nil
	}

	const query = `
		INSERT INTO completed_assignment (assignment_id, student_id) VALUES ($1, $2)
		ON CONFLICT (assignment_id, student_id) DO NOTHING
	`
	_, err := DB.Exec(query, a.ID, studentID)
	if err != nil {
		return false, err
	}

	a.Completed.Value = true
	return true, nil
}

// DB - это ваша глобальная или контекстная переменная для соединения с базой данных
var DB *sqlx.DB
