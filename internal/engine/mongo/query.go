package mongo

import (
	"context"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/mql"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.QueryExecutor = (*connection)(nil)

type mongoQueryCursor struct {
	cursor   *mongo.Cursor
	pending  *bson.D
	closed   bool
	password string
}

type sliceCursor struct {
	grid   domain.Grid
	index  int
	closed bool
}

// Query parses the restricted MQL surface and opens one forward-only cursor.
func (c *connection) Query(ctx context.Context, request domain.QueryRequest) (ports.QueryCursor, error) {
	command, err := mql.Parse(request.Statement)
	if err != nil {
		return nil, domain.NewError(domain.KindGuard, "query", err.Error())
	}
	collection := command.Collection
	if collection == "" {
		collection = request.Ref.Name
	}
	database := request.Database
	if database == "" {
		database = request.Ref.Database
	}
	if collection == "" || database == "" {
		return nil, domain.NewError(domain.KindConfig, "query", "select a collection before running MQL")
	}
	filter, err := mongoDocument(command.Filter)
	if err != nil {
		return nil, domain.NewError(domain.KindConfig, "query", err.Error())
	}
	coll := c.client.Database(database).Collection(collection)

	switch command.Name {
	case mql.Find:
		opts := options.Find()
		if command.Sort != nil {
			sort, err := mongoDocument(command.Sort)
			if err != nil {
				return nil, domain.NewError(domain.KindConfig, "query", err.Error())
			}
			opts.SetSort(sort)
		}
		if command.Projection != nil {
			projection, err := mongoDocument(command.Projection)
			if err != nil {
				return nil, domain.NewError(domain.KindConfig, "query", err.Error())
			}
			opts.SetProjection(projection)
		}
		if command.Skip > 0 {
			opts.SetSkip(command.Skip)
		}
		if command.Limit > 0 {
			opts.SetLimit(command.Limit)
		}
		cursor, err := coll.Find(ctx, filter, opts)
		if err != nil {
			return nil, safeError("query", c.password, err)
		}
		return &mongoQueryCursor{cursor: cursor, password: c.password}, nil
	case mql.Aggregate:
		pipeline := bson.A{}
		for _, stage := range command.Pipeline {
			value, err := mongoValue(stage)
			if err != nil {
				return nil, domain.NewError(domain.KindConfig, "query", err.Error())
			}
			document, ok := value.(bson.D)
			if !ok {
				return nil, domain.NewError(domain.KindConfig, "query", "aggregate stages must be objects")
			}
			pipeline = append(pipeline, document)
		}
		cursor, err := coll.Aggregate(ctx, pipeline)
		if err != nil {
			return nil, safeError("query", c.password, err)
		}
		return &mongoQueryCursor{cursor: cursor, password: c.password}, nil
	case mql.Count:
		count, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			return nil, safeError("query", c.password, err)
		}
		grid := domain.Grid{
			Columns: []domain.Column{{Name: "count"}},
			Rows:    []domain.Row{{Cells: []domain.Cell{{Text: strconv.FormatInt(count, 10), Kind: domain.CellNumber}}}},
		}
		return &sliceCursor{grid: grid}, nil
	case mql.Distinct:
		var values []any
		if err := coll.Distinct(ctx, command.Field, filter).Decode(&values); err != nil {
			return nil, safeError("query", c.password, err)
		}
		grid := domain.Grid{Columns: []domain.Column{{Name: command.Field}}}
		for _, value := range values {
			grid.Rows = append(grid.Rows, domain.Row{Cells: []domain.Cell{mongoCell(value)}})
		}
		return &sliceCursor{grid: grid}, nil
	default:
		return nil, domain.NewError(domain.KindGuard, "query", "unsupported MQL command")
	}
}

