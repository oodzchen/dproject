package utils

import (
	"fmt"
	"net/http"
)

func HttpError(msg string, err error, w http.ResponseWriter, code int) {
	fmt.Printf("%+v\n", err)
	http.Error(w, http.StatusText(code)+"\n"+msg, code)
}
