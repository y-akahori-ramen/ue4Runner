package ueRunnerTask_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
	"github.com/y-akahori-ramen/ue4Runner/logServer"
	"github.com/y-akahori-ramen/ue4Runner/ueRunnerTask"
)

func TestRunUE4(t *testing.T) {
	exe := filepath.FromSlash("C:/Users/xpk20/Desktop/UESandBox/WindowsNoEditor/UnrealSandBox.exe")

	// exe := filepath.FromSlash("path to unreal package exe")
	timeOutDuration := time.Second * 4
	fileServerAddr := "localhost:8000"
	fileServerURL := fmt.Sprint("http://", fileServerAddr)
	taskServerAddr := "localhost:8080"
	taskServerURL := fmt.Sprint("http://", taskServerAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wgForRunServer sync.WaitGroup

	// ファイルサーバーをたてる
	wgForRunServer.Add(1)
	go func() {
		err := os.MkdirAll("./tmp", 0777)
		if err != nil {
			log.Fatal(err)
		}

		logSrv, err := logServer.NewLogServer("./tmp")
		if err != nil {
			log.Fatal(err)
		}

		server := &http.Server{Addr: fileServerAddr, Handler: logSrv.NewHTTPHandler()}

		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()

		wgForRunServer.Done()
		server.ListenAndServe()
	}()

	// UEを実行するTaskRunnerサーバーをたてる
	wgForRunServer.Add(1)
	go func() {
		taskSrv := gojobcoordinatortest.NewTaskRunnerServer(1)
		go func() {
			taskSrv.RunWithContext(ctx)
		}()

		uploader := ueRunnerTask.NewLogServerUploader(fileServerURL)
		factory, err := ueRunnerTask.NewTaskFactory(exe, timeOutDuration, &uploader)
		if err != nil {
			log.Fatal(err)
		}

		taskSrv.AddFactory(ueRunnerTask.TaskName, factory.NewTask)
		server := &http.Server{Addr: taskServerAddr, Handler: taskSrv.NewHTTPHandler()}
		go func() {
			<-ctx.Done()
			server.Shutdown(context.Background())
		}()

		wgForRunServer.Done()
		server.ListenAndServe()
	}()

	// サーバー起動を待つ
	wgForRunServer.Wait()

	// タスクを開始しタスクIDを得る
	var taskID string
	{
		// TaskRunnerへの開始リクエストを作成
		var postReq *http.Request
		requestParam := ueRunnerTask.TaskParam{LogFileServer: fileServerURL, Args: []string{}}
		mapData, err := gojobcoordinatortest.StructToMap(requestParam)
		if err != nil {
			t.Fatal(err)
		}
		startReqData := gojobcoordinatortest.TaskStartRequest{ProcName: ueRunnerTask.TaskName, Params: &mapData}

		postReq, err = gojobcoordinatortest.NewJSONRequest(http.MethodPost, fmt.Sprint(taskServerURL, "/start"), startReqData)
		if err != nil {
			t.Fatal(err)
		}

		// リクエストを送る
		resp, err := http.DefaultClient.Do(postReq)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatal()
		}

		// 開始リクエストのレスポンスを読み込み、タスクIDを取得
		var startResponse gojobcoordinatortest.JobStartResponse
		err = gojobcoordinatortest.ReadJSONFromResponse(resp, &startResponse)
		if err != nil {
			t.Fatal()
		}
		taskID = startResponse.ID
	}

	// タスクIDの状態を一定間隔で監視
	ticker := time.NewTicker(time.Second * 1)
	for range ticker.C {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprint(taskServerURL, "/status/", taskID), nil)
		if err != nil {
			t.Fatal()
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal()
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatal()
		}
		var statusResponse gojobcoordinatortest.TaskStatusResponse
		err = gojobcoordinatortest.ReadJSONFromResponse(resp, &statusResponse)
		resp.Body.Close()
		if err != nil {
			t.Fatal()
		}

		log.Print("StatusCheck:", statusResponse.Status)
		if statusResponse.Status != gojobcoordinatortest.StatusBusy {
			break
		}
	}

	cancel()
}
