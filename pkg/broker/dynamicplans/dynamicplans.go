package dynamicplans

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

func FromEnv() ([]*template.Template, error) {
	planPath, found := os.LookupEnv("ATLAS_BROKER_TEMPLATEDIR")
	if !found {
		return nil, nil
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
		if ext != ".tpl" {
			continue
		}

		text, err := ioutil.ReadFile(filepath.Join(planPath, f.Name()))
		if err != nil {
			return nil, err
		}

		// trim .tpl
		basename := strings.TrimSuffix(f.Name(), ext)
		// also trim .yml/.yaml/.json (if any)
		basename = strings.TrimSuffix(basename, filepath.Ext(basename))

		t, err := template.
			New(basename).
			Funcs(sprig.TxtFuncMap()).
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
