package atlasclient

import (
	"context"
	"net/url"

	"github.com/mongodb-forks/digest"
	c "github.com/mongodb/atlas-osb/pkg/broker/credentials"
	"github.com/mongodb/atlas-osb/test/cfe2e/config"
	"github.com/mongodb/atlas-osb/test/cfe2e/model/test"
	. "github.com/onsi/gomega" // nolint
	"go.mongodb.org/atlas/mongodbatlas"
)

type Client struct {
	Atlas *mongodbatlas.Client
}

func AClient(keys c.Credential) Client {
	t := digest.NewTransport(keys["publicKey"], keys["privateKey"])
	tc, err := t.Client()
	if err != nil {
		panic(err)
	}

	var client Client
	client.Atlas = mongodbatlas.NewClient(tc)
	u, _ := url.Parse(config.CloudQAHost)
	client.Atlas.BaseURL = u

	return client
}

func (c *Client) GetAzurePrivateEndpointStatus(testFlow test.Test) string {
	projectInfo, _, err := c.Atlas.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())

	pConnection, _, err := c.Atlas.PrivateEndpoints.List(context.Background(), projectInfo.ID, "AZURE", &mongodbatlas.ListOptions{})
	Expect(err).ShouldNot(HaveOccurred())
	return pConnection[0].Status
}

func (c *Client) GetClusterSize(testFlow test.Test) string {
	projectInfo, _, err := c.Atlas.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(projectInfo.ID).ShouldNot(BeEmpty())
	clusterInfo, _, err := c.Atlas.Clusters.Get(context.Background(), projectInfo.ID, testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())
	return clusterInfo.ProviderSettings.InstanceSizeName
}

func (c *Client) GetBackupState(testFlow test.Test) bool {
	projectInfo, _, err := c.Atlas.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())
	clusterInfo, _, err := c.Atlas.Clusters.Get(context.Background(), projectInfo.ID, testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())
	return *clusterInfo.ProviderBackupEnabled
}

// GetDatabaseUsersList
func (c *Client) GetDatabaseUsersList(testFlow test.Test) []mongodbatlas.DatabaseUser {
	projectInfo, _, err := c.Atlas.Projects.GetOneProjectByName(context.Background(), testFlow.ServiceIns)
	Expect(err).ShouldNot(HaveOccurred())
	users, _, err := c.Atlas.DatabaseUsers.List(context.Background(), projectInfo.ID,
		&mongodbatlas.ListOptions{
			PageNum:      0,
			ItemsPerPage: 0,
		},
	)
	Expect(err).ShouldNot(HaveOccurred())
	return users
}
