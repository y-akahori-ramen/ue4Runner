package ueRunnerTask

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/y-akahori-ramen/gojobcoordinatortest"
)

// TaskFactory TaskUE4Runnerのファクトリ
// gojobcoordinatortest.TaskRunnerServerのファクトリ登録に使用する
type TaskFactory struct {
	exePath  string
	timeOut  time.Duration
	uploader Uploader
}

// NewTaskFactory TaskUE4RunnerFactoryを作成する
// exePath 実行するUEパッケージのexe
// timeOut タイムアウト設定。一定時間以上ログファイルに更新がなければフリーズとして扱う
// uploader 実行結果ファイルのアップローダー
func NewTaskFactory(exePath string, timeOut time.Duration, uploader Uploader) (TaskFactory, error) {
	if runtime.GOOS != "windows" {
		return TaskFactory{}, errors.New("Windows専用タスクです")
	}

	_, err := os.Stat(exePath)
	if err != nil {
		return TaskFactory{}, fmt.Errorf("ファイルが存在しません:%s", exePath)
	}

	return TaskFactory{exePath: exePath, timeOut: timeOut, uploader: uploader}, nil
}

// NewTask gojobcoordinatortestのタスク開始リクエストを受け取り、タスクを返す
func (factory *TaskFactory) NewTask(req *gojobcoordinatortest.TaskStartRequest) (gojobcoordinatortest.Task, error) {
	var runnerParam TaskParam
	err := gojobcoordinatortest.MapToStruct(*req.Params, &runnerParam)
	if err != nil {
		return nil, err
	}

	return &Task{exePath: factory.exePath, param: runnerParam, timeOut: factory.timeOut, uploader: factory.uploader}, nil
}
