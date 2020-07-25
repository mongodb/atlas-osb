package statestorage 

import (
	"context"
	"fmt"
    "log"
    "errors"
	"net/http"
    "net/url"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/mongodbrealm"
	"go.uber.org/zap"
)

const (
    BROKER_MAINENTANCE_PROJECT_NAME = "Atlas Service Broker Mainentance"
    BROKER_REALM_STATE_APP_NAME = "broker-state"
)


type RealmStateStorage struct {
	OrgID       string                 `json:"orgId,omitempty"`
	Client      *mongodbatlas.Client   
    RealmClient *mongodbrealm.Client
    RealmApp    *mongodbrealm.RealmApp
    RealmProject  *mongodbatlas.Project
}
func keyForOrg(key *mongodbatlas.APIKey, orgID string) (bool) {
    for role := range key.Roles {
        if key.Roles[role].OrgID == orgID {
            return true
        }
    }
    return false
}

func GetRealmStateStorage(creds *credentials.Credentials, baseURL string,logger *zap.SugaredLogger, orgID string) (*RealmStateStorage, error) {
    if len(orgID) == 0 {
        return nil, errors.New("orgID must be set")
    }

	if len(creds.Orgs)==0 {
        return nil, fmt.Errorf("Cannot use OrgStateStorage without Org apikey")
	}

    var orgKey *mongodbatlas.APIKey

    for k, v := range creds.Orgs {
        if keyForOrg(&v.APIKey, orgID) {
            orgKey = &v.APIKey
		    logger.Infow("Using org key for realm storage", "k", k)
        }
    }

	if orgKey==nil {
		return nil, fmt.Errorf("Not able to find orgID=%s in credentials", orgID)
	}

    hc, err := digest.NewTransport(orgKey.PublicKey, orgKey.PrivateKey).Client()
    if err != nil {
        return nil, err
    }

    client, err := mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
    if err != nil {
        return nil, err
    }
    realmClient, err := mongodbrealm.New(hc, mongodbrealm.SetBaseURL(baseURL))
    if err != nil {
        return nil, err
    }

    // Get or create a RealmApp for this orgID -
    // Each Organization using the broker will have 1 special 
    // Atlas Group - called "Atlas Service Broker"
    //
    mainPrj, err := getOrCreateBrokerMaintentaceGroup(orgID, client)
    if err != nil {
        log.Fatalf(err.Error())
        return nil, err
    }
    realmApp, err := getOrCreateRealmAppForOrg(mainPrj.ID, realmClient)
    if err != nil {
        log.Fatalf(err.Error())
        return nil, err
    }
    rss := &RealmStateStorage{
        OrgID: orgID,
        Client: client,
        RealmClient: realmClient,
        RealmApp: realmApp,
        RealmProject: mainPrj,
    }
    return rss, nil
}

func getOrCreateBrokerMaintentaceGroup(orgID string, client *mongodbatlas.Client) (*mongodbatlas.Project, error) {
    p := BROKER_MAINENTANCE_PROJECT_NAME
    project, _, err := client.Projects.GetOneProjectByName(context.Background(), p)
    if err != nil {
        log.Printf("getOrCreateBrokerMaintentaceGroup err:%+v",err)
        prj := mongodbatlas.Project{
            Name: p,
            OrgID: orgID,
        }
        project, _, err = client.Projects.Create(context.Background(), &prj)
        if err != nil {
            return nil, err
        }
    }
    return project, nil
}
func getOrCreateRealmAppForOrg(groupID string, realmClient *mongodbrealm.Client) (*mongodbrealm.RealmApp, error) {
    app := mongodbrealm.RealmAppInput{
        Name: BROKER_REALM_STATE_APP_NAME,
        ClientAppID: "atlas-osb",
        Location: "to-do-can-we-get-cf-space-info",
    }

    realmApp, _, err := realmClient.RealmApps.Get(context.Background(), groupID, app.Name)
    if err != nil {
        log.Printf("Error fetching maintenance realm app: %+v",err)
        log.Printf("Attempt create app: %+v",app)
        realmApp, _, err := realmClient.RealmApps.Create(context.Background(), groupID, &app)  
        if err != nil {
            log.Fatalf(err.Error())
            return nil, err
        }
        log.Printf("Created realm app: %+v",realmApp)
        return nil, err
    } else {
        log.Printf("Found existing realm app: %+v",realmApp)
    }
    return realmApp, nil
}

