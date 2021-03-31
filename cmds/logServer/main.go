package main

import (
	"log"
	"net/http"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/y-akahori-ramen/ue4Runner/logServer"
)

type options struct {
	Addr     string `short:"a" long:"addr" description:"ログファイルサーバーのアドレス" default:"localhost:8080"`
	Dir      string `short:"d" long:"dir" description:"ログファイルサーバーのデータ保存先ディレクトリ" required:"true"`
	User     string `short:"u" long:"user" description:"ログファイルサーバーのBasic認証のユーザー名" required:"true"`
	Password string `short:"p" long:"password" description:"ログファイルサーバーのBasic認証のパスワード" required:"true"`
}

type basicAuthHandler struct {
	nextHandler http.Handler
	user        string
	password    string
}

func (b *basicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, pw, ok := r.BasicAuth()
	if !ok || user != b.user || pw != b.password {
		w.Header().Add("WWW-Authenticate", `Basic`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	b.nextHandler.ServeHTTP(w, r)
}

func main() {
	var opt options

	_, err := flags.Parse(&opt)
	if err != nil {
		log.Fatal(err)
	}

	server, err := logServer.NewLogServer(opt.Dir)
	if err != nil {
		log.Fatal(err)
	}

	dirPathAbs, err := filepath.Abs(opt.Dir)
	log.Printf("サーバー起動します\n対象ディレクトリ:%v\nAddr:%v/files/", dirPathAbs, opt.Addr)

	handler := basicAuthHandler{nextHandler: server.NewHTTPHandler(), user: opt.User, password: opt.Password}

	err = http.ListenAndServe(opt.Addr, &handler)
	if err != nil {
		log.Fatal(err)
	}
}
