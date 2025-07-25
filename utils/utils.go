package utils

import (
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"go.uber.org/zap"
	"net/http"
	"time"
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

func RespondJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	currentTimeBefore := time.Now()
	fmt.Print("json time ::", currentTimeBefore)
	response, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Failed to serialize JSON response", http.StatusInternalServerError)
		return
	}
	currentTimeAfter := time.Now()
	fmt.Print("json time ::", currentTimeAfter)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}

// zap logger
var Logger *zap.Logger

func InitLogger() {
	var err error
	Logger, err = zap.NewDevelopment()
	if err != nil {
		panic("Failed to initialize zap logger: " + err.Error())
	}
	zap.ReplaceGlobals(Logger)
}

func SyncLogger() {
	if Logger != nil {
		Logger.Sync()
	}
}
