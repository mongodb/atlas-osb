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

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/atlas/mongodbatlas"
	"gopkg.in/yaml.v2"
	osb "sigs.k8s.io/go-open-service-broker-client/v2"
)

const (
	publicKeyAPIEnv           = "ATLAS_PUBLIC_KEY"
	privateKeyAPIEnv          = "ATLAS_PRIVATE_KEY"
	brokerHTTPAuthUsernameEnv = "ATLAS_BROKER_HTTP_AUTH_USERNAME"
	brokerHTTPAuthPasswordEnv = "ATLAS_BROKER_HTTP_AUTH_PASSWORD" // nolint
	// projectIDEnv  = "ATLAS_PROJECT_ID"
	brokerHostportEnv = "ATLAS_BROKER_HOSTPORT"
)

var (
	envPublicAPIKey           = os.Getenv(publicKeyAPIEnv)
	envPrivateAPIKey          = os.Getenv(privateKeyAPIEnv)
	envBrokerHTTPAuthUsername = os.Getenv(brokerHTTPAuthUsernameEnv)
	envBrokerHTTPAuthPassword = os.Getenv(brokerHTTPAuthPasswordEnv)
	envBrokerHostport         = os.Getenv(brokerHostportEnv)
)

func GetBroker(url string) (osb.Client, *mongodbatlas.Client, error) {
	config := osb.DefaultClientConfiguration()
	config.URL = url
	config.Insecure = true

	basicAuthConfig := &osb.BasicAuthConfig{
		Username: envBrokerHTTPAuthUsername,
		Password: envBrokerHTTPAuthPassword,
	}

	config.AuthConfig = &osb.AuthConfig{
		BasicAuthConfig: basicAuthConfig,
	}

	client, err := osb.NewClient(config)
	if err != nil {
		return nil, nil, err
	}

	t := digest.NewTransport(envPublicAPIKey, envPrivateAPIKey)
	tc, err := t.Client()
	if err != nil {
		log.Fatalf(err.Error())
	}

	atlasclient := mongodbatlas.NewClient(tc)

	return client, atlasclient, nil
}

