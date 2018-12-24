package proxy

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var errLogger = log.New(os.Stderr, "", log.LstdFlags)

func ReturnServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(500)
	msg := fmt.Sprintf("%v", err)
	errLogger.Printf("goproxy: %s\n", msg)
	_, _ = w.Write([]byte(msg))
}

func ReturnBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(400)
	msg := fmt.Sprintf("%v", err)
	errLogger.Printf("goproxy: %s\n", msg)
	_, _ = w.Write([]byte(msg))
}
