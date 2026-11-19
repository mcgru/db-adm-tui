package mongo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.RowAdmin = (*connection)(nil)

func (c *connection) InsertRow(ctx context.Context, spec domain.RowInsert) (domain.MutationResult, error) {
	if spec.Ref.Kind != domain.KindCollection || spec.Ref.Database == "" || spec.Ref.Name == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "row", "the collection is not selected")
	}
	document, err := mongoRowDocument(spec.Document)
	if err != nil {
		return domain.MutationResult{}, err
	}
	statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).insertOne(<document>)",
		spec.Ref.Database, spec.Ref.Name)
	if _, err := c.client.Database(spec.Ref.Database).Collection(spec.Ref.Name).InsertOne(ctx, document); err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	return domain.MutationResult{Statement: statement, Affected: 1}, nil
}

func (c *connection) DeleteRow(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error) {
	if spec.Ref.Kind != domain.KindCollection || spec.Ref.Database == "" || spec.Ref.Name == "" {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "row", "the collection is not selected")
	}
	var id domain.RowValue
	found := false
	for _, key := range spec.Keys {
		if key.Column == "_id" {
			id = key
			found = true
			break
		}
	}
	if !found {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "row", "the document has no _id")
	}
	filter := bson.D{{Key: "_id", Value: mongoIDValue(id)}}
	statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).deleteOne({_id:<key>})",
		spec.Ref.Database, spec.Ref.Name)
	result, err := c.client.Database(spec.Ref.Database).Collection(spec.Ref.Name).DeleteOne(ctx, filter)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	return domain.MutationResult{Statement: statement, Affected: result.DeletedCount}, nil
}

func mongoRowDocument(raw string) (bson.D, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, domain.NewError(domain.KindConfig, "row", "the document is empty")
	}
	var document bson.D
	if err := bson.UnmarshalExtJSON([]byte(raw), false, &document); err != nil {
		return nil, domain.NewError(domain.KindConfig, "row", "the document is not valid extended JSON")
	}
	return document, nil
}

func mongoIDValue(value domain.RowValue) any {
	if value.Null {
		return nil
	}
	switch value.Kind {
	case domain.CellJSON:
		var parsed any
		if err := bson.UnmarshalExtJSON([]byte(value.Value), false, &parsed); err == nil {
			return parsed
		}
	case domain.CellNumber:
		if n, err := strconv.ParseInt(value.Value, 10, 64); err == nil {
			return n
		}
		if n, err := strconv.ParseFloat(value.Value, 64); err == nil {
			return n
		}
	case domain.CellBool:
		if b, err := strconv.ParseBool(value.Value); err == nil {
			return b
		}
	case domain.CellDatetime:
		if parsed, err := time.Parse(time.RFC3339Nano, value.Value); err == nil {
			return parsed
		}
	}
	if objectID, err := bson.ObjectIDFromHex(value.Value); err == nil {
		return objectID
	}
	return value.Value
}
