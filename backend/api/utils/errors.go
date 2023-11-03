package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"net/http"
)

const (
	// StatusNotFound Вместо StatusNotFound и StatusInternalServerError используются стандартные коды HTTP.
	StatusNotFound            = http.StatusNotFound
	StatusInternalServerError = http.StatusInternalServerError
)

func LogErr(err error) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s ERROR: %v\n", red("[ERROR]"), err)
}

// InternalErr Функция для обработки внутренних ошибок
func InternalErr(err error) int {
	LogErr(err)
	return StatusInternalServerError
}

// DbErrToStatus Функция аналогичная db_err_to_status из Rust
func DbErrToStatus(err error, status int) int {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return status
	default:
		return InternalErr(err)
	}
}
