package app

import (
	"context"
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
	url := app.url + endpoint
	r, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		fmt.Print(err)
		return ""
	}
	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		fmt.Print(err)
		return ""
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Print(string(data))
	return string(data)
}

func (app *App) PutData(endpoint string, ds string) error {
	url := app.url + endpoint
	r, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, strings.NewReader(ds))
	if err != nil {
		fmt.Print(err)
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
