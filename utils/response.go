package utils

import (
	"encoding/json"
	"net/http"
)

// Response represents a generic response
type Response struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Error   string      `json:"error"`
}

// RespondWithError sends an error response
func RespondWithError(w http.ResponseWriter, code int, msg string) {
	RespondWithJSON(w, code, Response{
		Status: "error",
		Code:   code,
		Error:  msg,
	})
}

// RespondWithOk response
func RespondWithOk(w http.ResponseWriter, msg string) {
	RespondWithJSON(w, http.StatusOK, Response{
		Status:  "success",
		Code:    http.StatusOK,
		Message: msg,
	})
}

// RespondWithJSON ... This
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	return
}
