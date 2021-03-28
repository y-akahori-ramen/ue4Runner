package main

import (
	"os"
	"testing"
)

func TestFileControl(t *testing.T) {

	dirName := "dummy"
	err := os.Mkdir(dirName, 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirName)

	f, err := os.Create("dummyFile")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove("dummyFile")
	}()

	fileCtrl, err := newFileControl(dirName)
	if err != nil {
		t.Fatal(err)
	}

	err = fileCtrl.save("sample1.json", f)
	if err != nil {
		t.Fatal(err)
	}

	// すでに存在する名前での保存は失敗する
	err = fileCtrl.save("sample1.json", f)
	if err == nil {
		t.Fatal("すでに存在するファイルへの上書きに成功しています")
	}

	// 存在するファイルの削除は成功する
	err = fileCtrl.delete("sample1.json")
	if err != nil {
		t.Fatal(err)
	}

	// 存在しないファイルの削除は失敗する
	err = fileCtrl.delete("sample1.json")
	if err == nil {
		t.Fatal("存在しないファイルの削除に成功しています")
	}

	// ファイルを削除したので保存に成功する
	err = fileCtrl.save("sample1.json", f)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFileControlNew(t *testing.T) {
	// 存在しないディレクトリを指定してFileControlを作成しようとすると失敗する
	_, err := newFileControl("dummy")
	if err == nil {
		t.Fatal("作成に成功しているのは不正")
	}

	// ファイルを指定して作成しようとすると失敗する
	dummyFile, err := os.Create("dummyFile")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		dummyFile.Close()
		os.Remove("dummyFile")
	}()

	_, err = newFileControl("dummyFile")
	if err == nil {
		t.Fatal("作成に成功しているのは不正")
	}

	// ディレクトリを指定して作成すると成功
	err = os.Mkdir("dummyDir", 0777)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("dummyDir")

	_, err = newFileControl("dummyDir")
	if err != nil {
		t.Fatal(err)
	}
}
