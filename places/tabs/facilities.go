package places

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// 🌳 Facilities
func GetFacility(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetFacility not implemented yet"))
}

func PostFacility(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("PostFacility not implemented yet"))
}

func PutFacility(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("PutFacility not implemented yet"))
}

func DeleteFacility(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("DeleteFacility not implemented yet"))
}

func GetFacilities(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("GetFacilities not implemented yet"))
}
