package utils

import (
	"log"
	"net/http"
)

func HttpError(errId string, err error, w http.ResponseWriter, code int) {
	log.Printf("Error at %s: %s\n", errId, err.Error())
	http.Error(w, http.StatusText(code)+"\n"+err.Error(), code)
}
