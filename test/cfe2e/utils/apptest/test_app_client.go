package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type App struct {
	url string
}

func NewTestAppClient(url string) *App {
	return &App{
		url: url,
	}
}

func (app *App) Get(endpoint string) string {
	res, err := http.Get(app.url + endpoint)
	if err != nil {
		fmt.Print(err)
		return ""
	}
	defer res.Body.Close()
	data, _ := ioutil.ReadAll(res.Body)
	fmt.Print(string(data))
	return string(data)
}

func (app *App) PutData(endpoint string, ds string) error {
	url := app.url + endpoint
	r, err := http.NewRequest("PUT", url, strings.NewReader(ds))
	if err != nil {
		return err
	}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
