package utils

import (
	"log"
	"net/http"
)

func HttpError(errId string, err error, w http.ResponseWriter, code int) {
	log.Printf("Error at %s: %s", errId, err.Error())
	http.Error(w, http.StatusText(code), code)
}
