package statestorage 

import (
	"context"
	"fmt"
    "errors"
	"net/http"
    "net/url"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/mongodbrealm"
	"github.com/pivotal-cf/brokerapi/domain"
	"go.uber.org/zap"
    "strings"
    "reflect"
    "encoding/json"
)

const (
    BROKER_MAINENTANCE_PROJECT_NAME = "Atlas Service Broker Mainentance"
    BROKER_REALM_STATE_APP_NAME = "broker-state"
)

var (
    InstanceNotFound    = errors.New("unable to find instance in state storage")
)

//type StateStorage interface {
//    FindOne(context.Context, string)                   (*map[string]interface{}, error)
//    InsertOne(context.Context, string, interface{})    (*map[string]interface{}, error)
//    DeleteOne(context.Context, string)                 (error)
//
//}


type RealmStateStorage struct {
	OrgID       string                 `json:"orgId,omitempty"`
	Client      *mongodbatlas.Client   
    RealmClient *mongodbrealm.Client
    RealmApp    *mongodbrealm.RealmApp
    RealmProject  *mongodbatlas.Project
    Logger      *zap.SugaredLogger
}

func keyForOrg(key *mongodbatlas.APIKey, orgID string) (bool) {
    for role := range key.Roles {
        if key.Roles[role].OrgID == orgID {
            return true
        }
    }
    return false
}

func OrgIDs(creds *credentials.Credentials) ([]string) {
    orgs := make([]string, 0, len(creds.Orgs))
    for _, value := range creds.Orgs {
        for role := range value.APIKey.Roles {
            orgs = append(orgs, value.APIKey.Roles[role].OrgID)
        }
    }
    return orgs
}

