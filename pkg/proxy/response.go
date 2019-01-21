package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

var errLogger = log.New(os.Stderr, "", log.LstdFlags)

func ReturnInternalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	msg := fmt.Sprintf("%v", err)
	errLogger.Printf("goproxy: %s\n", msg)
	_, _ = w.Write([]byte(msg))
}

func ReturnBadRequest(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	msg := fmt.Sprintf("%v", err)
	errLogger.Printf("goproxy: %s\n", msg)
	_, _ = w.Write([]byte(msg))
}

func ReturnSuccess(w http.ResponseWriter, data []byte) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func ReturnJsonData(w http.ResponseWriter, data interface{}) {
	js, err := json.Marshal(data)
	if err != nil {
		ReturnInternalServerError(w, err)
	} else {
		ReturnSuccess(w, js)
	}
}
