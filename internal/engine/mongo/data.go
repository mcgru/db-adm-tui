package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.DataReader = (*connection)(nil)

// Page implements ports.DataReader with an _id-sorted find and offset paging.
// Each document decodes as an ordered bson.D so the first page establishes
// _id first and first-seen field order; the TUI unions later pages.
func (c *connection) Page(ctx context.Context, query domain.DataQuery) (domain.DataPage, error) {
	if query.Ref.Kind != domain.KindCollection {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "this object has no data view")
	}
	if query.Offset < 0 || query.Limit <= 0 {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "the page window is not valid")
	}
	if query.Ref.Database == "" || query.Ref.Name == "" {
		return domain.DataPage{}, domain.NewError(domain.KindConfig, "data", "the collection is not selected")
	}
	filter, err := buildMongoFilter(query.Filter)
	if err != nil {
		return domain.DataPage{}, err
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "_id", Value: 1}}).
		SetSkip(int64(query.Offset)).
		SetLimit(int64(query.Limit + 1))
	cursor, err := c.client.Database(query.Ref.Database).Collection(query.Ref.Name).Find(ctx, filter, opts)
	if err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}
	defer cursor.Close(ctx)

	var docs []bson.D
	if err := cursor.All(ctx, &docs); err != nil {
		return domain.DataPage{}, safeError("data", c.password, err)
	}
	grid, hasMore := buildMongoGrid(docs, query.Limit)
	total, source := c.estimate(ctx, query.Ref)
	return domain.DataPage{
		Grid:           grid,
		Offset:         query.Offset,
		HasMore:        hasMore,
		EstimatedTotal: total,
		EstimateSource: source,
		OrderKey:       "_id",
		OrderKeyKnown:  true,
	}, nil
}

func (c *connection) estimate(ctx context.Context, ref domain.ObjectRef) (int64, string) {
	total, err := c.client.Database(ref.Database).Collection(ref.Name).EstimatedDocumentCount(ctx)
	if err != nil || total < 0 {
		return -1, ""
	}
	return total, domain.EstimateCollection
}

// buildMongoFilter maps one DataFilter to the native regexp query. Field is a
// dotted path, so a nested field filters without touching the document shape.
// The empty filter keeps the _id-sorted page unchanged.
func buildMongoFilter(filter *domain.DataFilter) (bson.D, error) {
	if filter == nil {
		return bson.D{}, nil
	}
	if !domain.ValidMongoPath(filter.Field) {
		return nil, domain.NewError(domain.KindConfig, "data", "the filter field is not valid")
	}
	regex := bson.D{{Key: "$regex", Value: filter.Pattern}}
	if filter.CaseInsensitive {
		regex = append(regex, bson.E{Key: "$options", Value: "i"})
	}
	return bson.D{{Key: filter.Field, Value: regex}}, nil
}

// buildMongoGrid maps ordered documents to a grid with the union of top-level
// fields: _id first, then fields in first-seen order. Missing fields stay empty,
// while explicit nulls remain null cells.
func buildMongoGrid(docs []bson.D, limit int) (domain.Grid, bool) {
	hasMore := len(docs) > limit
	if hasMore {
		docs = docs[:limit]
	}

	columns := mongoColumns(docs)
	grid := domain.Grid{Columns: make([]domain.Column, len(columns))}
	for i, name := range columns {
		grid.Columns[i] = domain.Column{Name: name}
	}
	for _, doc := range docs {
		values := make(map[string]any, len(doc))
		for _, element := range doc {
			values[element.Key] = element.Value
		}
		cells := make([]domain.Cell, len(columns))
		for i, name := range columns {
			value, ok := values[name]
			if !ok {
				cells[i] = domain.Cell{Text: "", Kind: domain.CellText}
				continue
			}
			cells[i] = mongoCell(value)
		}
		grid.Rows = append(grid.Rows, domain.Row{Cells: cells})
	}
	return grid, hasMore
}

func mongoColumns(docs []bson.D) []string {
	found := map[string]bool{}
	hasID := false
	for _, doc := range docs {
		for _, element := range doc {
			if element.Key == "_id" {
				hasID = true
				break
			}
		}
		if hasID {
			break
		}
	}
	var columns []string
	if hasID {
		columns = append(columns, "_id")
		found["_id"] = true
	}
	for _, doc := range docs {
		for _, element := range doc {
			if found[element.Key] {
				continue
			}
			found[element.Key] = true
			columns = append(columns, element.Key)
		}
	}
	return columns
}

func mongoCell(value any) domain.Cell {
	switch v := value.(type) {
	case nil:
		return domain.NullCell()
	case string:
		return domain.Cell{Text: v, Kind: domain.CellText}
	case bool:
		return domain.Cell{Text: fmt.Sprint(v), Kind: domain.CellBool}
	case int32, int64, float64, bson.Decimal128:
		return domain.Cell{Text: fmt.Sprint(v), Kind: domain.CellNumber}
	case bson.ObjectID:
		return domain.Cell{Text: v.Hex(), Kind: domain.CellText}
	case bson.DateTime:
		return domain.Cell{Text: v.Time().UTC().Format(time.RFC3339Nano), Kind: domain.CellDatetime}
	case bson.Binary:
		return domain.Cell{Text: fmt.Sprintf("<binary %d bytes>", len(v.Data)), Kind: domain.CellBinary}
	case bson.D, bson.A, bson.M, bson.Raw:
		return mongoJSON(v)
	default:
		return mongoJSON(v)
	}
}

func mongoJSON(value any) domain.Cell {
	data, err := bson.MarshalExtJSON(value, false, false)
	if err != nil {
		return domain.Cell{Text: fmt.Sprint(value), Kind: domain.CellJSON}
	}
	return domain.Cell{Text: string(data), Kind: domain.CellJSON}
}
