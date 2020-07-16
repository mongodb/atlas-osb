package statestorage 

import (
	"context"
	"fmt"
	"net/http"
    "net/url"
	"github.com/mongodb/mongodb-atlas-service-broker/pkg/broker/credentials"
	"github.com/Sectorbob/mlab-ns2/gae/ns/digest"
	"github.com/mongodb/go-client-mongodb-atlas/mongodbatlas"
	"go.uber.org/zap"
)


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
    fmt.Sprintf("atlas-osb-%s",okey)
    project, r, err := client.Projects.GetOneProjectByName(context.Background(), projectName)
	if err != nil && (r.StatusCode == http.StatusNotFound || r.StatusCode ==http.StatusUnauthorized) {
		logger.Infow("statestorage project not found, attempt create", "error", err, "projectName", projectName)
		project = &mongodbatlas.Project{}
        project.Name = projectName
        project.OrgID = creds.Orgs[okey].Roles[0].OrgID
        project, r, err = client.Projects.Create(context.Background(), project)
		logger.Infow("statestorage project created","project", project)
	}
    if err != nil {
        return nil, "", err
    }

    clusterName := "statestorage"
    cluster, r, err := client.Clusters.Get(context.Background(), project.ID, clusterName)
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
	    cluster, r, err = client.Clusters.Create(context.Background(), project.ID, cluster)
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
        err = nil 
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