func GetStateStorage(creds *credentials.Credentials, baseURL string,logger *zap.SugaredLogger, orgID string) (*RealmStateStorage, error) {
	if len(creds.Orgs)==0 {
        return nil, fmt.Errorf("Cannot use OrgStateStorage without Org apikey")
	}

    if len(orgID) == 0 {
        orgs := OrgIDs(creds)
        logger.Infow("GetStateStorage --- ","creds",creds)

        logger.Infow("GetStateStorage --- ","orgs",orgs)
        if len(orgs) == 0 {
            return nil, errors.New("orgID must be set")
        }
        orgID = orgs[0]
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
    realmClient.SetCurrentRealmAtlasApiKey ( &mongodbrealm.RealmAtlasApiKey{
        Username: orgKey.PublicKey,
        Password: orgKey.PrivateKey,
    })

    // Get or create a RealmApp for this orgID -
    // Each Organization using the broker will have 1 special 
    // Atlas Group - called "Atlas Service Broker"
    //
    mainPrj, err := getOrCreateBrokerMaintentaceGroup(orgID, client, logger)
    if err != nil {
        logger.Errorw("Error getOrCreateBrokerMaintentaceGroup","err",err)
        return nil, err
    }
    logger.Infow("Found mainteneance project","mainPrj",mainPrj)
    realmApp, err := getOrCreateRealmAppForOrg(mainPrj.ID, realmClient, logger)
    if err != nil {
        logger.Errorw("Error getOrCreateRealmAppForOrg","err",err)
        return nil, err
    }
    rss := &RealmStateStorage{
        OrgID: orgID,
        Client: client,
        RealmClient: realmClient,
        RealmApp: realmApp,
        RealmProject: mainPrj,
        Logger: logger,
    }
    return rss, nil
}

func getOrCreateBrokerMaintentaceGroup(orgID string, client *mongodbatlas.Client,logger *zap.SugaredLogger) (*mongodbatlas.Project, error) {
    p := BROKER_MAINENTANCE_PROJECT_NAME
    project, _, err := client.Projects.GetOneProjectByName(context.Background(), p)
    if err != nil {
        logger.Infow("getOrCreateBrokerMaintentaceGroup","err",err)
        prj := mongodbatlas.Project{
            Name: p,
            OrgID: orgID,
        }
        project, _, err = client.Projects.Create(context.Background(), &prj)
        if err != nil {
            return nil, err
        }
        logger.Infow("getOrCreateBrokerMaintentaceGroup CREATED","project",project)
    }
    logger.Infow("getOrCreateBrokerMaintentaceGroup FOUND","project",project)
    return project, nil
}

func getOrCreateRealmAppForOrg(groupID string, realmClient *mongodbrealm.Client,logger *zap.SugaredLogger) (*mongodbrealm.RealmApp, error) {
    app := mongodbrealm.RealmAppInput{
        Name: BROKER_REALM_STATE_APP_NAME,
        ClientAppID: "atlas-osb",
        Location: "US-VA",
        /* [US-VA, AU, US-OR, IE] */
    }

    apps, _, err := realmClient.RealmApps.List(context.Background(),groupID,nil)
    var realmApp *mongodbrealm.RealmApp

    for _, ra := range apps {
        logger.Infow("Found realm app","ra",ra)
        if ra.Name == app.Name {
            realmApp = &ra
        }
    }
    //realmApp, _, err := realmClient.RealmApps.Get(context.Background(), groupID, app.Name)
    if realmApp == nil {
        logger.Infow("Error fetching maintenance realm app","err",err)
        logger.Infow("Attempt create","app",app)
        realmApp, _, err := realmClient.RealmApps.Create(context.Background(), groupID, &app)  
        if err != nil {
            logger.Errorw("Error createing realm app", "err", err)
            return nil, err
        }
        logger.Infow("Created realm app","realmApp",realmApp)
        return nil, err
    } else {
        logger.Infow("Found existing realm app","realmApp",realmApp)
    }
    return realmApp, nil
}



func (ss *RealmStateStorage) FindOne(ctx context.Context, key string) (*domain.GetInstanceDetailsSpec, error) {
    // Need to find the one value whose "name" = key
    values, _, err := ss.RealmClient.RealmValues.List(ctx,ss.RealmProject.ID,ss.RealmApp.ID,nil)
    if err != nil {
        // return proper InstanceNotFound, if error is realm
        if strings.Contains(err.Error(), "value not found") {
            err = InstanceNotFound
        }
        return nil, err
    }

    idForKey := ""

    for _, v := range values {
        if v.Name == key {
            idForKey = v.ID
        }
    }

    val, err := ss.Get(ctx,idForKey)
    if err != nil {
        // return proper InstanceNotFound, if error is realm
        if strings.Contains(err.Error(), "value not found") {
            err = InstanceNotFound
        }
        return nil, err
    }
    if val.Value == nil {
        return nil, errors.New("val.Value was nil from realm, should never happen")
    }
    spec := domain.GetInstanceDetailsSpec{}
    //err = json.Unmarshal([]byte(val.Value), &spec)
    sss := reflect.ValueOf(val.Value).Interface().(string) 
    err = json.Unmarshal(([]byte)sss, &spec)
    if err != nil {
        return nil, err
    }


    //spec := &domain.GetInstanceDetailsSpec{
    //    ServiceID: fmt.Sprintf("%s",val.Value["serviceID"]),
    //    PlanID: val.Value["planID"].(string),
    //    DashboardURL: val.Value["dashboardURL"].(string),
    //    Parameters: reflect.ValueOf(val.Value["parameters"]).Interface().(interface{}),
    //}

    return &spec, nil
}

func (ss *RealmStateStorage) DeleteOne(ctx context.Context, key string) (error) {
    _, err := ss.RealmClient.RealmValues.Delete(ctx,ss.RealmProject.ID,ss.RealmApp.ID, key)
    return err

}

func (ss *RealmStateStorage) Put(ctx context.Context, key string, value *domain.GetInstanceDetailsSpec) (*mongodbrealm.RealmValue, error) {

    vv := make(map[string]interface{})
    vv["serviceID"]=value.ServiceID
    vv["planID"]=value.PlanID
    vv["dashboardURL"]=value.DashboardURL
    vv["parameters"]=value.Parameters
    val := &mongodbrealm.RealmValue{
        Name: key,
        Value: vv,
    }
    v, _, err := ss.RealmClient.RealmValues.Create(ctx,ss.RealmProject.ID,ss.RealmApp.ID, val)
    return v, err
}

func (ss *RealmStateStorage) Get(ctx context.Context, key string) (*mongodbrealm.RealmValue, error) {
    v, _, err := ss.RealmClient.RealmValues.Get(ctx,ss.RealmProject.ID,ss.RealmApp.ID, key)
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


