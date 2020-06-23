package dynamicplans

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
)

func FromEnv() ([]*template.Template, error) {
	planPath, found := os.LookupEnv("ATLAS_BROKER_TEMPLATEDIR")
	if !found {
		planPath = "./samples/plans"
	}

	files, err := ioutil.ReadDir(planPath)
	if err != nil {
		return nil, err
	}

	templates := []*template.Template{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		ext := filepath.Ext(f.Name())
		if ext != ".yml" && ext != ".yaml" && ext != ".json" {
			continue
		}

		text, err := ioutil.ReadFile(filepath.Join(planPath, f.Name()))
		if err != nil {
			return nil, err
		}

		basename := strings.TrimSuffix(f.Name(), ext)
		t, err := template.
			New(basename).
			Funcs(map[string]interface{}{
				"yaml": func(v interface{}) (string, error) {
					out, err := yaml.Marshal(v)
					return string(out), err
				},
				"json": func(v interface{}) (string, error) {
					out, err := json.Marshal(v)
					// TODO: remove this atrocity when github.com/goccy/go-yaml/issues/142 is fixed
					return strings.ReplaceAll(string(out), ":", ": "), err
				},
				"required": func(v string) (string, error) {
					if len(v) == 0 {
						return v, errors.New("required value is empty")
					}
					return v, nil
				},
			}).
			Parse(string(text))

		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	if len(templates) == 0 {
		return nil, errors.New("no templates found")
	}

	return templates, nil
}
