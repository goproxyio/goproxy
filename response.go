package main

import (
	"fmt"
	"net/http"
	"os"
)

func ReturnServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	msg := fmt.Sprintf("%v", err)
	fmt.Fprintf(os.Stderr, "goproxy: %s\n", msg)
	w.Write([]byte(msg))
}
