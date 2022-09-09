package util

import (
	"encoding/json"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"net/http"
)

func ReadJson(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteJson(w, 500, model.NewResultWithMsg(err.Error()))
		return false
	}
	return true
}

func WriteJsonOk(w http.ResponseWriter, v any) {
	WriteJson(w, 200, v)
}

func WriteJson(w http.ResponseWriter, code int, v any) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
