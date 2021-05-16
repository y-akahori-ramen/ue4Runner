package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/y-akahori-ramen/gojobcoordinatortest"
	"github.com/y-akahori-ramen/ue4Runner/ueRunnerTask"
)

type options struct {
	Addr               string `long:"addr" description:"実行サーバーアドレス" default:"localhost:8080"`
	UEExe              string `long:"ueExePath" description:"起動するUEのExeパス" required:"true"`
	FileServerURL      string `long:"fileServer" description:"実行結果のアップロード先サーバー" required:"true"`
	FileServerUserName string `long:"user" description:"アップロード先サーバーのユーザー名" default:""`
	FileServerPassword string `long:"password" description:"アップロード先サーバーのパスワード" default:""`
	TimeOutSec         int    `long:"timeOutSec" description:"一定時間ログ更新がなければフリーズとして扱う時間" default:"60"`
}

func main() {
	var opt options

	_, err := flags.Parse(&opt)
	if err != nil {
		log.Fatal(err)
	}

	server := gojobcoordinatortest.NewTaskRunnerServer(1)

	// UE起動タスクのファクトリを登録
	uploader := ueRunnerTask.NewLogServerUploaderWithBasicAuth(opt.FileServerURL, opt.FileServerUserName, opt.FileServerPassword)
	timeOut := time.Second * time.Duration(opt.TimeOutSec)
	factory, err := ueRunnerTask.NewTaskFactory(opt.UEExe, timeOut, &uploader)
	if err != nil {
		log.Fatal(err)
	}
	server.AddFactory(ueRunnerTask.TaskName, factory.NewTask)

	router := server.NewHTTPHandler()
	go func() {
		server.Run()
	}()

	fmt.Printf("UE4実行サーバー起動します addr:%v\n", opt.Addr)

	log.Fatal(http.ListenAndServe(opt.Addr, router))
}
