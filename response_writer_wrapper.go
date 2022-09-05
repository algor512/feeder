package main

import "net/http"

type ResponseWriterWrapper struct {
	http.ResponseWriter
	Status int
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{w, http.StatusOK}
}

func (ww *ResponseWriterWrapper) WriteHeader(code int) {
	ww.Status = code
	ww.ResponseWriter.WriteHeader(code)
}
