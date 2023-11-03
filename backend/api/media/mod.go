package media

import (
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Type string

const (
	Image   Type = "image"
	Video   Type = "video"
	Audio   Type = "audio"
	Youtube Type = "youtube"
	Gif     Type = "gif"
)

func (mt Type) GetFileExtension() string {
	switch mt {
	case Image:
		return "png"
	case Video:
		return "mp4"
	case Audio:
		return "mp3"
	case Gif:
		return "gif"
	case Youtube:
		return ""
	default:
		return ""
	}
}

type Error string

const (
	SqlxError    Error = "SqlxError"
	Base64Error  Error = "Base64Error"
	IOError      Error = "IOError"
	UnknownError Error = "UnknownError"
)

func (e Error) Error() string {
	return string(e)
}

type Media struct {
	UUID      uuid.UUID `db:"uuid"`
	URI       string    `db:"uri"`
	MediaType string    `db:"media_type"`
}

type PgTypeInfo struct {
	Name string
}

func NewPgTypeInfo(name string) PgTypeInfo {
	return PgTypeInfo{Name: name}
}

type HasArrayType interface {
	ArrayTypeInfo() PgTypeInfo
}

func (mt Type) ArrayTypeInfo() PgTypeInfo {
	return NewPgTypeInfo("_media_type")
}

type NewMediaData struct {
	Data      string `json:"data"`
	MediaType Type   `json:"media_type"`
}

func (nmd *NewMediaData) GetURI() (string, error) {
	if nmd.MediaType == Youtube {
		return nmd.Data, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(nmd.Data)
	if err != nil {
		return "", errors.New(string(Base64Error))
	}

	fileUUID := uuid.New()
	filename := filepath.Join(os.Getenv("MEDIA_DIR"), fileUUID.String()+"."+nmd.MediaType.GetFileExtension())

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		return "", errors.New(string(IOError))
	}

	return filename, nil
}

func GetByUUID(uuidValue *uuid.UUID) (*Media, error) {
	var media Media
	query := `SELECT uuid, uri, media_type FROM media WHERE uuid=$1`
	err := DB.Get(&media, query, uuidValue)
	if err != nil {
		return nil, err
	}
	return &media, nil
}

func (mt Type) ToString() string {
	switch mt {
	case Image:
		return "image"
	case Video:
		return "video"
	case Audio:
		return "audio"
	case Gif:
		return "gif"
	// Добавьте остальные необходимые case
	default:
		return "unknown"
	}
}

func FromMediaData(data *NewMediaData) (*Media, error) {
	if data == nil {
		return nil, errors.New("provided media data is nil")
	}
	uri, err := data.GetURI()
	if err != nil {
		return nil, err
	}
	log.Printf("Media type before: %v", data.MediaType)
	mediaTypeString := strings.ToLower(string(data.MediaType))
	log.Printf("Media type after: %v", mediaTypeString)
	query := `INSERT INTO media (uri, media_type) VALUES ($1, $2) RETURNING uuid, uri, media_type`
	var media Media
	err = DB.QueryRowx(query, uri, mediaTypeString).StructScan(&media)
	if err != nil {
		log.Printf("Error scanning media data into struct: %v", err)
		return nil, err
	}

	return &media, nil
}

func UploadMultiple(datas []*NewMediaData) ([]string, error) {
	var uris []string
	uriChan := make(chan string, len(datas))
	errChan := make(chan error, len(datas))

	for _, data := range datas {
		go func(d *NewMediaData) {
			uri, err := d.GetURI()
			if err != nil {
				errChan <- err
				return
			}
			uriChan <- uri
		}(data)
	}

	for range datas {
		select {
		case uri := <-uriChan:
			uris = append(uris, uri)
		case err := <-errChan:
			// You can decide on how to handle multiple errors.
			// This just returns the first error encountered.
			return nil, err
		}
	}

	if len(uris) != len(datas) {
		return nil, errors.New("failed to get all URIs")
	}

	return uris, nil
}

func FromMediaDatas(mediaDatas []*NewMediaData) ([]*Media, error) {
	var uris []string
	var mediaTypes []Type
	for _, data := range mediaDatas {
		uri, err := data.GetURI()
		if err != nil {
			return nil, err
		}
		uris = append(uris, uri)
		mediaTypes = append(mediaTypes, data.MediaType)
	}

	query := `INSERT INTO media (uri, media_type)
		SELECT * FROM UNNEST($1::VARCHAR[], $2::media_type[])
		RETURNING uuid, uri, media_type`

	var medias []*Media
	err := DB.Select(&medias, query, uris, mediaTypes)
	if err != nil {
		return nil, err
	}

	return medias, nil
}

func (m *Media) Update(newData *NewMediaData) error {
	newUri, err := newData.GetURI()
	if err != nil {
		return err
	}

	query := `UPDATE media SET uri=$1, media_type=$2 WHERE uuid=$3`
	_, err = DB.Exec(query, newUri, newData.MediaType, m.UUID)
	if err != nil {
		return err
	}

	m.URI = newUri
	m.MediaType = newData.MediaType.ToString()

	return nil
}

var DB *sqlx.DB