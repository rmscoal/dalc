package api

import "net/http"

const (
	JSON_CONTENT_TYPE  string = "application/json"
	PLAIN_CONTENT_TYPE string = "text/plain"
)

const (
	ContentTypeHeader = "Content-Type"
)

func NewErrorMessage(w http.ResponseWriter, err error, status int) {
	w.Header().Set(ContentTypeHeader, PLAIN_CONTENT_TYPE)
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}

func NewOkMessage(w http.ResponseWriter, message string) {
	w.Header().Set(ContentTypeHeader, PLAIN_CONTENT_TYPE)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(message))
}

func NewOk(w http.ResponseWriter, data []byte) {
	w.Header().Set(ContentTypeHeader, JSON_CONTENT_TYPE)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func NewCreated(w http.ResponseWriter, data []byte) {
	w.Header().Set(ContentTypeHeader, JSON_CONTENT_TYPE)
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
