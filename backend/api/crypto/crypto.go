package crypto

import (
	"crypto/sha512"
	"encoding/hex"
	"math/rand"
	"unicode"
)

const maxIterations = 127

func EncodePassword(password string) string {
	hasher := sha512.New()
	hasher.Write([]byte(password))
	bytes := hasher.Sum(nil)

	iterations := rand.Intn(maxIterations)
	for i := 0; i < iterations; i++ {
		hasher.Reset()
		hasher.Write(bytes)
		bytes = hasher.Sum(nil)
	}

	return hex.EncodeToString(bytes)
}

func VerifyPassword(password, passwordHash string) bool {
	hasher := sha512.New()
	hasher.Write([]byte(password))
	bytes := hasher.Sum(nil)

	for i := 0; i < maxIterations; i++ {
		hasher.Reset()
		hasher.Write(bytes)
		bytes = hasher.Sum(nil)

		if hex.EncodeToString(bytes) == passwordHash {
			return true
		}
	}
	return false
}

func ValidatePassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasUppercase := false
	hasLowercase := false
	hasDigit := false
	hasSpecial := false

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUppercase = true
		case unicode.IsLower(ch):
			hasLowercase = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case !unicode.IsLetter(ch) && !unicode.IsDigit(ch):
			hasSpecial = true
		}
	}

	return hasUppercase && hasLowercase && hasDigit && hasSpecial
}

func ValidateUsername(username string) bool {
	if len(username) < 3 {
		return false
	}

	for _, ch := range username {
		if !(unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_') {
			return false
		}
	}

	return true
}
