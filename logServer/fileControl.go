package logServer

import (
	"fmt"
	"io"
	"os"
	"path"
)

type fileControl string

func newFileControl(dir string) (fileControl, error) {
	// 保存先ディレクトリが存在するか
	fileStat, err := os.Stat(dir)
	if os.IsNotExist(err) || !fileStat.IsDir() {
		return fileControl(""), fmt.Errorf("ファイル制御対象ディレクトリが存在しません %v", dir)
	}
	return fileControl(dir), nil
}

func (ctrl fileControl) makePath(name string) string {
	return path.Join(string(ctrl), name)
}

// 指定した名前でファイルを保存する。すでにファイルが存在している場合はエラー扱いとなる
func (ctrl fileControl) save(name string, src io.Reader) error {
	filePath := ctrl.makePath(name)

	_, err := os.Stat(filePath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("%v はすでに存在するため新規に保存できません", filePath)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	io.Copy(f, src)

	return nil
}

// 指定した名前のファイルを削除する。ファイルが存在しない場合はエラー扱いとなる
func (ctrl fileControl) delete(name string) error {
	filePath := ctrl.makePath(name)

	fileStat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%v は存在しないため削除できません", name)
	}
	if fileStat.IsDir() {
		return fmt.Errorf("%v はディレクトリのため削除できません", name)
	}

	err = os.Remove(filePath)

	return err
}
