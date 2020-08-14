// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dynamicplans

import (
	"errors"
	"io/ioutil"
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
				"default": dfault,
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

// custom default function to fix Sprig's stupidity with booleans
func dfault(d interface{}, given ...interface{}) interface{} {
	if empty(given) || empty(given[0]) {
		return d
	}
	return given[0]
}

// custom empty function to fix Sprig's stupidity with booleans
func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
	default:
		return g.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return g.Len() == 0
	case reflect.Bool:
		// return !g.Bool()
		return false // bool can NEVER be empty!
	case reflect.Complex64, reflect.Complex128:
		return g.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return g.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return g.Float() == 0
	case reflect.Struct:
		return false
	}
}
