package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestMongoRowDocument(t *testing.T) {
	document, err := mongoRowDocument(`{"name":"Ada","score":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(document) != 2 || document[0].Key != "name" || document[1].Value != int32(1) {
		t.Fatalf("document = %#v", document)
	}
	if _, err := mongoRowDocument(`[`); err == nil {
		t.Fatal("invalid document accepted")
	}
}

func TestMongoIDValue(t *testing.T) {
	objectID := bson.NewObjectID()
	got := mongoIDValue(domain.RowValue{Column: "_id", Value: objectID.Hex(), Kind: domain.CellText})
	if id, ok := got.(bson.ObjectID); !ok || id != objectID {
		t.Fatalf("id = %#v", got)
	}
	if got := mongoIDValue(domain.RowValue{Column: "_id", Value: "7", Kind: domain.CellNumber}); got != int64(7) {
		t.Fatalf("number = %#v", got)
	}
	if got := mongoIDValue(domain.RowValue{Column: "_id", Value: "plain", Kind: domain.CellText}); got != "plain" {
		t.Fatalf("text = %#v", got)
	}
	when := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	if got := mongoIDValue(domain.RowValue{Column: "_id", Value: when.Format(time.RFC3339Nano), Kind: domain.CellDatetime}); got != when {
		t.Fatalf("datetime = %#v", got)
	}
}

func TestMongoRowIDs(t *testing.T) {
	ids, err := mongoRowIDs([][]domain.RowValue{
		{{Column: "_id", Value: "1", Kind: domain.CellNumber}},
		{{Column: "_id", Value: "2", Kind: domain.CellNumber}},
	})
	if err != nil || len(ids) != 2 || ids[0] != int64(1) || ids[1] != int64(2) {
		t.Fatalf("ids = %#v, %v", ids, err)
	}
	if _, err := mongoRowIDs([][]domain.RowValue{{{Column: "other", Value: "1"}}}); err == nil {
		t.Fatal("missing _id accepted")
	}
	if _, err := mongoRowIDs([][]domain.RowValue{{{Column: "_id", Null: true}}}); err == nil {
		t.Fatal("null _id accepted")
	}
}

func TestMongoUpdateValue(t *testing.T) {
	if got, err := mongoUpdateValue(domain.RowValue{Value: "hello", Kind: domain.CellText}); err != nil || got != "hello" {
		t.Fatalf("text = %#v, %v", got, err)
	}
	if got, err := mongoUpdateValue(domain.RowValue{Value: "42", Kind: domain.CellNumber}); err != nil || got != int64(42) {
		t.Fatalf("number = %#v, %v", got, err)
	}
	if got, err := mongoUpdateValue(domain.RowValue{Value: "true", Kind: domain.CellBool}); err != nil || got != true {
		t.Fatalf("bool = %#v, %v", got, err)
	}
	if got, err := mongoUpdateValue(domain.RowValue{Value: `{"a":1}`, Kind: domain.CellJSON}); err != nil {
		t.Fatalf("json = %#v, %v", got, err)
	}
	if got, err := mongoUpdateValue(domain.RowValue{Null: true}); err != nil || got != nil {
		t.Fatalf("null = %#v, %v", got, err)
	}
	if _, err := mongoUpdateValue(domain.RowValue{Default: true}); err == nil {
		t.Fatal("default accepted")
	}
	if _, err := mongoUpdateValue(domain.RowValue{Value: "x", Kind: domain.CellNumber}); err == nil {
		t.Fatal("bad number accepted")
	}
}
