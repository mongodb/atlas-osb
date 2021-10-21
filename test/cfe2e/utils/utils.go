package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
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

func GetFieldFromFile(path, name string) (string, error) {
	data, err := ioutil.ReadFile(path)
	fmt.Print(string(data))
	if err != nil {
		return "", err
	}
	match := fmt.Sprintf("%s: \"(.*)\"", name)
	field := regexp.MustCompile(match).FindSubmatch(data)
	if len(field) < 2 {
		return "", errors.Errorf("can not find field")
	}
	return string(field[1]), err
}
