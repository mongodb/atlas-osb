package main

import (
    "context"
    "github.com/Sectorbob/mlab-ns2/gae/ns/digest"
    "flag"
    //"github.com/davecgh/go-spew/spew"
    //"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/mongodbrealm"
    "io/ioutil"
    "fmt"
    "log"
    "os"
    "encoding/json"
    //"gopkg.in/yaml.v2"
    //osb "sigs.k8s.io/go-open-service-broker-client/v2"
)

const (
	publicKeyApiEnv  = "ATLAS_PUBLIC_KEY"
	privateKeyApiEnv = "ATLAS_PRIVATE_KEY"
	projectIDEnv  = "ATLAS_PROJECT_ID"
    appIDEnv = "ATLAS_APP_ID"
)

var (
    envPublicApiKey  = os.Getenv(publicKeyApiEnv)
	envPrivateApiKey = os.Getenv(privateKeyApiEnv)
	envProjectID  = os.Getenv(projectIDEnv)
	envAppID = os.Getenv(appIDEnv)
)


func main() {

    // Simple cli to manage
    // Realm app values
    // Usage
    // $realmval --groupid <GROUP_ID> --appid <APP_ID> <Key> [Value|-f <PathToValueFile>]
    //
    // If no appid, then pass --create-app <APPNAME|We generate a name>
    // Values needs to be valid JSON string or file

    var groupID string
    var appID string
    var publicApiKey string
    var privateApiKey string
    var key string
    var value string
    var verbose bool
    var createApp bool
    var deleteApp bool
    var deleteFlag bool
    var private bool

    flag.BoolVar(&verbose,"verbose",false,"Enable verbose output")
    flag.BoolVar(&createApp,"create-app",false,"Set to create the app from --value")
    flag.BoolVar(&deleteApp,"delete-app",false,"Set to delelete the given -appid")
    flag.BoolVar(&deleteFlag,"delete-key",false,"Set to delelete the given --key (as _id for key)")
    flag.StringVar(&groupID, "groupid", envProjectID, "MongoDB Atlas Project Id, env ATLAS_PROJECT_ID")
    flag.StringVar(&appID, "appid", envAppID, "MongoDB Realm App Id, env ATLAS_APP_ID")
    flag.StringVar(&key, "key", "", "Key for new value, or used as keyid without value or for --delete")
    flag.StringVar(&value, "value", "", "JSON string for your new value")
    flag.BoolVar(&private, "private", false, "Set to mark new value as private, default false")
    flag.StringVar(&publicApiKey, "publicApiKey", envPublicApiKey, "MongoDB Atlas Public Api Key, or ATLAS_PUBLIC_KEY") 
    flag.StringVar(&privateApiKey, "privateApiKey",envPrivateApiKey, "MongoDB Atlas Private Api Key, or ATLAS_PUBLIC_KEY") 

    flag.Parse()

    if !verbose {
        log.SetOutput(ioutil.Discard)
    }


    t := digest.NewTransport(publicApiKey, privateApiKey)
    tc, err := t.Client()
    if err != nil {
        log.Fatalf(err.Error())
    }
    atlasclient := mongodbrealm.NewClient(tc)
    
    atlasclient.SetCurrentRealmAtlasApiKey ( &mongodbrealm.RealmAtlasApiKey{
        Username: publicApiKey,
        Password: privateApiKey,
    })

    log.Printf("atlasclient.GetCurrentRealmAtlasApiKey(): %+v", atlasclient.GetCurrentRealmAtlasApiKey())


    if (len(appID) > 0) && (len(key) > 0) {

        if deleteFlag {
            log.Printf("attempt delete realmValue with keyID: %+v",key)
            value, err := atlasclient.RealmValues.Delete(context.Background(),groupID,appID,key)
            if err != nil {
                log.Fatalf(err.Error())
            }
            log.Printf("delete done: %+v",value)
            v, _ := json.Marshal(value)
            fmt.Println(string(v))
        } else {
            if len(value)!=0 {
                realmValue,err := atlasclient.RealmValueFromString(key, value)
                realmValue.Private = private
                if err != nil {
                    log.Fatalf(err.Error())
                }
                log.Printf("attempt create groupID:%s appID:%s realmValue:%+v",groupID, appID,realmValue)
                value, _, err := atlasclient.RealmValues.Create(context.Background(),groupID,appID,realmValue)
                if err != nil {
                    log.Fatalf(err.Error())
                }
                log.Printf("create done %+v",value)
                v, _ := json.Marshal(value)
                fmt.Println(string(v))
            } else {
                log.Printf("Found key %s but no value, attempt get value with _id=%s",key,key)
                value, _, err := atlasclient.RealmValues.Get(context.Background(),groupID,appID,key)
                if err != nil {
                    log.Fatalf(err.Error())
                }
                log.Printf("create done %+v",value)
                v, _ := json.Marshal(value)
                fmt.Println(string(v))
            }
        }
    }  

    if len(appID) > 0 {   // list values
        if deleteApp {

            log.Printf("attempt delete realm app with appID: %+v",appID)
            app, err := atlasclient.RealmApps.Delete(context.Background(),groupID,appID)
            if err != nil {
                log.Fatalf(err.Error())
            }
            log.Printf("delete app done: %+v",app)
            v, _ := json.Marshal(app)
            fmt.Println(string(v))
        } else {
            log.Printf("listing values")
            values, _, err := atlasclient.RealmValues.List(context.Background(),groupID,appID,nil)
            if err != nil {
                log.Fatalf(err.Error())
            }

            v, _ := json.Marshal(values)
            fmt.Println(string(v))
        }
    } else {    // list apps
        if createApp {

            if len(value)==0 {
                log.Fatalf("create-app set but no --value")
            }
            appInput,err := atlasclient.RealmAppInputFromString(value)
            if err != nil {
                log.Fatalf(err.Error())
            }
            log.Printf("Attempt create appInput: %+v",appInput)
            app, _, err := atlasclient.RealmApps.Create(context.Background(), groupID, appInput)  
            if err != nil {
                log.Fatalf(err.Error())
            }

            v, _ := json.Marshal(app)
            fmt.Println(string(v))

        } else {
            apps, _, err := atlasclient.RealmApps.List(context.Background(),groupID,nil)
            if err != nil {
                log.Fatalf(err.Error())
            }

            v, _ := json.Marshal(apps)
            fmt.Println(string(v))
        }
    }

}



