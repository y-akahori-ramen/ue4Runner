package ueRunnerTask

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

// Uploader zipファイルのアップローダーインターフェイス
type Uploader interface {
	// Upload ファイルをアップロードし、アップロードしたファイルをダウンロードするURLを返す
	// path アップロードするファイルパス
	Upload(path string) (string, error)
}

// LogServerUploader logServerへアップロードするアップローダー
type LogServerUploader struct {
	url      string
	user     string
	password string
}

// NewLogServerUploaderWithBasicAuth Basic認証付きのlogServer用アップローダー
func NewLogServerUploaderWithBasicAuth(url string, username string, password string) LogServerUploader {
	return LogServerUploader{url: url, user: username, password: password}
}

// NewLogServerUploader logServer用アップローダー
func NewLogServerUploader(url string) LogServerUploader {
	return NewLogServerUploaderWithBasicAuth(url, "", "")
}

func (uploader *LogServerUploader) Upload(path string) (string, error) {

	// ファイルサーバーへzipをアップロードする
	fileID := filepath.Base(path)
	postUrl := fmt.Sprintf("%s/upload/%s", uploader.url, fileID)

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ファイル読み込みに失敗しました: %v %v: ", file, err)
	}

	req, err := http.NewRequest(http.MethodPost, postUrl, bytes.NewBuffer(file))
	if err != nil {
		return "", fmt.Errorf("HTTPリクエストの作成に失敗しました: %v", err)
	}

	if uploader.user != "" && uploader.password != "" {
		req.SetBasicAuth(uploader.user, uploader.password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ファイルアップロードに失敗しました: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ファイルアップロードのレスポンスが不正です: %v", http.StatusText(resp.StatusCode))
	}

	return fmt.Sprintf("%s/files/%s", uploader.url, fileID), nil
}
