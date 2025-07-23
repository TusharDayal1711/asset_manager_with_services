package utils

import (
	"encoding/json"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

func ParseJSONBody(r *http.Request, dst interface{}) error {
	decoder := jsoniter.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(dst)
	if err != nil {
		return err
	}
	return nil
}

type Role string

const (
	adminRole          Role = "admin"
	employeeMangerRole Role = "employee_manager"
	assetManagerRole   Role = "asset_manager"
)

func RespondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to serialize JSON response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}
