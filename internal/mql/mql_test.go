package mql

import "testing"

func TestParseFind(t *testing.T) {
	command, err := Parse(`db.customers.find({ status: 'active', age: {$gte: 18} }).sort({name: 1}).skip(10).limit(25);`)
	if err != nil {
		t.Fatal(err)
	}
	if command.Name != Find || command.Collection != "customers" {
		t.Fatalf("command = %+v", command)
	}
	if command.Skip != 10 || command.Limit != 25 {
		t.Fatalf("skip=%d limit=%d", command.Skip, command.Limit)
	}
	filter, ok := command.Filter.(Object)
	if !ok || len(filter) != 2 || filter[0].Key != "status" || filter[0].Value != "active" {
		t.Fatalf("filter = %#v", command.Filter)
	}
}

func TestParseFindWithoutPrefix(t *testing.T) {
	command, err := Parse(`find({})`)
	if err != nil {
		t.Fatal(err)
	}
	if command.Name != Find || command.Collection != "" {
		t.Fatalf("command = %+v", command)
	}
}

func TestParseAggregate(t *testing.T) {
	command, err := Parse(`aggregate([{$match: {status: "active"}}, {$group: {_id: "$status", n: {$sum: 1}}}])`)
	if err != nil {
		t.Fatal(err)
	}
	if command.Name != Aggregate || len(command.Pipeline) != 2 {
		t.Fatalf("command = %+v", command)
	}
}

func TestParseCountAndDistinct(t *testing.T) {
	count, err := Parse(`count({status: "open"})`)
	if err != nil || count.Name != Count {
		t.Fatalf("count = %+v, %v", count, err)
	}
	distinct, err := Parse(`distinct("status", {region: "eu"})`)
	if err != nil || distinct.Name != Distinct || distinct.Field != "status" {
		t.Fatalf("distinct = %+v, %v", distinct, err)
	}
}

func TestParseLiterals(t *testing.T) {
	command, err := Parse(`find({_id: ObjectId("507f1f77bcf86cd799439011"), at: ISODate("2026-11-18T12:00:00Z")})`)
	if err != nil {
		t.Fatal(err)
	}
	filter := command.Filter.(Object)
	if _, ok := filter[0].Value.(ObjectID); !ok {
		t.Fatalf("ObjectId = %#v", filter[0].Value)
	}
	if _, ok := filter[1].Value.(Date); !ok {
		t.Fatalf("ISODate = %#v", filter[1].Value)
	}
}

func TestParseRejectsUnsafeInput(t *testing.T) {
	for _, statement := range []string{
		`db.customers.insertOne({name: "x"})`,
		`db.customers.find({$where: "this.name"})`,
		`db.customers.find({}); db.customers.drop()`,
		`db.customers.find({}); drop()`,
		`find({}).map(function (x) { return x; })`,
		`eval("1+1")`,
		`db.customers.find({}, {}, {})`,
		`db.customers.find({name: /x/})`,
		`aggregate("pipeline")`,
		``,
	} {
		if _, err := Parse(statement); err == nil {
			t.Fatalf("%q parsed", statement)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, seed := range []string{`db.c.find({})`, `insertOne({})`, ``, `find({$where: "x"})`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, statement string) {
		_, _ = Parse(statement)
	})
}