func (ss *RealmStateStorage) Put(key string, value map[string]interface{}) (*mongodbrealm.RealmValue, error) {
    val := &mongodbrealm.RealmValue{
        Name: key,
        Value: value,
    }
    v, _, err := ss.RealmClient.RealmValues.Create(context.Background(),ss.RealmProject.ID,ss.RealmApp.ID, val)
    return v, err
}

func (ss *RealmStateStorage) Get(key string) (*mongodbrealm.RealmValue, error) {
    v, _, err := ss.RealmClient.RealmValues.Get(context.Background(),ss.RealmProject.ID,ss.RealmApp.ID, key)
    return v, err
}



var defaultUser = &mongodbatlas.DatabaseUser{
    Username:     "admin",
    Password:     "admin",
    Roles: []mongodbatlas.Role{{
        DatabaseName: "statestorage",
        RoleName:     "readWrite",
    }},
}

func GetOrgStateStorage(creds *credentials.Credentials, baseURL string,logger *zap.SugaredLogger) (*mongodbatlas.Cluster, string, error) {

	if len(creds.Orgs)==0 {
		return nil, "", fmt.Errorf("Cannot use OrgStateStorage without Org apikey")
	}

    okeys := make([]string, 0, len(creds.Orgs))
    for k := range creds.Orgs {
        okeys = append(okeys, k)
    }
    okey := okeys[0]
    hc, err := digest.NewTransport(creds.Orgs[okey].PublicKey, creds.Orgs[okey].PrivateKey).Client()
    if err != nil {
        return nil, "", err
    }

    client, err := mongodbatlas.New(hc, mongodbatlas.SetBaseURL(baseURL))
    if err != nil {
        return nil, "", err
    }

    // algorithm
    // try to get the "special atlas-osb" project for this org.
    // this project will have a known fixed-format name.
    // such as, "atlas-osb"
    projectName := "atlas-osb"
    logger.Infof("atlas-osb-%s",okey)
    project, r, err := client.Projects.GetOneProjectByName(context.Background(), projectName)
	if err != nil && (r.StatusCode == http.StatusNotFound || r.StatusCode ==http.StatusUnauthorized) {
		logger.Infow("statestorage project not found, attempt create", "error", err, "projectName", projectName)
		project = &mongodbatlas.Project{}
        project.Name = projectName
        project.OrgID = creds.Orgs[okey].Roles[0].OrgID
        project, _, err = client.Projects.Create(context.Background(), project)
		logger.Infow("statestorage project created","project", project)
	}
    if err != nil {
        return nil, "", err
    }

    clusterName := "statestorage"
    cluster, _, err := client.Clusters.Get(context.Background(), project.ID, clusterName)
	if err != nil && r.StatusCode == http.StatusNotFound {
		logger.Errorw("Failed to get statestorage cluster", "error", err, "clusterName", clusterName)
        // We can add broker config to allow override for these settings.
	    cluster = &mongodbatlas.Cluster{
            ClusterType:   "REPLICASET",
            Name:          clusterName,
            ProviderSettings: &mongodbatlas.ProviderSettings{
                ProviderName:     "AWS",
                InstanceSizeName: "M10",
                RegionName:       "US_EAST_1",
                },
        }
	    cluster, _, err = client.Clusters.Create(context.Background(), project.ID, cluster)
        if err != nil {
            return nil, "", err
        }
        err = nil 
	    logger.Infow("statestorage cluster created","cluster", cluster)


        defaultUser.GroupID = project.ID
        user, _, err := client.DatabaseUsers.Create(context.Background(), project.ID, defaultUser)
        if err != nil {
            return nil, "", err
        }
        logger.Infow("statestorage cluster created: default","user", user)

    }

    if err != nil {
        return nil, "", err
    }

    logger.Infow("Found existing cluster", "cluster", cluster)
    conn, err := url.Parse(cluster.ConnectionStrings.StandardSrv)
    if err != nil {
        logger.Errorw("Failed to parse connection string", "error", err, "connString", cluster.ConnectionStrings.StandardSrv)
    }

    conn.Path = "statestorage"

    logger.Infow("New User ConnectionString", "conn", conn)

    conn.User = url.UserPassword(defaultUser.Username, defaultUser.Password)



    connstr := conn.String()
    return cluster, connstr, nil

}


