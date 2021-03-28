package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

type logServer struct {
	fileCtrl fileControl
	dirName  string
}

func newLogServer(dir string) (*logServer, error) {
	server := &logServer{dirName: dir}

	var err error
	server.fileCtrl, err = newFileControl(dir)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (server *logServer) uploaderHandler(w http.ResponseWriter, r *http.Request) {

	if r.ContentLength <= 0 {
		http.Error(w, "Body is None", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	vars := mux.Vars(r)
	err := server.fileCtrl.save(vars["contentID"], r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (server *logServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := server.fileCtrl.delete(vars["contentID"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (server *logServer) newHTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(server.dirName))))
	r.HandleFunc("/upload/{contentID}", server.uploaderHandler).Methods("POST")
	r.HandleFunc("/delete/{contentID}", server.deleteHandler).Methods("POST")
	return r
}
