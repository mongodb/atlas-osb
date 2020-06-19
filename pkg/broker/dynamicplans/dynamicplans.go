package dynamicplans

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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
		t, err := template.New(basename).Parse(string(text))
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, nil
}
