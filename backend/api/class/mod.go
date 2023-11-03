package class

import (
	"api/account"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
)

type NewClassData struct {
	TeacherID  int32    `json:"teacher_id"`
	Name       string   `json:"name"`
	StudentIDs *[]int32 `json:"student_ids,omitempty"` // `omitempty` опционально, если вы хотите, чтобы это поле игнорировалось, когда оно пустое
}

type Error string

const (
	SqlxError       Error = "SQL error"
	AccountNotFound Error = "Account with ID %d not found"
	NotATeacher     Error = "Account with ID %d is not a teacher"
	NotAStudent     Error = "Account with ID %d is not a student"
)

func (e Error) Error() string {
	return string(e)
}

var DB *sqlx.DB

type Class struct {
	ID        int32  `db:"id"`
	TeacherID int32  `db:"teacher_id"`
	Name      string `db:"name"`
}

func GetByID(id int32) (*Class, error) {
	class := &Class{}
	err := DB.Get(class, "SELECT id, teacher_id, name FROM class WHERE id = $1", id)
	if err != nil {
		log.Printf("Error retrieving class with ID %d: %v", id, err)
		return nil, err
	}
	return class, nil
}

func GetAllByTeacherID(teacherID int32) ([]Class, error) {
	acct, err := account.GetByID(teacherID)
	if err != nil {
		errMsg := fmt.Sprintf(string(AccountNotFound), teacherID)
		return nil, errors.New(errMsg)
	}
	if acct.AccountType != account.Teacher.String() {
		return nil, Error(fmt.Sprintf(string(NotATeacher), teacherID))
	}

	var classes []Class
	err = DB.Select(&classes, "SELECT * FROM class WHERE teacher_id=$1", teacherID)
	if err != nil {
		log.Printf("SQL error: %v", err) // или fmt.Printf для вывода в консоль
		return nil, fmt.Errorf("sql error: %w", err)
	}
	return classes, nil
}

func GetAllByStudentID(studentID int32) ([]Class, error) {
	acct, err := account.GetByID(studentID)
	if err != nil {
		return nil, Error(fmt.Sprintf(string(AccountNotFound), studentID))
	}
	if acct.AccountType != account.Student.String() {
		return nil, Error(fmt.Sprintf(string(NotAStudent), studentID))
	}

	// Извлекаем все классы, связанные со студентом по его ID
	var classes []Class
	query := `SELECT c.* FROM class c JOIN student s ON c.id = s.class_id WHERE s.student_id = $1`
	err = DB.Select(&classes, query, studentID)
	if err != nil {
		log.Printf("SQL error: %v", err) // или fmt.Printf для вывода в консоль
		return nil, fmt.Errorf("sql error: %w", err)
	}
	return classes, nil
}

func FromClassData(data *NewClassData) (*Class, error) {
	if !account.ExistsID(data.TeacherID) {
		return nil, Error(fmt.Sprintf(string(AccountNotFound), data.TeacherID))
	}
	acct, err := account.GetByID(data.TeacherID)
	if err != nil {
		return nil, err
	}
	if acct.AccountType != account.Teacher.String() {
		return nil, Error(fmt.Sprintf(string(NotATeacher), data.TeacherID))
	}

	// Создаем новый класс
	classObj := &Class{
		TeacherID: data.TeacherID,
		Name:      data.Name,
	}

	var lastInsertId int32
	err = DB.QueryRowx(
		`INSERT INTO class (teacher_id, name) VALUES ($1, $2) RETURNING id`,
		classObj.TeacherID, classObj.Name,
	).Scan(&lastInsertId)

	if err != nil {
		log.Printf("Error inserting new class and getting last insert ID: %v", err)
		return nil, Error(fmt.Sprintf(string(SqlxError)))
	}

	// Использование полученного ID
	classObj.ID = lastInsertId

	return classObj, nil
}

func (c *Class) Delete() error {
	_, err := DB.Exec(`DELETE FROM class WHERE id = $1`, c.ID)
	if err != nil {
		return err
	}
	return nil
}

func (c *Class) GetAllStudents() ([]int32, error) {
	var studentIDs []int32
	err := DB.Select(&studentIDs, `SELECT student_id FROM student WHERE class_id = $1`, c.ID)
	if err != nil {
		return nil, err
	}
	return studentIDs, nil
}

func (c *Class) AddStudents(studentIDs *[]int32) error {
	// Мы будем использовать транзакции, чтобы добавить несколько студентов
	tx, err := DB.Beginx()
	if err != nil {
		return err
	}

	for _, studentID := range *studentIDs {
		_, err := tx.Exec(`INSERT INTO student (student_id, class_id) VALUES ($1, $2)`, studentID, c.ID)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				// Мы можем логировать ошибку отката или обернуть ее вместе с первоначальной ошибкой
				return fmt.Errorf("failed to insert student: %v, failed to rollback: %v", err, rollbackErr)
			}
			return err
		}
	}

	// Завершаем транзакцию
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (c *Class) RemoveStudents(studentIDs *[]int32) error {
	query, args, err := sqlx.In(`DELETE FROM student WHERE class_id = ? AND student_id IN (?)`, c.ID, *studentIDs)
	if err != nil {
		return err
	}
	// Sqlx.In возвращает запрос, который использует вопросительные знаки (?) вместо параметров плейсхолдера.
	// Мы используем DB.Rebind, чтобы преобразовать его в нужный нам формат (например, $1, $2 для PostgreSQL).
	query = DB.Rebind(query)
	_, err = DB.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil
}
