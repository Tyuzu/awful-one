package places

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// 🖼️ Exhibits
func GetExhibit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetExhibit not implemented yet"))
}

func PostExhibit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("PostExhibit not implemented yet"))
}

func PutExhibit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("PutExhibit not implemented yet"))
}

func DeleteExhibit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("DeleteExhibit not implemented yet"))
}

func GetExhibits(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetExhibits not implemented yet"))
}
