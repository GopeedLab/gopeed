package util

import (
	"encoding/json"
	"github.com/monkeyWie/gopeed-core/pkg/rest/model"
	"net/http"
)

func ReadJson(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		WriteJson(w, model.NewError(500, err.Error()))
		return false
	}
	return true
}

func WriteJson(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.NewOkWithData(v))
}
