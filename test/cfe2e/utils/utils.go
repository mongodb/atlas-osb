package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sethvargo/go-password/password"
)

func GenID() string {
	id, _ := password.Generate(10, 3, 0, true, true)
	return id
}

func SaveToFile(path string, data []byte) error {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
