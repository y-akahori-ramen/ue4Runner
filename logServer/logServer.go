package logServer

import (
	"net/http"

	"github.com/gorilla/mux"
)

// LogServer UEのログを保存するログファイルサーバー
type LogServer struct {
	fileCtrl fileControl
	dirName  string
}

// NewLogServer 指定したディレクトリを保存先として使用するログファイルサーバーの作成
func NewLogServer(dir string) (*LogServer, error) {
	server := &LogServer{dirName: dir}

	var err error
	server.fileCtrl, err = newFileControl(dir)
	if err != nil {
		return nil, err
	}

	return server, nil
}

// NewHTTPHandler ログファイルサーバーのHTTPHandlerを作成する
func (server *LogServer) NewHTTPHandler() http.Handler {
	r := mux.NewRouter()
	r.PathPrefix("/files/").Handler(http.StripPrefix("/files/", http.FileServer(http.Dir(server.dirName))))
	r.HandleFunc("/upload/{contentID}", server.uploaderHandler).Methods("POST")
	r.HandleFunc("/delete/{contentID}", server.deleteHandler).Methods("POST")
	return r
}

func (server *LogServer) uploaderHandler(w http.ResponseWriter, r *http.Request) {

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

func (server *LogServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	err := server.fileCtrl.delete(vars["contentID"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
