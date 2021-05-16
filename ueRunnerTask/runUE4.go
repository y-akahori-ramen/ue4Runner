package ueRunnerTask

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/y-akahori-ramen/ziptool"
)

// getLatestModTime 指定したディレクトリ以下のうち最も更新時間が最新のものを取得する。ディレクトリが存在しない場合はエラーを返す
func getLatestModTime(dir_path string) (time.Time, error) {
	latest_time := time.Time{}

	stat, err := os.Stat(dir_path)
	if os.IsNotExist(err) {
		return latest_time, fmt.Errorf("%vは存在しません", dir_path)
	}
	if !stat.IsDir() {
		return latest_time, fmt.Errorf("%vはディレクトリではありません", dir_path)
	}

	err = filepath.Walk(dir_path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 最も更新時間が新しいものをlatest_timeに入れる
		if latest_time.Before(info.ModTime()) {
			latest_time = info.ModTime()
		}
		return nil
	})

	return latest_time, err
}

// copyAfter srcDirで指定したディレクトリ内のファイルのうちbaseTimeで指定した時間より更新時刻が後のものをdstDirにコピーする
func copyAfter(srcDir string, dstDir string, baseTime time.Time) error {
	stat, err := os.Stat(srcDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("%vは存在しません", srcDir)
	}
	if !stat.IsDir() {
		return fmt.Errorf("%vはディレクトリではありません", srcDir)
	}

	err = filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.ModTime().After(baseTime) {
			dstFilePath := filepath.Join(dstDir, strings.TrimPrefix(path, srcDir))

			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// 出力するファイルの保存先ディレクトリがなければ作成
			if err := os.MkdirAll(filepath.Dir(dstFilePath), 0777); err != nil {
				return err
			}
			err = ioutil.WriteFile(dstFilePath, b, 0777)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// terminateUE 指定したUEExeファイル名で起動されたUEを強制終了させる
func terminateUE(exeNameWithoutExt string) error {
	processes, err := ps.Processes()
	if err != nil {
		return err
	}

	// exe名のプロセスが複数起動しており、すべてを終了させるとUE4を落とすことができる
	for _, p := range processes {
		if strings.Contains(p.Executable(), exeNameWithoutExt) {
			proc, err := os.FindProcess(p.Pid())
			if err != nil {
				return err
			}
			err = proc.Kill()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// runUE4 UE4パッケージを実行し実行時に出力されたSavedディレクトリ内のファイルを指定された場所にzip出力する
// この関数はWindowsのみで動作する
//
// ctx
// 　キャンセル用コンテキスト
// exe
//　起動するUEのexeを指定
// logFileName
// 　UEログのファイル名指定
// outputName
// 　起動した際に出力された物をzipアーカイブしたファイルの出力先
//　 savedディレクトリ内のLogs/Profiling/Screenshotsが対象
// timeOutDuration
// 　フリーズ判定用時間
// 　この時間が経過してもUEログに更新がなければフリーズ扱いとして強制終了する
// additionalArg
// 　UE起動時の追加引数
// 　フリーズ判定する関係でUEログのファイル名はlogFileNameで渡された名前で固定される
// 　additionalArgにUEログファイル名指定が含まれる場合はエラーとなる
func runUE4(ctx context.Context, exe string, logfileName string, outputName string, timeOutDuration time.Duration, additionalArg ...string) error {

	// Windowsのみの対応
	if runtime.GOOS != "windows" {
		return errors.New("Windows以外からは利用できません")
	}

	// 指定のexeは存在しているか
	stat, err := os.Stat(exe)
	if os.IsNotExist(err) {
		return fmt.Errorf("%vは存在しません", exe)
	}
	if stat.IsDir() {
		return fmt.Errorf("%vはディレクトリです。実行可能ファイルを指定してください。", exe)
	}

	// additionalArgにログファイル名を指定するオプションが存在しないか
	for _, arg := range additionalArg {
		if strings.Contains(arg, "-log=") {
			return fmt.Errorf("additionalArgでログファイル名の指定がされています: %v", arg)
		}
	}

	// savedディレクトリのパス作成
	// Windowsの場合はexeの同階層にexeファイル名の名前のディレクトリがあり、その中にsavedディレクトリが作られる。
	exeNameWithoutExt := filepath.Base(exe[:len(exe)-len(filepath.Ext(exe))])
	savedDir := filepath.Join(filepath.Dir(exe), exeNameWithoutExt, "Saved")

	// 起動前にsavedディレクトリ内の更新時刻のうち最も新しい時刻を調べる。
	// この時刻より後の更新時刻になっているものが今回の起動により作られたファイルとなる。
	latestModTimeBeforeUELaunch := time.Time{}
	checkDirNames := [...]string{"Logs", "Profiling", "Screenshots"}
	for _, dirName := range checkDirNames {
		checkDirPath := filepath.Join(savedDir, dirName)
		modTime, err := getLatestModTime(checkDirPath)
		if err == nil {
			if latestModTimeBeforeUELaunch.Before(modTime) {
				latestModTimeBeforeUELaunch = modTime
			}
		}
	}

	// 関数完了通知用
	completeUE, comple := context.WithCancel(context.Background())

	// フリーズ判定の開始
	// 一定時間ファイル更新がないか、contextが完了した場合にUEを強制終了させる。
	logFilePath := filepath.Join(savedDir, "Logs", logfileName)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Printf("ファイル %s を監視します。タイムアウト %v", logFilePath, timeOutDuration)

		prev_mod_time := time.Now()

		ticker := time.NewTicker(timeOutDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stat, err := os.Stat(logFilePath)
				if err != nil {
					log.Printf("ファイル %s の状態取得に失敗しました。UEを強制終了します。", logFilePath)
					terminateUE(exeNameWithoutExt)
					return
				}

				diff := stat.ModTime().Sub(prev_mod_time)
				if diff == 0 {
					log.Printf("ファイル %s が %v 経過しても変化ありませんでした。UEを強制終了します。", logFilePath, timeOutDuration)
					terminateUE(exeNameWithoutExt)
					return
				}

				prev_mod_time = stat.ModTime()
			case <-ctx.Done():
				log.Print("外部からキャンセルが指示されました。UEを強制終了します。")
				terminateUE(exeNameWithoutExt)
				return
			case <-completeUE.Done():
				// UE実行が終了したら監視も終了させる
				return
			}
		}
	}()

	// UE4起動
	args := append([]string{fmt.Sprintf("-log=%v", logfileName)}, additionalArg...)
	exec.Command(exe, args...).Run()
	comple()
	wg.Wait()

	// 今回の実行により更新されたファイルを一時ディレクトリへコピーし、zipにまとめる
	tempDir, err := ioutil.TempDir("", "*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	tempSavedDir := filepath.Join(tempDir, "Saved")
	err = os.Mkdir(tempSavedDir, 0777)
	if err != nil {
		return err
	}

	for _, dirName := range checkDirNames {
		srcDirPath := filepath.Join(savedDir, dirName)
		dstDirPath := filepath.Join(tempSavedDir, dirName)
		copyAfter(srcDirPath, dstDirPath, latestModTimeBeforeUELaunch)
	}

	err = ziptool.Archive(outputName, tempSavedDir)
	if err == nil {
		log.Print("実行結果をアーカイブしました:", outputName)
	}
	return err
}
