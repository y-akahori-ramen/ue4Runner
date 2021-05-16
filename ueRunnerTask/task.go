package ueRunnerTask

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

const (
	TaskName = "UE4Runner"
)

// TaskParam タスク設定
// gojobcoordinatortest.TaskStartRequestのParamsに指定する設定
type TaskParam struct {
	LogFileServer string
	Args          []string
}

// TaskResult タスク成功時の戻り値
// タスクが成功した場合に gojobcoordinatortest.TaskStatusResponseのResultValuesに指定される
type TaskResult struct {
	ZipURL string
}

// Task UE4を実行しSaved以下に出力されたファイルをzipにまとめ指定のファイルサーバーにアップロードする
// ファイルサーバーはこのリポジトリ内の logServer\logServer.go で立てたサーバーを指定する
type Task struct {
	exePath  string
	timeOut  time.Duration
	param    TaskParam
	uploader Uploader
}

// Run タスク実行
func (task *Task) Run(ctx context.Context, taskID string, done chan<- *gojobcoordinatortest.TaskResult) {
	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: false}
		return
	}
	defer os.RemoveAll(tempDir)

	// 実行結果のzipの保存先
	zipName := fmt.Sprint(taskID, ".zip")
	zipPath := filepath.Join(tempDir, zipName)

	logger := log.New(log.Default().Writer(), fmt.Sprintf("[%s]", taskID), log.Default().Flags())
	logger.Print("UEを起動します:", task.exePath, " Args:", task.param.Args)
	err = runUE4(ctx, task.exePath, "log.txt", zipPath, task.timeOut, task.param.Args...)
	if err != nil {
		logger.Print("UE実行でエラーが発生しました")
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: false}
		return
	}

	// ファイルサーバーへzipをアップロードする
	logger.Printf("出力されたzipをアップロードします file:%s", zipPath)
	downloadURL, err := task.uploader.Upload(zipPath)

	if err != nil {
		logger.Printf("zipアップロードに失敗しました:%v", err)
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: false}
		return
	}

	// アップロードしたzipのダウンロードURLを結果として返す
	resultParam := TaskResult{ZipURL: downloadURL}
	mapData, err := gojobcoordinatortest.StructToMap(resultParam)
	if err != nil {
		logger.Print("パラメータ生成に失敗しました:", err)
		done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: false}
		return
	}
	done <- &gojobcoordinatortest.TaskResult{ID: taskID, Success: true, ResultValues: &mapData}
}