func main() {
	var plan string
	var name string
	var params string
	var operation string
	var otherArgs []string
	var verbose bool

	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.StringVar(&operation, "op", "catalog", "Broker operation catalog, provision, etc")
	flag.StringVar(&params, "values", "", "parameters in yaml filename or string")
	flag.StringVar(&plan, "plan", "", "name of plan to create")
	flag.StringVar(&name, "name", "", "name for plan instance (service name)")

	flag.Parse()

	// if !verbose {
	//    log.SetOutput(ioutil.Discard)
	// }

	// if flag.NArg() == 0 {
	//    flag.Usage()
	//    os.Exit(1)
	// }

	otherArgs = flag.Args()
	log.Printf("other args: %+v\n", otherArgs)

	log.Println("params:", params)
	var parameters map[string]interface{}

	if len(params) > 0 {
		yamlFile, err := ioutil.ReadFile(params)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
			yamlFile = []byte(params)
		}
		log.Println("yamlFile:", yamlFile)
		err = json.Unmarshal(yamlFile, &parameters)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}
		log.Println("parameters:", parameters)
	} else {
		log.Println("where is my automobile?")
	}

	url := envBrokerHostport
	if len(url) == 0 {
		url = "http://localhost:4000"
	}
	log.Println("url", url)
	broker, client, err := GetBroker(url)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	if operation == "catalog" {
		catalog, err2 := broker.GetCatalog()
		log.Println("catalog", catalog)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		// d, err := yaml.Marshal(&catalog)
		d, err := json.Marshal(&catalog)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("%s", string(d))
	}
	if operation == "services" {
		var groupID string
		if len(otherArgs) > 0 {
			groupID = otherArgs[0]
		} else {
			groupID = os.Getenv("ATLAS_GROUP_ID")
		}
		log.Println("services groupId:", groupID)
		var err2 error
		clusters, _, err2 := client.Clusters.List(context.Background(), groupID, nil)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		d, err := json.Marshal(&clusters)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("%s", string(d))
		users, _, err2 := client.DatabaseUsers.List(context.Background(), groupID, nil)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		e, err2 := json.Marshal(&users)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		log.Printf("%s", string(e))

		ips, _, err2 := client.ProjectIPWhitelist.List(context.Background(), groupID, nil)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		f, err2 := json.Marshal(&ips)
		if err2 != nil {
			log.Fatalf("error: %v", err2)
		}
		log.Printf("%s", string(f))
	}
	if operation == "create-service" {
		log.Printf("create-service")
		log.Println("parameters:", parameters)
		if len(plan) == 0 || len(name) == 0 {
			log.Fatalln("missing --plan or --name")
		}
		serviceID := "aosb-cluster-service-template"
		servicePlan := fmt.Sprintf("%s-%s", "aosb-cluster-plan-template", plan)
		serviceInstanceName := name

		log.Println("servicePlan:", servicePlan)
		log.Println("serviceInstanceName:", serviceInstanceName)

		request := &osb.ProvisionRequest{
			InstanceID:        serviceInstanceName,
			ServiceID:         serviceID,
			PlanID:            servicePlan,
			Parameters:        parameters,
			OrganizationGUID:  "fake",
			SpaceGUID:         "fake",
			AcceptsIncomplete: true,
		}
		log.Println("request:", request)

		resp, err := broker.ProvisionInstance(request)
		if err != nil {
			fmt.Println(">>>>>>>", err)
			errHTTP, isError := osb.IsHTTPError(err)
			if isError {
				// handle error response from broker
				fmt.Println("errHttp:", errHTTP)
			} else {
				// handle errors communicating with the broker
				fmt.Println("error provision:", err)
			}
		}

		log.Println("resp:", resp)
	}
	if operation == "get-instance" {
		req := &osb.GetInstanceRequest{
			InstanceID: name,
		}
		resp, err := broker.GetInstance(req)
		if err != nil {
			fmt.Println(">>>>>>>", err)
			errHTTP, isError := osb.IsHTTPError(err)
			if isError {
				// handle error response from broker
				fmt.Println("errHttp:", errHTTP)
			} else {
				// handle errors communicating with the broker
				fmt.Println("error provision:", err)
			}
		}

		log.Println("resp:", resp)
	}
	if operation == "bind" || operation == "unbind" {
		if len(otherArgs) < 2 {
			log.Fatalln("Missing <SERVICE_INSTANCE_NAME> <BINDING_ID>")
		}
		serviceInstanceName := otherArgs[0]
		log.Println("serviceInstanceName:", serviceInstanceName)
		bindingID := otherArgs[1]
		log.Println("bindingId:", bindingID)
		spew.Dump(parameters)
		if operation == "bind" {
			request := &osb.BindRequest{
				InstanceID: serviceInstanceName,
				Parameters: parameters,
				BindingID:  bindingID,
				ServiceID:  "aosb-cluster-service-aws",
				PlanID:     "aosb-cluster-plan-aws-m10",
			}
			log.Println("bind request:", request)
			resp, err := broker.Bind(request)
			if err != nil {
				errHTTP, isError := osb.IsHTTPError(err)
				if isError {
					fmt.Println("errHttp:", errHTTP)
				} else {
					fmt.Println("error bind:", err)
					fmt.Println("resp:", resp)
				}
			}
			d, err2 := yaml.Marshal(&resp)
			if err2 != nil {
				log.Fatalf("error: %v", err2)
			}
			fmt.Printf("%s", string(d))
		}
		if operation == "unbind" {
			request := &osb.UnbindRequest{
				InstanceID: serviceInstanceName,
				BindingID:  bindingID,
				ServiceID:  "aosb-cluster-service-aws",
				PlanID:     "aosb-cluster-plan-aws-m10",
			}
			log.Println("unbind request:", request)
			resp, err := broker.Unbind(request)
			if err != nil {
				errHTTP, isError := osb.IsHTTPError(err)
				if isError {
					fmt.Println("errHttp:", errHTTP)
				} else {
					fmt.Println("error provision:", err)
				}
			}
			d, err2 := yaml.Marshal(&resp)
			if err2 != nil {
				log.Fatalf("error: %v", err2)
			}
			fmt.Printf("%s", string(d))
		}
	}
}
