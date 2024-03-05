package util

import (
	"io"
	"os"
)

func CopyFile(srcPath, dstPath string) error {
	fin, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	return err
}
