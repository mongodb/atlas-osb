package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"go.mongodb.org/atlas/mongodbatlas"
)

func (b *Broker) addUserToProject(ctx context.Context, client *mongodbatlas.Client, planContext dynamicplans.Context, p *dynamicplans.Plan) error {
	email, ok := planContext["email"].(string)
	if !ok {
		return fmt.Errorf("email should be string, got %T = %v", planContext["email"], planContext["email"])
	}

	password, ok := planContext["password"].(string)
	if !ok {
		var err error
		password, err = generatePassword()
		if err != nil {
			return err
		}
	}

	role := p.Settings[overrideAtlasUserRole]
	if role == "" {
		role = "GROUP_READ_ONLY"
	}

	u := &mongodbatlas.AtlasUser{
		EmailAddress: email,
		Password:     password,
		Country:      "US",
		Username:     email,
		Roles: []mongodbatlas.AtlasRole{
			{
				GroupID:  p.Project.ID,
				RoleName: role,
			},
		},
	}

	_, r, err := client.AtlasUsers.Create(ctx, u)
	if err != nil && r.StatusCode != http.StatusConflict {
		return err
	}

	// TODO: add existing users here
	u, _, err = client.AtlasUsers.GetByName(ctx, email)
	if err != nil {
		return err
	}

	client.AtlasUsers.Update(ctx, u.ID, u)
	return err
}

func (b *Broker) removeUserFromProject(ctx context.Context, client *mongodbatlas.Client, planContext dynamicplans.Context, p *dynamicplans.Plan) error {
	email, ok := planContext["email"].(string)
	if !ok {
		return fmt.Errorf("email should be string, got %T = %v", planContext["email"], planContext["email"])
	}

	u, _, err := client.AtlasUsers.GetByName(ctx, email)
	if err != nil {
		return err
	}

	_, err = client.Projects.RemoveUserFromProject(ctx, p.Project.ID, u.ID)
	return err
}

func (b *Broker) performOperation(ctx context.Context, client *mongodbatlas.Client, planContext dynamicplans.Context, p *dynamicplans.Plan, op string) error {
	switch op {
	case "AddUserToProject":
		return b.addUserToProject(ctx, client, planContext, p)

	case "RemoveUserFromProject":
		return b.removeUserFromProject(ctx, client, planContext, p)

	default:
		return fmt.Errorf("unknown operation %q", op)
	}
}
