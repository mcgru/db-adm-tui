package mongo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/mcgru/db-adm-tui/internal/domain"
	"github.com/mcgru/db-adm-tui/internal/ports"
)

var _ ports.StatsReader = (*connection)(nil)

func (c *connection) Statistics(ctx context.Context, ref domain.ObjectRef) (domain.Statistics, error) {
	if ref.Kind != domain.KindCollection {
		return domain.Statistics{}, domain.NewError(domain.KindConfig, "statistics", "this object has no statistics view")
	}
	stats := domain.Statistics{EstimatedTotal: -1, SizeBytes: -1, IndexSizeBytes: -1}
	if total, err := c.client.Database(ref.Database).Collection(ref.Name).EstimatedDocumentCount(ctx); err == nil && total >= 0 {
		stats.EstimatedTotal = total
		stats.EstimateSource = domain.EstimateCollection
	}
	var result struct {
		StorageSize int64            `bson:"storageSize"`
		IndexSizes  map[string]int64 `bson:"indexSizes"`
	}
	if err := c.client.Database(ref.Database).RunCommand(ctx, bson.D{{Key: "collStats", Value: ref.Name}}).Decode(&result); err == nil {
		stats.SizeBytes = result.StorageSize
		if len(result.IndexSizes) > 0 {
			var total int64
			for _, size := range result.IndexSizes {
				total += size
			}
			stats.IndexSizeBytes = total
		}
	}
	return stats, nil
}

func (c *connection) ExactCount(ctx context.Context, ref domain.ObjectRef) (int64, error) {
	if ref.Kind != domain.KindCollection {
		return 0, domain.NewError(domain.KindConfig, "count", "this object has no count view")
	}
	count, err := c.client.Database(ref.Database).Collection(ref.Name).CountDocuments(ctx, bson.D{})
	if err != nil {
		return 0, safeError("count", c.password, err)
	}
	return count, nil
}

// Structure samples the first 100 documents and reports every field path with
// its presence count. Nested documents use dotted paths; arrays stay leaves.
func (c *connection) Structure(ctx context.Context, ref domain.ObjectRef) (domain.Structure, error) {
	if ref.Kind != domain.KindCollection {
		return domain.Structure{}, domain.NewError(domain.KindConfig, "structure", "this object has no structure view")
	}
	cursor, err := c.client.Database(ref.Database).Collection(ref.Name).
		Find(ctx, bson.D{}, options.Find().SetLimit(100))
	if err != nil {
		return domain.Structure{}, safeError("structure", c.password, err)
	}
	defer cursor.Close(ctx)
	var docs []bson.D
	if err := cursor.All(ctx, &docs); err != nil {
		return domain.Structure{}, safeError("structure", c.password, err)
	}
	return sampleStructure(docs), nil
}

func sampleStructure(docs []bson.D) domain.Structure {
	type fieldStat struct {
		kind    string
		present int
	}
	stats := map[string]*fieldStat{}
	for _, doc := range docs {
		seen := map[string]bool{}
		walkMongoFields("", doc, func(path string, value any) {
			if path == "" || seen[path] {
				return
			}
			seen[path] = true
			kind := mongoType(value)
			stat, ok := stats[path]
			if !ok {
				stats[path] = &fieldStat{kind: kind, present: 1}
				return
			}
			stat.present++
			if stat.kind != kind {
				stat.kind = "mixed"
			}
		})
	}
	paths := make([]string, 0, len(stats))
	for path := range stats {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		if paths[i] == "_id" {
			return paths[j] != "_id"
		}
		if paths[j] == "_id" {
			return false
		}
		return paths[i] < paths[j]
	})
	structure := domain.Structure{Sampled: len(docs), Source: domain.StructureSample}
	for _, path := range paths {
		stat := stats[path]
		structure.Fields = append(structure.Fields, domain.StructureField{
			Name:    path,
			Type:    stat.kind,
			Present: stat.present,
			Sampled: len(docs),
		})
	}
	return structure
}

func walkMongoFields(prefix string, value any, emit func(path string, value any)) {
	switch typed := value.(type) {
	case bson.D:
		for _, element := range typed {
			path := element.Key
			if prefix != "" {
				path = prefix + "." + element.Key
			}
			emit(path, element.Value)
			walkMongoFields(path, element.Value, emit)
		}
	case bson.M:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			emit(path, typed[key])
			walkMongoFields(path, typed[key], emit)
		}
	}
}

func mongoType(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bson.Decimal128:
		return "number"
	case bson.ObjectID:
		return "objectId"
	case bson.DateTime:
		return "date"
	case bson.Binary:
		return "binary"
	case bson.D, bson.M:
		return "object"
	case bson.A:
		return "array"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func (c *connection) Indexes(ctx context.Context, ref domain.ObjectRef) ([]domain.Index, error) {
	if ref.Kind != domain.KindCollection {
		return nil, domain.NewError(domain.KindConfig, "indexes", "this object has no index view")
	}
	cursor, err := c.client.Database(ref.Database).Collection(ref.Name).Indexes().List(ctx)
	if err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	defer cursor.Close(ctx)
	var specs []bson.D
	if err := cursor.All(ctx, &specs); err != nil {
		return nil, safeError("indexes", c.password, err)
	}
	sizes := c.indexSizes(ctx, ref)
	out := make([]domain.Index, 0, len(specs))
	for _, spec := range specs {
		index := domain.Index{SizeBytes: -1}
		var keys bson.D
		for _, element := range spec {
			switch element.Key {
			case "name":
				index.Name, _ = element.Value.(string)
			case "key":
				keys, _ = element.Value.(bson.D)
			case "unique":
				index.Unique, _ = element.Value.(bool)
			case "sparse":
				if sparse, ok := element.Value.(bool); ok && sparse {
					index.Stats = appendIndexStat(index.Stats, "sparse")
				}
			case "expireAfterSeconds":
				index.Stats = appendIndexStat(index.Stats, fmt.Sprintf("ttl %vs", element.Value))
			}
		}
		for _, key := range keys {
			index.Columns = append(index.Columns, key.Key)
		}
		index.Method = mongoIndexMethod(keys)
		if size, ok := sizes[index.Name]; ok {
			index.SizeBytes = size
		}
		if index.Stats == "" {
			index.Stats = "unavailable"
		}
		out = append(out, index)
	}
	return out, nil
}

func (c *connection) indexSizes(ctx context.Context, ref domain.ObjectRef) map[string]int64 {
	var result struct {
		IndexSizes map[string]int64 `bson:"indexSizes"`
	}
	err := c.client.Database(ref.Database).RunCommand(ctx, bson.D{{Key: "collStats", Value: ref.Name}}).Decode(&result)
	if err != nil {
		return nil
	}
	return result.IndexSizes
}

func mongoIndexMethod(keys bson.D) string {
	if len(keys) == 0 {
		return "mongo"
	}
	switch value := keys[0].Value.(type) {
	case string:
		return value
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return "btree"
	default:
		return "mongo"
	}
}

func appendIndexStat(existing, next string) string {
	if existing == "" {
		return next
	}
	return strings.Join([]string{existing, next}, " · ")
}
