package util

import (
	"encoding/json"
	"github.com/monkeyWie/gopeed/pkg/rest/model"
	"net/http"
)

func ReadJson(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteJson(w, http.StatusInternalServerError, model.NewResultWithMsg(err.Error()))
		return false
	}
	return true
}

func WriteJsonOk(w http.ResponseWriter, v any) {
	WriteJson(w, http.StatusOK, v)
}

func WriteJson(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
