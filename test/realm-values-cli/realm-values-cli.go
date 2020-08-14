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

	"github.com/mongodb/mongodb-atlas-service-broker/pkg/mongodbrealm"
)

const (
	publicKeyAPIEnv  = "ATLAS_PUBLIC_KEY"
	privateKeyAPIEnv = "ATLAS_PRIVATE_KEY"
	projectIDEnv     = "ATLAS_PROJECT_ID"
	appIDEnv         = "ATLAS_APP_ID"
)

var (
	envPublicAPIKey  = os.Getenv(publicKeyAPIEnv)
	envPrivateAPIKey = os.Getenv(privateKeyAPIEnv)
	envProjectID     = os.Getenv(projectIDEnv)
	envAppID         = os.Getenv(appIDEnv)
)

var (
	groupID       = flag.String("groupid", envProjectID, "MongoDB Atlas Project Id, env ATLAS_PROJECT_ID")
	appID         = flag.String("appid", envAppID, "MongoDB Realm App Id, env ATLAS_APP_ID")
	publicAPIKey  = flag.String("publicApiKey", envPublicAPIKey, "MongoDB Atlas Public Api Key, or ATLAS_PUBLIC_KEY")
	privateAPIKey = flag.String("privateApiKey", envPrivateAPIKey, "MongoDB Atlas Private Api Key, or ATLAS_PUBLIC_KEY")
	key           = flag.String("key", "", "Key for new value, or used as keyid without value or for --delete")
	value         = flag.String("value", "", "JSON string for your new value")
	verbose       = flag.Bool("verbose", false, "Enable verbose output")
	createApp     = flag.Bool("create-app", false, "Set to create the app from --value")
	deleteApp     = flag.Bool("delete-app", false, "Set to delelete the given -appid")
	deleteFlag    = flag.Bool("delete-key", false, "Set to delelete the given --key (as _id for key)")
	private       = flag.Bool("private", false, "Set to mark new value as private, default false")
)

func main() {
	// Simple cli to manage
	// Realm app values
	// Usage
	// $realmval --groupid <GROUP_ID> --appid <APP_ID> <Key> [Value|-f <PathToValueFile>]
	//
	// If no appid, then pass --create-app <APPNAME|We generate a name>
	// Values needs to be valid JSON string or file
	flag.Parse()

	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}

	atlasclient, err := mongodbrealm.New(
		context.Background(),
		nil,
		mongodbrealm.SetAPIAuth(context.Background(), *publicAPIKey, *privateAPIKey),
	)
	if err != nil {
		log.Fatal(err)
	}

	if (len(*appID) > 0) && (len(*key) > 0) {
		if *deleteFlag {
			log.Printf("attempt delete realmValue with keyID: %+v", key)
			value, err := atlasclient.RealmValues.Delete(context.Background(), *groupID, *appID, *key)
			if err != nil {
				log.Fatalf(err.Error())
			}
			log.Printf("delete done: %+v", value)
			v, _ := json.Marshal(value)
			fmt.Println(string(v))
		} else {
			if len(*value) != 0 {
				realmValue, err := atlasclient.RealmValueFromString(*key, *value)
				realmValue.Private = *private
				if err != nil {
					log.Fatalf(err.Error())
				}
				log.Printf("attempt create groupID:%s appID:%s realmValue:%+v", *groupID, *appID, realmValue)
				value, _, err := atlasclient.RealmValues.Create(context.Background(), *groupID, *appID, realmValue)
				if err != nil {
					log.Fatalf(err.Error())
				}
				log.Printf("create done %+v", value)
				v, _ := json.Marshal(value)
				fmt.Println(string(v))
			} else {
				log.Printf("Found key %q but no value, attempt get value with _id=%q", *key, *key)
				value, _, err := atlasclient.RealmValues.Get(context.Background(), *groupID, *appID, *key)
				if err != nil {
					log.Fatalf(err.Error())
				}
				log.Printf("get done %+v", value)
				v, _ := json.Marshal(value)
				fmt.Println(string(v))
			}
		}
	}

	if len(*appID) > 0 { // list values
		if *deleteApp {
			log.Printf("attempt delete realm app with appID: %+v", *appID)
			app, err := atlasclient.RealmApps.Delete(context.Background(), *groupID, *appID)
			if err != nil {
				log.Fatalf(err.Error())
			}
			log.Printf("delete app done: %+v", app)
			v, _ := json.Marshal(app)
			fmt.Println(string(v))
		} else {
			log.Printf("listing values")
			values, _, err := atlasclient.RealmValues.List(context.Background(), *groupID, *appID, nil)
			if err != nil {
				log.Fatalf(err.Error())
			}

			v, _ := json.Marshal(values)
			fmt.Println(string(v))
		}
	} else { // list apps
		if *createApp {
			if len(*value) == 0 {
				log.Fatalf("create-app set but no --value")
			}
			appInput, err := atlasclient.RealmAppInputFromString(*value)
			if err != nil {
				log.Fatalf(err.Error())
			}
			log.Printf("Attempt create appInput: %+v", appInput)
			app, _, err := atlasclient.RealmApps.Create(context.Background(), *groupID, appInput)
			if err != nil {
				log.Fatalf(err.Error())
			}

			v, _ := json.Marshal(app)
			fmt.Println(string(v))
		} else {
			apps, _, err := atlasclient.RealmApps.List(context.Background(), *groupID, nil)
			if err != nil {
				log.Fatalf(err.Error())
			}

			v, _ := json.Marshal(apps)
			fmt.Println(string(v))
		}
	}
}
