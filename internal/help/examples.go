package help

// examples is the extended, example-first help shown when Enter is pressed in
// the short context-help modal. Keys are lowercase terms from the completion
// and UI vocabulary.
var examples = map[string]string{
	"find": `Basic filter:
db.customers.find({ status: "active" })

$and with two conditions:
db.customers.find({
  $and: [
    { status: "active" },
    { age: { $gte: 18 } }
  ]
})

Projection: return name and status, hide _id:
db.customers.find(
  { status: "active" },
  { name: 1, status: 1, _id: 0 }
)

Chain sort and limit:
db.customers.find({ status: "active" }).sort({ name: 1 }).limit(20)`,

	"aggregate": `Filter and group:
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $group: { _id: "$customer", total: { $sum: "$amount" } } },
  { $sort: { total: -1 } },
  { $limit: 10 }
])

Project only selected fields:
db.customers.aggregate([
  { $project: { _id: 0, name: 1, status: 1 } }
])`,

	"count": `MQL count all documents:
db.customers.count({})

MQL count with a filter:
db.customers.count({ status: "active" })

SQL count rows:
SELECT COUNT(*) FROM public.customers;

SQL count groups:
SELECT status, COUNT(*) FROM public.customers GROUP BY status;`,

	"distinct": `MQL distinct values:
db.customers.distinct("status")

MQL distinct with a filter:
db.customers.distinct("status", { region: "eu" })

SQL DISTINCT:
SELECT DISTINCT status FROM public.customers;`,

	"$match": `Only matching documents continue down the pipeline:
db.orders.aggregate([
  { $match: { status: "paid", total: { $gte: 100 } } }
])`,

	"$group": `Group by customer and sum amounts:
db.orders.aggregate([
  { $group: { _id: "$customer", total: { $sum: "$amount" } } }
])`,

	"$sort": `Sort ascending and descending:
db.customers.find({}).sort({ name: 1, created_at: -1 })

Inside a pipeline:
db.orders.aggregate([{ $sort: { total: -1 } }])`,

	"$project": `Keep name, hide _id:
db.customers.aggregate([
  { $project: { _id: 0, name: 1, status: 1 } }
])`,

	"$lookup": `Join customer data into orders:
db.orders.aggregate([
  { $lookup: {
      from: "customers",
      localField: "customer_id",
      foreignField: "_id",
      as: "customer"
  } }
])`,

	"$unwind": `Expand one array element per document:
db.orders.aggregate([
  { $unwind: "$items" }
])`,

	"$and": `All predicates must match:
db.customers.find({
  $and: [
    { status: "active" },
    { age: { $gte: 18 } }
  ]
})`,

	"$or": `Either predicate may match:
db.customers.find({
  $or: [
    { status: "active" },
    { status: "trial" }
  ]
})`,

	"$in": `Match any value in a list:
db.customers.find({ status: { $in: ["active", "trial"] } })`,

	"$regex": `Case-insensitive match (read-only):
db.customers.find({ name: { $regex: "ann", $options: "i" } })`,

	"$exists": `Require a field:
db.customers.find({ email: { $exists: true } })`,

	"$sum": `Sum one field per group:
db.orders.aggregate([
  { $group: { _id: "$customer", total: { $sum: "$amount" } } }
])`,

	"$avg": `Average one field per group:
db.orders.aggregate([
  { $group: { _id: "$customer", avg_total: { $avg: "$amount" } } }
])`,

	"$addtoset": `Collect unique values per group:
db.orders.aggregate([
  { $group: { _id: "$customer", statuses: { $addToSet: "$status" } } }
])`,

	"$first": `Keep the first value per group:
db.orders.aggregate([
  { $sort: { created_at: 1 } },
  { $group: { _id: "$customer", first_order: { $first: "$_id" } } }
])`,

	"$last": `Keep the last value per group:
db.orders.aggregate([
  { $sort: { created_at: 1 } },
  { $group: { _id: "$customer", last_order: { $last: "$_id" } } }
])`,

	"$limit": `MQL pipeline limit:
db.orders.aggregate([{ $limit: 10 }])

MQL find chain limit:
db.orders.find({}).limit(10)

SQL LIMIT:
SELECT * FROM public.customers LIMIT 100;`,

	"$skip": `Pipeline skip:
db.orders.aggregate([{ $skip: 20 }])

Find chain skip:
db.orders.find({}).skip(20)`,

	"$count": `Emit one count document:
db.orders.aggregate([{ $count: "matching_orders" }])`,

	"limit": `SQL LIMIT:
SELECT * FROM public.customers LIMIT 100;

MQL find chain limit:
db.orders.find({}).limit(10)

MQL pipeline limit:
db.orders.aggregate([{ $limit: 10 }])`,

	"avg": `Average a column:
SELECT AVG(amount) FROM public.orders;`,

	"min": `Smallest value:
SELECT MIN(created_at) FROM public.orders;`,

	"max": `Largest value:
SELECT MAX(created_at) FROM public.orders;`,

	"$out": `Write results to a collection; blocked while r/o:
db.orders.aggregate([
  { $match: { status: "paid" } },
  { $out: "paid_orders" }
])`,

	"$merge": `Merge results into a collection; blocked while r/o:
db.orders.aggregate([
  { $group: { _id: "$customer", total: { $sum: "$amount" } } },
  { $merge: { into: "customer_totals" } }
])`,

	"objectid": `Exact id match:
db.customers.find({ _id: ObjectId("507f1f77bcf86cd799439011") })`,

	"isodate": `Match a timestamp:
db.orders.find({ created_at: { $gte: ISODate("2026-01-01T00:00:00Z") } })`,

	"select": `Project columns:
SELECT id, name, status FROM public.customers;

Filter and order:
SELECT id, name FROM public.customers
WHERE status = 'active'
ORDER BY name
LIMIT 100;`,

	"from": `Read from a table:
SELECT * FROM public.customers;

Read from a subquery:
SELECT * FROM (SELECT id FROM public.customers) AS c;`,

	"where": `Filter rows:
SELECT id FROM public.customers WHERE status = 'active';

Multiple conditions:
SELECT id FROM public.customers
WHERE status = 'active' AND created_at >= '2026-01-01';`,

	"join": `Join orders to customers:
SELECT o.id, c.name
FROM public.orders AS o
JOIN public.customers AS c ON c.id = o.customer_id;`,

	"left": `Keep every customer, even without orders:
SELECT c.id, o.id
FROM public.customers AS c
LEFT JOIN public.orders AS o ON o.customer_id = c.id;`,

	"inner": `Only matching rows:
SELECT c.id, o.id
FROM public.customers AS c
INNER JOIN public.orders AS o ON o.customer_id = c.id;`,

	"on": `The join predicate:
SELECT c.id, o.id
FROM public.customers AS c
JOIN public.orders AS o ON o.customer_id = c.id;`,

	"group": `Group and count:
SELECT status, COUNT(*) AS n
FROM public.customers
GROUP BY status;`,

	"order": `Sort by one or more columns:
SELECT id, name FROM public.customers
ORDER BY name ASC, id DESC;`,

	"having": `Filter groups:
SELECT status, COUNT(*) AS n
FROM public.customers
GROUP BY status
HAVING COUNT(*) > 10;`,

	"offset": `Skip rows:
SELECT * FROM public.customers ORDER BY id LIMIT 100 OFFSET 200;`,

	"explain": `Read plan:
EXPLAIN SELECT * FROM public.customers WHERE status = 'active';

Plan with execution:
EXPLAIN ANALYZE SELECT * FROM public.customers;`,

	"with": `Common table expression:
WITH active AS (
  SELECT * FROM public.customers WHERE status = 'active'
)
SELECT * FROM active;`,

	"sum": `Sum a column:
SELECT SUM(amount) FROM public.orders;`,

	"coalesce": `Fallback for null:
SELECT COALESCE(name, 'unknown') FROM public.customers;`,

	"case": `Conditional value:
SELECT CASE WHEN status = 'active' THEN 1 ELSE 0 END AS is_active
FROM public.customers;`,

	"union": `Stack two result sets:
SELECT id FROM public.customers
UNION
SELECT id FROM archived_customers;`,

	"insert": `Write row; requires r/w:
INSERT INTO public.customers (name, status)
VALUES ('ann', 'active');`,

	"update": `Change row; requires r/w:
UPDATE public.customers
SET status = 'active'
WHERE id = 1;`,

	"delete": `Remove row; requires r/w:
DELETE FROM public.customers WHERE id = 1;`,

	"create": `Create object; requires r/w:
CREATE TABLE public.audit_log (id bigint, created_at timestamptz);`,

	"alter": `Change object; requires r/w:
ALTER TABLE public.customers ADD COLUMN email text;`,

	"drop": `Remove object; requires r/w:
DROP TABLE public.audit_log;`,

	"returning": `Return changed rows:
UPDATE public.customers
SET status = 'active'
WHERE id = 1
RETURNING id, status;`,

	"like": `Pattern match:
SELECT id FROM public.customers WHERE name LIKE 'ann%';`,

	"ilike": `Case-insensitive pattern:
SELECT id FROM public.customers WHERE name ILIKE 'ann%';`,

	"table": `Tree: Enter opens a table's Data tab, Right expands its schema.

SQL shorthand:
TABLE public.customers;`,

	"values": `Literal rows:
VALUES (1, 'ann'), (2, 'bob');`,

	"cast": `Convert a value:
SELECT CAST(id AS text) FROM public.customers;`,

	"now": `Server timestamp:
SELECT NOW();`,

	"current_date": `Server date:
SELECT CURRENT_DATE;`,

	"tree": `Enter opens the selected object.
Right expands the node.
Left collapses it.
/ searches loaded/expanded tree names; typing jumps to the first match; Enter jumps to it; n/N walk matches.
Double / inserts the selected name; Ins inserts the full value such as schema.table.
On a database, schema, table, or collection, e/u opens the membership matrix and r opens privileges.
w asks to switch r/o to r/w from the tree.
Ctrl+D disconnects.`,

	"details": `Select an object and press 1.
Ctrl+X s computes an exact count.
Esc cancels a running count.`,

	"indexes": `Select a table or collection and press 2.
The pane shows details, statistics, and covered columns.`,

	"privileges": `Select an object, database, or principal and open its Privileges tab.
Failed sections show 'not permitted' without hiding the others.`,

	"data": `Enter opens the Data tab.
m appends the next page.
/ filters loaded rows.
Ctrl+F runs a server regexp on the focused field.
Ctrl+A loads a larger window.`,

	"results": `Run a query with Enter or Ctrl+X Ctrl+X.
m appends more results.
/ filters loaded rows.
Esc returns to the object.`,

	"query": `Enter runs the statement.
Ctrl+X Ctrl+X runs it from any zone.
Ctrl+Space completes the word at the cursor.
Up/Down recalls history.`,

	"completion": `Type part of a word and press Ctrl+Space.
Esc then Tab opens the same popup without moving focus.
The cursor may sit at the beginning, middle, or end of the word; accepting replaces the whole word.
The popup appears above the cursor column.
In MQL, $o: offers $or and keeps the colon.
Accepting $or, $and, $in, or $nin adds : [  ] with the cursor inside.
Accepting a function such as COUNT or ObjectId adds () with the cursor inside.
Up/Down move; Enter accepts; Esc closes.
Tab alone still moves focus.`,

	"history": `Focus the query bar and press Up.
Down walks back toward the current draft.
Consecutive duplicate statements are collapsed.`,

	"mode": `The badge starts r/o.
Ctrl+X Ctrl+W asks before enabling r/w; with tree focus, w does the same.
Enter, e, or w accepts the confirmation.
Ctrl+X Ctrl+W again returns to r/o immediately.
Esc releases an open Results cursor before switching.`,

	"filter": `With tree focus, / searches loaded/expanded tree rows; n/N walk matches.
In Data/Results, / filters loaded rows only; n/N walk matches.
Ctrl+F runs a server-side regexp in Data.
Esc clears the filter.`,

	"security": `Expand Security to see users and roles.
In read-write mode press Ctrl+X Ctrl+R to open the access matrix.
Tab or 1/2 switches Principal x Role and Principal/Role x Object.
Ctrl+Up/Down jump within the selected column; Ctrl+Left/Right within the selected row.
The leading [db]/[schema] columns stay frozen while tables/collections scroll when width allows.
Space toggles a cell, e opens native privilege details, / filters, Esc closes.
Use + on the roles group to create a role; role + - can drop one.
On Security, users, or roles, e opens membership and r opens privileges.
On a specific user, e focuses the user column and r focuses the user row.
Select a principal and open Privileges for native grants and memberships.`,

	"status": `The bottom line shows counts, elapsed query time, errors, and cancellation hints.
The top line shows the mode badge and privileged tag.`,

	"help": `Ctrl+F1 explains the focused element or the word under the cursor.
Enter in the help modal shows examples.
Esc closes it.`,
}
