package main

import (
	"fmt"
	"net/http"
)

func ReturnServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	msg := fmt.Sprintf("%v", err)
	w.Write([]byte(msg))
}