func (q *mongoQueryCursor) Next(ctx context.Context, limit int) (domain.QueryPage, error) {
	if q == nil || q.cursor == nil || q.closed {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result cursor is closed")
	}
	if limit <= 0 {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result window is not valid")
	}
	docs := make([]bson.D, 0, limit)
	for len(docs) < limit {
		doc, ok, err := q.next(ctx)
		if err != nil {
			return domain.QueryPage{}, safeError("query", q.password, err)
		}
		if !ok {
			return domain.QueryPage{Grid: mongoDocumentsGrid(docs)}, nil
		}
		docs = append(docs, doc)
	}
	doc, ok, err := q.next(ctx)
	if err != nil {
		return domain.QueryPage{}, safeError("query", q.password, err)
	}
	if ok {
		q.pending = &doc
	}
	return domain.QueryPage{Grid: mongoDocumentsGrid(docs), HasMore: ok}, nil
}

func (q *mongoQueryCursor) next(ctx context.Context) (bson.D, bool, error) {
	if q.pending != nil {
		doc := *q.pending
		q.pending = nil
		return doc, true, nil
	}
	if !q.cursor.Next(ctx) {
		return nil, false, q.cursor.Err()
	}
	var doc bson.D
	if err := q.cursor.Decode(&doc); err != nil {
		return nil, false, err
	}
	return doc, true, nil
}

func (q *mongoQueryCursor) Close(ctx context.Context) error {
	if q == nil || q.closed {
		return nil
	}
	q.closed = true
	if q.cursor == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return q.cursor.Close(ctx)
}

func (q *sliceCursor) Next(ctx context.Context, limit int) (domain.QueryPage, error) {
	if q == nil || q.closed {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result cursor is closed")
	}
	if limit <= 0 {
		return domain.QueryPage{}, domain.NewError(domain.KindConfig, "query", "the result window is not valid")
	}
	start := q.index
	end := start + limit
	if end > len(q.grid.Rows) {
		end = len(q.grid.Rows)
	}
	page := domain.QueryPage{
		Grid:    domain.Grid{Columns: q.grid.Columns, Rows: q.grid.Rows[start:end]},
		HasMore: end < len(q.grid.Rows),
	}
	q.index = end
	return page, nil
}

func (q *sliceCursor) Close(ctx context.Context) error {
	if q == nil {
		return nil
	}
	q.closed = true
	return nil
}

func mongoDocumentsGrid(docs []bson.D) domain.Grid {
	grid, _ := buildMongoGrid(docs, len(docs))
	return grid
}

func mongoDocument(value any) (bson.D, error) {
	if value == nil {
		return bson.D{}, nil
	}
	object, ok := value.(mql.Object)
	if !ok {
		return nil, errUnsupportedValue("the argument must be an object")
	}
	document := make(bson.D, 0, len(object))
	for _, pair := range object {
		converted, err := mongoValue(pair.Value)
		if err != nil {
			return nil, err
		}
		document = append(document, bson.E{Key: pair.Key, Value: converted})
	}
	return document, nil
}

func mongoValue(value any) (any, error) {
	switch v := value.(type) {
	case nil, string, bool, int64, float64:
		return v, nil
	case mql.Object:
		return mongoDocument(v)
	case mql.Array:
		array := make(bson.A, 0, len(v))
		for _, item := range v {
			converted, err := mongoValue(item)
			if err != nil {
				return nil, err
			}
			array = append(array, converted)
		}
		return array, nil
	case []any:
		array := make(bson.A, 0, len(v))
		for _, item := range v {
			converted, err := mongoValue(item)
			if err != nil {
				return nil, err
			}
			array = append(array, converted)
		}
		return array, nil
	case mql.ObjectID:
		id, err := bson.ObjectIDFromHex(string(v))
		if err != nil {
			return nil, errUnsupportedValue("ObjectId expects 24 hex characters")
		}
		return id, nil
	case mql.Date:
		return bson.NewDateTimeFromTime(v.Time()), nil
	default:
		return nil, errUnsupportedValue("unsupported MQL value")
	}
}

type errUnsupportedValue string

func (e errUnsupportedValue) Error() string { return string(e) }
