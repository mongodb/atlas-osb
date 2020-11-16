package broker

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mongodb/atlas-osb/pkg/broker/dynamicplans"
	"github.com/pkg/errors"
	"go.mongodb.org/atlas/mongodbatlas"
)

const (
	overrideAtlasUserRoles = "overrideAtlasUserRoles"
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

	firstName, ok := planContext["firstName"].(string)
	if !ok {
		firstName = "Unnamed"
	}

	lastName, ok := planContext["lastName"].(string)
	if !ok {
		lastName = "Unnamed"
	}

	country, ok := planContext["country"].(string)
	if !ok {
		country = "US"
	}

	roleNames, ok := p.Settings[overrideAtlasUserRoles].([]interface{})
	if !ok {
		roleNames = []interface{}{"GROUP_READ_ONLY"}
	}

	roles := make([]mongodbatlas.AtlasRole, 0, len(roleNames))
	for _, r := range roleNames {
		role, ok := r.(string)
		if !ok {
			return fmt.Errorf("role name must be a string, got %v (%T)", r, r)
		}

		roles = append(roles, mongodbatlas.AtlasRole{
			GroupID:  p.Project.ID,
			RoleName: role,
		})
	}

	u := &mongodbatlas.AtlasUser{
		EmailAddress: email,
		Password:     password,
		Country:      country,
		Username:     email,
		FirstName:    firstName,
		LastName:     lastName,
		Roles:        roles,
	}

	_, r, err := client.AtlasUsers.Create(ctx, u)
	if err != nil && r.StatusCode != http.StatusConflict {
		return errors.Wrap(err, "cannot create Atlas user")
	}

	// user successfully invited
	if err == nil {
		return nil
	}

	// 409 Conflict: user already exists in the system, need to add them to the project
	u, _, err = client.AtlasUsers.GetByName(ctx, email)
	if err != nil {
		return errors.Wrap(err, "cannot get Atlas user by name")
	}

	_, _, err = client.AtlasUsers.Update(ctx, u.ID, roles)

	return errors.Wrap(err, "cannot update Atlas user")
}

func (b *Broker) removeUserFromProject(ctx context.Context, client *mongodbatlas.Client, planContext dynamicplans.Context, p *dynamicplans.Plan) error {
	email, ok := planContext["email"].(string)
	if !ok {
		return fmt.Errorf("email should be string, got %T = %v", planContext["email"], planContext["email"])
	}

	u, _, err := client.AtlasUsers.GetByName(ctx, email)
	if err != nil {
		return errors.Wrap(err, "cannot get Atlas user by name")
	}

	_, err = client.Projects.RemoveUserFromProject(ctx, p.Project.ID, u.ID)

	return errors.Wrap(err, "cannot remove Atlas user from Project")
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
