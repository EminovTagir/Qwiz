package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

// AllTrue checks if all elements in a slice of bools are true.
func AllTrue(results []bool) bool {
	for _, result := range results {
		if !result {
			return false
		}
	}
	return true
}

// CountCorrect counts the number of true values in a slice of bools.
func CountCorrect(results []bool) int {
	count := 0
	for _, result := range results {
		if result {
			count++
		}
	}
	return count
}

func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				LogErr(e.Err)
				c.JSON(int(e.Type), gin.H{
					"error": e.Error(),
				})
			}
			c.Abort()
		}
	}
}

func LimitRequestBody(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

func ExpandTilde(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return strings.Replace(path, "~", home, 1), nil
	}
	return path, nil
}
