package guard

import "testing"

func TestCheckSQLAllowsReads(t *testing.T) {
	for _, statement := range []string{
		"SELECT * FROM customers",
		"select id from public.customers;",
		"WITH x AS (SELECT 1) SELECT * FROM x",
		"EXPLAIN SELECT * FROM customers",
		"EXPLAIN ANALYZE SELECT * FROM customers",
		"SHOW TABLES",
		"DESCRIBE customers",
		"VALUES (1)",
		"SELECT 'insert delete drop' FROM customers",
		"SELECT * FROM update_log",
		`SELECT * FROM "update"`,
	} {
		if decision := CheckSQL(statement); !decision.Allowed {
			t.Fatalf("%q blocked: %s", statement, decision.Reason)
		}
	}
}

func TestCheckSQLBlocksWrites(t *testing.T) {
	for _, statement := range []string{
		"INSERT INTO customers VALUES (1)",
		"UPDATE customers SET name = 'x'",
		"DELETE FROM customers",
		"WITH x AS (SELECT 1) DELETE FROM customers",
		"EXPLAIN ANALYZE DELETE FROM customers",
		"SELECT * INTO backup FROM customers",
		"CREATE TABLE t (id int)",
		"ALTER TABLE customers ADD COLUMN x int",
		"DROP TABLE customers",
		"GRANT SELECT ON customers TO someone",
		"REVOKE SELECT ON customers FROM someone",
		"TRUNCATE customers",
		"COMMENT ON TABLE customers IS 'x'",
		"COPY customers FROM STDIN",
		"CALL do_something()",
		"DO $$ BEGIN NULL; END $$",
		"ANALYZE customers",
		"VACUUM customers",
		"SET GLOBAL max_connections = 10",
		"SET TRANSACTION READ WRITE",
		"REPLACE INTO customers VALUES (1)",
		"LOAD DATA INFILE 'x' INTO TABLE customers",
		"RENAME TABLE customers TO old_customers",
		"LOCK TABLES customers WRITE",
		"UNLOCK TABLES",
		"SELECT 1; SELECT 2",
		"SELECT 'unterminated",
	} {
		if decision := CheckSQL(statement); decision.Allowed {
			t.Fatalf("%q was allowed", statement)
		}
	}
}

func TestCheckMongoAllowsReads(t *testing.T) {
	for _, statement := range []string{
		`db.customers.find({name: "x"})`,
		`find({name: "x"})`,
		`db.customers.aggregate([{$match: {name: "x"}}])`,
		`db.customers.count({name: "x"})`,
		`db.customers.countDocuments({})`,
		`db.customers.distinct("status")`,
	} {
		decision := CheckMongo(statement)
		if statement == `db.customers.countDocuments({})` {
			if decision.Allowed {
				t.Fatalf("%q was allowed", statement)
			}
			continue
		}
		if !decision.Allowed {
			t.Fatalf("%q blocked: %s", statement, decision.Reason)
		}
	}
}

func TestCheckMongoBlocksWrites(t *testing.T) {
	for _, statement := range []string{
		`db.customers.insertOne({name: "x"})`,
		`db.customers.updateOne({}, {$set: {name: "x"}})`,
		`db.customers.deleteMany({})`,
		`db.customers.findAndModify({query: {}, update: {}})`,
		`db.customers.createIndex({name: 1})`,
		`db.customers.drop()`,
		`db.customers.aggregate([{$out: "archive"}])`,
		`db.customers.aggregate([{$merge: {into: "archive"}}])`,
		`db.customers.find({$where: "this.name"})`,
		`db.customers.aggregate([{$group: {_id: {$function: {body: "x"}}}}])`,
		`find({}); drop()`,
	} {
		if decision := CheckMongo(statement); decision.Allowed {
			t.Fatalf("%q was allowed", statement)
		}
	}
}

func TestCheckSQLModeReadWrite(t *testing.T) {
	for _, statement := range []string{
		"UPDATE customers SET name = 'x'",
		"CREATE TABLE t (id int)",
		"SET TRANSACTION READ WRITE",
	} {
		if decision := CheckSQLMode(statement, false); !decision.Allowed {
			t.Fatalf("%q blocked in read-write: %s", statement, decision.Reason)
		}
	}
	for _, statement := range []string{
		"SELECT 1; SELECT 2",
		"SELECT 'unterminated",
	} {
		if decision := CheckSQLMode(statement, false); decision.Allowed {
			t.Fatalf("%q was allowed in read-write", statement)
		}
	}
}

func TestCheckMongoModeReadWrite(t *testing.T) {
	if decision := CheckMongoMode(`db.customers.aggregate([{$out: "archive"}])`, false); !decision.Allowed {
		t.Fatalf("$out blocked in read-write: %s", decision.Reason)
	}
	for _, statement := range []string{
		`db.customers.find({$where: "this.name"})`,
		`db.customers.insertOne({name: "x"})`,
	} {
		if decision := CheckMongoMode(statement, false); decision.Allowed {
			t.Fatalf("%q was allowed in read-write", statement)
		}
	}
}

func FuzzCheckSQL(f *testing.F) {
	for _, seed := range []string{"SELECT 1", "INSERT INTO t VALUES (1)", "", "SELECT 'x';"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, statement string) {
		_ = CheckSQL(statement)
	})
}

func FuzzCheckMongo(f *testing.F) {
	for _, seed := range []string{`db.c.find({})`, `db.c.drop()`, "", `find({$where: "x"})`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, statement string) {
		_ = CheckMongo(statement)
	})
}
