package dynamicplans

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
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
			Funcs(template.FuncMap{
				"randelem": func(v interface{}) (interface{}, error) {
					val := reflect.ValueOf(v)
					switch val.Kind() {
					case reflect.Map:
						r := val.MapRange()
						if !r.Next() {
							return nil, nil
						}
						return r.Value().Interface(), nil
					case reflect.Slice, reflect.Array, reflect.String:
						l := val.Len()
						if l == 0 {
							return nil, nil
						}
						return val.Index(rand.Intn(l)), nil
					}

					return nil, errors.New("invalid type")
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
