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
