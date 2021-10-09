package api

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/golang/gddo/httputil/header"
)

//TODO: vicky find how to keep the templates/* outside of the api. ie need to figure how to do relative path.
// check: https://stackoverflow.com/questions/66285635/how-do-you-use-go-1-16-embed-features-in-subfolders-packages

//go:embed templates/*
var dockerTempl embed.FS

type malformedRequest struct {
	status int
	msg    string
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintf("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go/22892986#22892986
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// TODO: To remove the duplication here. We don't need separate function for each file
func createDockerComposeFile(absolutepath string, s service) error {
	outputPath := filepath.Join(absolutepath, "docker-compose.yml")
	// Create the file:
	f, err := os.Create(outputPath)
	if err != nil {
		panic(err)
	}

	defer f.Close() // don't forget to close the file when finished.
	templ, err := template.ParseFS(dockerTempl, "templates/docker-compose-template.yml")
	if err != nil {
		return fmt.Errorf("ERROR: parsing template file %v", err)
	}
	// TODO: not sure is there a better way to pass data to template
	// A lot of this data is redundant. Already available in Service struct
	data := struct {
		UserID           string
		Architecture     string
		Type             string
		Port             int
		PostgresUsername string
		PostgresPassword string
	}{
		s.UserID,
		s.Architecture,
		s.Db.Type,
		s.Db.Port,
		s.Db.Username,
		s.Db.Password,
	}
	err = templ.Execute(f, data)
	if err != nil {
		return fmt.Errorf("ERROR: executing template file %v", err)
	}
	return nil
}
