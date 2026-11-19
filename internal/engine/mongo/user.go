package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.UserAdmin = (*connection)(nil)

func (c *connection) CreateUser(ctx context.Context, spec domain.UserSpec) (domain.MutationResult, error) {
	database := strings.TrimSpace(spec.Database)
	name := strings.TrimSpace(spec.Name)
	if database == "" || name == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the MongoDB user is not database-scoped")
	}
	if spec.Password == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password is empty")
	}
	if strings.ContainsAny(spec.Password, "\n\r") {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "user", "the password must not contain line breaks")
	}
	statement := fmt.Sprintf("db.getSiblingDB(%q).createUser({user:%q, pwd:\"***\", roles:[]})", database, name)
	err := c.client.Database(database).RunCommand(ctx, bson.D{
		{Key: "createUser", Value: name},
		{Key: "pwd", Value: spec.Password},
		{Key: "roles", Value: bson.A{}},
	}).Err()
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("user", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func (c *connection) DropUser(ctx context.Context, user domain.RoleRef) (domain.MutationResult, error) {
	database, name, err := mongoUserRef(user)
	if err != nil {
		return domain.MutationResult{}, err
	}
	statement := fmt.Sprintf("db.getSiblingDB(%q).dropUser(%q)", database, name)
	if err := c.client.Database(database).RunCommand(ctx, bson.D{{Key: "dropUser", Value: name}}).Err(); err != nil {
		return domain.MutationResult{Statement: statement}, safeError("user", c.password, err)
	}
	return domain.MutationResult{Statement: statement}, nil
}

func mongoUserRef(user domain.RoleRef) (string, string, error) {
	name := strings.TrimSpace(user.Name)
	database := strings.TrimSpace(user.Database)
	if index := strings.LastIndex(name, "@"); index >= 0 {
		if database == "" {
			database = name[index+1:]
		}
		name = name[:index]
	}
	if database == "" || name == "" {
		return "", "", domain.NewError(domain.KindConfig, "user", "the MongoDB user is not database-scoped")
	}
	return database, name, nil
}
