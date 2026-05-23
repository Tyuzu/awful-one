package places

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func notImplemented(w http.ResponseWriter, methodName string) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(methodName + " not implemented yet"))
}

// ❓ Fallback
func GetPlaceDetailsFallback(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetPlaceDetailsFallback not implemented yet"))
}

func GetDetailsFallback(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetDetailsFallback not implemented yet"))
}
