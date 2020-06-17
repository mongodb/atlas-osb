package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/goccy/go-yaml"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
)

type PlanContext struct {
	Project        mongodbatlas.Project
	PrivateKey     string
	PublicKey      string
	CollectionName string
	DatabaseName   string
	APIKey         mongodbatlas.APIKey
}

// Plan represents a set of MongoDB Atlas resources
type Plan struct {
	Version       string                             `json:"version,omitempty" yaml:"version,omitempty"`
	Name          string                             `json:"name,omitempty" yaml:"name,omitempty"`
	Description   string                             `json:"description,omitempty" yaml:"description,omitempty"`
	ApiKeys       []*mongodbatlas.APIKey             `json:"apiKeys,omitempty" yaml:"apiKeys,omitempty"`
	Project       *mongodbatlas.Project              `json:"project,omitempty" yaml:"project,omitempty"`
	Clusters      []*mongodbatlas.Cluster            `json:"clusters,omitempty" yaml:"clusters,omitempty"`
	DatabaseUsers []*mongodbatlas.DatabaseUser       `json:"databaseUsers,omitempty" yaml:"databaseUsers,omitempty"`
	IPWhitelists  []*mongodbatlas.ProjectIPWhitelist `json:"ipWhitelists,omitempty" yaml:"ipWhitelists,omitempty"`
}

func getPlans() ([]*template.Template, error) {
	planPath := getEnvOrDefault("ATLAS_BROKER_TEMPLATEDIR", "./samples/plans")
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

		t, err := template.New(f.Name()).Parse(string(text))
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, nil
}

func main() {
	t, err := getPlans()
	if err != nil {
		log.Fatal(err)
	}

	ctx := PlanContext{
		Project: mongodbatlas.Project{
			Name:         "testProject",
			ID:           "id123456",
			OrgID:        "oid123456",
			ClusterCount: 1,
			Created:      time.Now().String(),
			Links:        nil,
		},
		APIKey: mongodbatlas.APIKey{
			PrivateKey: "privkeyABCDEF123456",
			PublicKey:  "pubkeyABCDEF123456",
		},
	}

	b := new(bytes.Buffer)
	for _, t := range t {
		if err := t.Execute(b, ctx); err != nil {
			log.Fatal(err)
		}

		p := Plan{}
		yaml.NewDecoder(b).Decode(&p)
		json.NewEncoder(os.Stdout).Encode(p)
	}

	hc, err := digest.NewTransport("your public key", "your private key").Client()
	if err != nil {
		log.Fatal(err)
	}

	c, err := mongodbatlas.New(hc, mongodbatlas.SetBaseURL("https://cloud.mongodb.com"))
	if err != nil {
		log.Fatal(err)
	}

}
