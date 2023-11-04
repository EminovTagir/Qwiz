package media

import (
	"api/utils"
	"encoding/base64"
	"errors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
	mt = Type(strings.ToLower(string(mt)))
	switch mt {
	case Image:
		return "png"
	case Video:
		return "mp4"
	case Audio:
		return "mp3"
	case Gif:
		return "gif"
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
	if nmd == nil {
		log.Printf("NewMediaData is nil")
		return "", errors.New("newMediaData is nil")
	}
	log.Printf("Getting URI via nmd: %s", nmd)
	log.Printf("nmd.MediaType: %s", nmd.MediaType)
	if nmd.MediaType == Youtube {
		log.Printf("Returning YouTube URI: %s", nmd.Data)
		return nmd.Data, nil
	}

	log.Printf("Encoding base64 data for non-YouTube media...")
	decoded, err := base64.StdEncoding.DecodeString(nmd.Data)

	fileUUID := uuid.New()
	filename := filepath.Join(os.Getenv("MEDIA_DIR"), fileUUID.String()+"."+nmd.MediaType.GetFileExtension())
	log.Printf("File Extension: %s", nmd.MediaType.GetFileExtension())
	filename, err = utils.ExpandTilde(filename)

	if err != nil {
		log.Printf("Error decoding base64 data: %v", err)
		return "", errors.New("error decoding base64 data")
	}

	if err := os.WriteFile(filename, decoded, 0644); err != nil {
		log.Printf("Error writing file to disk: %v", err)
		return "", errors.New(string(IOError))
	}

	log.Printf("File saved successfully: %s", filename)

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
	log.Printf("Getting media via nmd: %s", mediaDatas)
	var uris []string
	var mediaTypes []Type
	for _, data := range mediaDatas {
		uri, err := data.GetURI()
		if err != nil {
			log.Printf("Error getting URI: %v", err)
			return nil, err
		}
		uris = append(uris, uri)
		mediaTypes = append(mediaTypes, data.MediaType)
	}

	log.Printf("URIs: %v", uris)
	log.Printf("Media Types: %v", mediaTypes)

	query := `INSERT INTO media (uri, media_type)
	SELECT * FROM UNNEST($1::VARCHAR[], $2::media_type[])
	RETURNING uuid, uri, media_type`

	var medias []*Media
	err := DB.Select(&medias, query, pq.StringArray(uris), pq.Array(mediaTypes))
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, err
	}

	log.Printf("Media inserted successfully: %v", medias)

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
