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
	if err := validMongoRowRef(spec.Ref); err != nil {
		return domain.MutationResult{}, err
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

func (c *connection) UpdateRows(ctx context.Context, spec domain.RowUpdate) (domain.MutationResult, error) {
	if err := validMongoRowRef(spec.Ref); err != nil {
		return domain.MutationResult{}, err
	}
	column := strings.TrimSpace(spec.Column)
	if column == "" || !domain.ValidMongoPath(column) || column == "_id" || strings.HasPrefix(column, "_id.") {
		return domain.MutationResult{}, domain.NewError(domain.KindConfig, "row", "the update field is not valid")
	}
	value, err := mongoUpdateValue(spec.Value)
	if err != nil {
		return domain.MutationResult{}, err
	}
	ids, err := mongoRowIDs(spec.Rows)
	if err != nil {
		return domain.MutationResult{}, err
	}
	filter := mongoIDFilter(ids)
	update := bson.D{{Key: "$set", Value: bson.D{{Key: column, Value: value}}}}
	collection := c.client.Database(spec.Ref.Database).Collection(spec.Ref.Name)
	if len(ids) == 1 {
		statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).updateOne({_id:<key>}, {$set:{%q:<value>}})",
			spec.Ref.Database, spec.Ref.Name, column)
		result, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
		}
		return domain.MutationResult{Statement: statement, Affected: result.ModifiedCount}, nil
	}
	statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).updateMany({_id:{$in:<keys>}}, {$set:{%q:<value>}})",
		spec.Ref.Database, spec.Ref.Name, column)
	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	return domain.MutationResult{Statement: statement, Affected: result.ModifiedCount}, nil
}

func (c *connection) DeleteRows(ctx context.Context, spec domain.RowDelete) (domain.MutationResult, error) {
	if err := validMongoRowRef(spec.Ref); err != nil {
		return domain.MutationResult{}, err
	}
	ids, err := mongoRowIDs(spec.Rows)
	if err != nil {
		return domain.MutationResult{}, err
	}
	filter := mongoIDFilter(ids)
	collection := c.client.Database(spec.Ref.Database).Collection(spec.Ref.Name)
	if len(ids) == 1 {
		statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).deleteOne({_id:<key>})",
			spec.Ref.Database, spec.Ref.Name)
		result, err := collection.DeleteOne(ctx, filter)
		if err != nil {
			return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
		}
		return domain.MutationResult{Statement: statement, Affected: result.DeletedCount}, nil
	}
	statement := fmt.Sprintf("db.getSiblingDB(%q).getCollection(%q).deleteMany({_id:{$in:<keys>}})",
		spec.Ref.Database, spec.Ref.Name)
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		return domain.MutationResult{Statement: statement}, safeError("row", c.password, err)
	}
	return domain.MutationResult{Statement: statement, Affected: result.DeletedCount}, nil
}

func validMongoRowRef(ref domain.ObjectRef) error {
	if ref.Kind != domain.KindCollection || ref.Database == "" || ref.Name == "" {
		return domain.NewError(domain.KindConfig, "row", "the collection is not selected")
	}
	return nil
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

// mongoRowIDs extracts the _id of every selected row/document. The TUI stores
// one key per row, so extra or missing key columns indicate a stale selection.
func mongoRowIDs(rows [][]domain.RowValue) ([]any, error) {
	if len(rows) == 0 {
		return nil, domain.NewError(domain.KindConfig, "row", "the document has no _id")
	}
	ids := make([]any, 0, len(rows))
	for _, row := range rows {
		if len(row) != 1 || row[0].Column != "_id" {
			return nil, domain.NewError(domain.KindConfig, "row", "the document has no _id")
		}
		if row[0].Null {
			return nil, domain.NewError(domain.KindConfig, "row", "the _id is empty")
		}
		if row[0].Kind == domain.CellBinary {
			return nil, domain.NewError(domain.KindConfig, "row", "binary _id values cannot be used for writes")
		}
		ids = append(ids, mongoIDValue(row[0]))
	}
	return ids, nil
}

func mongoIDFilter(ids []any) bson.D {
	if len(ids) == 1 {
		return bson.D{{Key: "_id", Value: ids[0]}}
	}
	return bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}}
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

// mongoUpdateValue parses the edited cell using the source cell kind. Text
// stays a string; JSON is parsed as extended JSON; numeric, bool, and datetime
// cells require a matching value.
func mongoUpdateValue(value domain.RowValue) (any, error) {
	if value.Default {
		return nil, domain.NewError(domain.KindConfig, "row", "MongoDB has no DEFAULT value")
	}
	if value.Null {
		return nil, nil
	}
	switch value.Kind {
	case domain.CellJSON:
		var parsed any
		if err := bson.UnmarshalExtJSON([]byte(value.Value), false, &parsed); err != nil {
			return nil, domain.NewError(domain.KindConfig, "row", "the value is not valid extended JSON")
		}
		return parsed, nil
	case domain.CellNumber:
		if n, err := strconv.ParseInt(value.Value, 10, 64); err == nil {
			return n, nil
		}
		if n, err := strconv.ParseFloat(value.Value, 64); err == nil {
			return n, nil
		}
		return nil, domain.NewError(domain.KindConfig, "row", "the value is not a number")
	case domain.CellBool:
		parsed, err := strconv.ParseBool(value.Value)
		if err != nil {
			return nil, domain.NewError(domain.KindConfig, "row", "the value is not a boolean")
		}
		return parsed, nil
	case domain.CellDatetime:
		parsed, err := time.Parse(time.RFC3339Nano, value.Value)
		if err != nil {
			return nil, domain.NewError(domain.KindConfig, "row", "the value is not an RFC3339 timestamp")
		}
		return parsed, nil
	case domain.CellBinary:
		return nil, domain.NewError(domain.KindConfig, "row", "binary values cannot be edited")
	default:
		return value.Value, nil
	}
}
