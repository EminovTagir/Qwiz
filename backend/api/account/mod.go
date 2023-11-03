package account

import (
	"api/crypto"
	"api/media"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"log"
	"strings"
	"sync"
)

type Type string

const (
	Student Type = "student"
	Parent  Type = "parent"
	Teacher Type = "teacher"
)

type Error struct {
	ErrType string
	Err     error
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.ErrType, e.Err)
}

type NewAccountError struct {
	BaseErr Error
	Message string
}

func (e NewAccountError) Error() string {
	if e.BaseErr.Err != nil {
		return e.BaseErr.Error()
	}
	return e.Message
}

// Глобальные данные для хранения ID и имен пользователей.
var (
	IDS          = &sync.Mutex{}
	USERNAMES    = &sync.Mutex{}
	idList       []int32
	usernameList []string
)

type Account struct {
	ID                 int32      `db:"id"`
	Username           string     `db:"username"`
	PasswordHash       string     `db:"password_hash"`
	ProfilePictureUUID *uuid.UUID `db:"profile_picture_uuid"` // sql.NullString
	AccountType        string     `db:"account_type"`
}

// DB Глобальная переменная для подключения к БД (аналог POOL в Rust).
var DB *sqlx.DB

func LoadCache() error {
	var (
		dbIDs       []int32
		dbUsernames []string
	)

	rows, err := DB.Query("SELECT id, username FROM account")
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	for rows.Next() {
		var id int32
		var username string
		if err := rows.Scan(&id, &username); err != nil {
			return err
		}
		dbIDs = append(dbIDs, id)
		dbUsernames = append(dbUsernames, username)
	}

	IDS.Lock()
	USERNAMES.Lock()
	defer IDS.Unlock()
	defer USERNAMES.Unlock()

	idList = dbIDs
	usernameList = dbUsernames

	return nil
}

func CacheID(id int32) {
	IDS.Lock()
	defer IDS.Unlock()
	idList = append(idList, id)
}

func UnCacheID(id int32) {
	IDS.Lock()
	defer IDS.Unlock()
	for i, cachedID := range idList {
		if cachedID == id {
			idList = append(idList[:i], idList[i+1:]...)
			break
		}
	}
}

func CacheUsername(username string) {
	USERNAMES.Lock()
	defer USERNAMES.Unlock()
	usernameList = append(usernameList, username)
}

func UnCacheUsername(username string) {
	USERNAMES.Lock()
	defer USERNAMES.Unlock()
	for i, cachedUsername := range usernameList {
		if cachedUsername == username {
			usernameList = append(usernameList[:i], usernameList[i+1:]...)
			break
		}
	}
}

func ExistsID(id int32) bool {
	var exists bool
	err := DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM account WHERE id = $1)", id)
	return err == nil && exists
}

func (t Type) String() string {
	switch t {
	case Student:
		return "student"
	case Parent:
		return "parent"
	case Teacher:
		return "teacher"
	default:
		return "unknown"
	}
}

func ExistsUsername(username string) bool {
	USERNAMES.Lock()
	defer USERNAMES.Unlock()

	for _, existingUsername := range usernameList {
		if existingUsername == username {
			return true
		}
	}

	return false
}

func GetByID(id int32) (*Account, error) {
	var account Account
	err := DB.Get(&account, "SELECT * FROM account WHERE id=$1", id)
	if err != nil {
		// Если аккаунт не найден, возвращаем соответствующую ошибку
		return nil, err
	}

	return &account, nil
}

func GetByUsername(username string) (*Account, error) {
	var account Account
	err := DB.Get(&account, "SELECT * FROM account WHERE username=$1", username)
	return &account, err
}

func New(username, password string, accountType Type, profilePicture *media.NewMediaData) (*Account, error) {
	if !crypto.ValidateUsername(username) {
		return nil, errors.New("invalid username")
	}
	if !crypto.ValidatePassword(password) {
		return nil, errors.New("invalid password")
	}

	if ExistsUsername(username) {
		return nil, errors.New("username taken")
	}

	passwordHash := crypto.EncodePassword(password)

	var profilePictureUUID *uuid.UUID
	if profilePicture != nil {
		mediaData, err := media.FromMediaData(profilePicture)
		if err != nil {
			log.Printf("Ошибка при создании mediaData: %v\n", err)
			return nil, err
		}
		log.Printf("MediaData создана с UUID: %v\n", mediaData.UUID)
		profilePictureUUID = &mediaData.UUID
	}

	account := &Account{
		Username:           username,
		PasswordHash:       passwordHash,
		AccountType:        accountType.String(),
		ProfilePictureUUID: profilePictureUUID,
	}
	_, err := DB.NamedExec("INSERT INTO account (username, password_hash, account_type, profile_picture_uuid) VALUES (:username, :password_hash, :account_type, :profile_picture_uuid)", account)
	if err != nil {
		return nil, err
	}

	CacheID(account.ID)
	CacheUsername(account.Username)

	return account, nil
}

func (a *Account) UpdatePassword(newPassword string) (bool, error) {
	log.Printf("Attempting to validate password: '%s'", newPassword)
	if !crypto.ValidatePassword(newPassword) {
		log.Printf("Validation failed for password: '%s'", newPassword)
		return false, nil
	}

	passwordHash := crypto.EncodePassword(newPassword)
	row := DB.QueryRow("UPDATE account SET password_hash=$1 WHERE id=$2 RETURNING password_hash", passwordHash, a.ID)
	err := row.Scan(&a.PasswordHash)
	return err == nil, err
}

func (a *Account) UpdateAccountType(newAccountType Type) error {
	lowerCaseType := strings.ToLower(string(newAccountType))
	row := DB.QueryRow("UPDATE account SET account_type=$1 WHERE id=$2 RETURNING account_type", lowerCaseType, a.ID)
	err := row.Scan(&a.AccountType)
	if err != nil {
		log.Printf("Ошибка при обновлении типа аккаунта: %v", err)
	}
	return err
}

func (a *Account) UpdateProfilePicture(newProfilePicture *media.NewMediaData) error {
	if a.ProfilePictureUUID != nil && *a.ProfilePictureUUID != uuid.Nil {
		uuidValue, err := uuid.Parse(a.ProfilePictureUUID.String())
		if err != nil {
			return err
		}
		mediaItem, err := media.GetByUUID(&uuidValue)
		if err != nil {
			return err
		}
		if mediaItem == nil {
			return errors.New("mediaItem is nil after GetByUUID")
		}
		return mediaItem.Update(newProfilePicture)
	}

	log.Printf("newProfilePicture: %v", newProfilePicture)
	mediaItem, err := media.FromMediaData(newProfilePicture)
	if err != nil {
		return err
	}
	if mediaItem == nil {
		return errors.New("mediaItem is nil after GetByUUID")
	}

	_, err = DB.Exec("UPDATE account SET profile_picture_uuid=$1 WHERE id=$2", mediaItem.UUID, a.ID)
	if err != nil {
		return err
	}

	a.ProfilePictureUUID = &mediaItem.UUID
	return nil
}

func (a *Account) Delete() error {
	_, err := DB.Exec("DELETE FROM account WHERE id=$1", a.ID)
	if err != nil {
		return err
	}

	UnCacheID(a.ID)
	UnCacheUsername(a.Username)

	return nil
}

func (a *Account) VerifyPassword(password string) (bool, error) {
	isValid := crypto.VerifyPassword(password, a.PasswordHash)

	if isValid {
		_, updateErr := a.UpdatePassword(password)
		return updateErr == nil, updateErr
	}

	return false, nil
}
