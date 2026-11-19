// Package help holds the offline context-help corpus shared by completion and
// Ctrl+F1. Every term offered by completion has an entry here.
package help

import (
	"sort"
	"strings"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

var sqlTerms = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
	"ON", "GROUP", "ORDER", "BY", "HAVING", "LIMIT", "OFFSET", "DISTINCT",
	"AS", "AND", "OR", "NOT", "NULL", "IS", "IN", "EXISTS", "CASE", "WHEN",
	"THEN", "ELSE", "END", "UNION", "ALL", "INSERT", "UPDATE", "DELETE",
	"CREATE", "ALTER", "DROP", "EXPLAIN", "ANALYZE", "SHOW", "DESCRIBE",
	"COUNT", "SUM", "AVG", "MIN", "MAX", "COALESCE", "NOW", "CURRENT_DATE",
	"CAST", "WITH", "RETURNING", "VALUES", "TABLE", "LATERAL", "ILIKE", "LIKE",
}

var mysqlTerms = []string{
	"USE", "REGEXP", "RLIKE", "IFNULL", "DATE_FORMAT", "STR_TO_DATE",
	"FIND_IN_SET", "GROUP_CONCAT", "JSON_EXTRACT", "MATCH", "AGAINST",
}

var mongoTerms = []string{
	"find", "aggregate", "count", "distinct",
	"$match", "$group", "$sort", "$limit", "$skip", "$project", "$lookup",
	"$unwind", "$count", "$out", "$merge", "$sum", "$avg", "$min", "$max",
	"$push", "$addToSet", "$first", "$last", "$regex", "$options", "$gte",
	"$gt", "$lte", "$lt", "$eq", "$ne", "$in", "$nin", "$exists", "$and",
	"$or", "$not", "$elemMatch", "ObjectId", "ISODate",
}

var uiTerms = []string{
	"tree", "details", "indexes", "privileges", "data", "results", "query",
	"completion", "history", "mode", "filter", "security", "status", "help",
}

var descriptions = map[string]string{
	"select":       "SQL statement that projects rows from one or more tables.",
	"from":         "Names the table, view, or subquery that supplies rows.",
	"where":        "Filters rows before grouping and projection.",
	"join":         "Combines rows from another table using an ON condition.",
	"left":         "LEFT JOIN keeps every row from the left side.",
	"right":        "RIGHT JOIN keeps every row from the right side.",
	"inner":        "INNER JOIN keeps only matching rows on both sides.",
	"outer":        "OUTER JOIN keeps unmatched rows as null-extended rows.",
	"on":           "The join condition that pairs rows from both sides.",
	"group":        "GROUP BY forms one result row per distinct key.",
	"order":        "ORDER BY sorts the result; add ASC or DESC.",
	"by":           "Used by GROUP BY and ORDER BY.",
	"having":       "Filters grouped rows after GROUP BY.",
	"limit":        "Caps the number of returned rows.",
	"offset":       "Skips a number of rows before returning results.",
	"distinct":     "Removes duplicate projected rows; in MQL, returns unique values for one field.",
	"as":           "Aliases a column or table in the result.",
	"and":          "Boolean conjunction; both operands must be true.",
	"or":           "Boolean disjunction; either operand may be true.",
	"not":          "Boolean negation.",
	"null":         "The absent-value marker; test it with IS NULL.",
	"is":           "Tests NULL or another predicate such as IS TRUE.",
	"in":           "Tests membership in a list or subquery.",
	"exists":       "Tests whether a subquery returns at least one row.",
	"case":         "Conditional expression: CASE WHEN ... THEN ... END.",
	"when":         "Starts one CASE branch.",
	"then":         "Returns the value of a CASE branch.",
	"else":         "Default value when no CASE branch matches.",
	"end":          "Closes a CASE expression.",
	"union":        "Stacks the rows of two compatible SELECT statements.",
	"all":          "With UNION, keeps duplicates; otherwise a modifier.",
	"insert":       "Adds rows. Blocked while read-only mode is on.",
	"update":       "Changes rows. Blocked while read-only mode is on.",
	"delete":       "Removes rows. Blocked while read-only mode is on.",
	"create":       "Creates an object. Blocked while read-only mode is on.",
	"alter":        "Changes an object. Blocked while read-only mode is on.",
	"drop":         "Removes an object. Blocked while read-only mode is on.",
	"explain":      "Shows the plan or plan with ANALYZE; EXPLAIN of a read is allowed.",
	"analyze":      "Collects planner statistics or runs EXPLAIN ANALYZE.",
	"show":         "Native server metadata command.",
	"describe":     "Lists columns or structure for an object.",
	"count":        "Counts rows or matching documents.",
	"sum":          "Adds numeric values; in MQL, an accumulator.",
	"avg":          "Averages numeric values; in MQL, an accumulator.",
	"min":          "Smallest value; in MQL, an accumulator.",
	"max":          "Largest value; in MQL, an accumulator.",
	"coalesce":     "Returns the first non-null argument.",
	"now":          "Current timestamp on the server.",
	"current_date": "Current date on the server.",
	"cast":         "Converts a value to another type.",
	"with":         "Starts a common table expression before the main statement.",
	"returning":    "Returns rows changed by a write statement.",
	"values":       "Builds a literal row set.",
	"table":        "Shorthand for SELECT * FROM a table; also a browsable SQL relation node.",
	"lateral":      "Lets a subquery reference earlier FROM items.",
	"ilike":        "Case-insensitive pattern match in PostgreSQL.",
	"like":         "Pattern match with % and _ wildcards.",
	"use":          "Selects a database in MySQL/MariaDB.",
	"regexp":       "MySQL/MariaDB regular-expression match.",
	"rlike":        "MariaDB alias for REGEXP.",
	"ifnull":       "Returns the second argument when the first is null.",
	"date_format":  "Formats a date with a MySQL format string.",
	"str_to_date":  "Parses a date from a MySQL format string.",
	"find_in_set":  "Returns the position of a value in a comma list.",
	"group_concat": "Concatenates grouped values.",
	"json_extract": "Extracts a JSON value by path.",
	"match":        "Full-text MATCH predicate.",
	"against":      "Full-text search expression used with MATCH.",

	"find":       "MQL command that reads documents from the selected collection.",
	"aggregate":  "MQL command that runs a pipeline of read stages.",
	"$match":     "Aggregation stage that filters documents.",
	"$group":     "Aggregation stage that groups documents and accumulates values.",
	"$sort":      "Aggregation stage or cursor method that orders documents.",
	"$limit":     "Aggregation stage or cursor method that caps documents.",
	"$skip":      "Aggregation stage or cursor method that skips documents.",
	"$project":   "Aggregation stage that shapes output fields.",
	"$lookup":    "Aggregation stage that joins another collection.",
	"$unwind":    "Aggregation stage that expands an array into documents.",
	"$count":     "Aggregation stage that emits one count document.",
	"$out":       "Aggregation stage that writes a collection; blocked while read-only.",
	"$merge":     "Aggregation stage that writes results; blocked while read-only.",
	"$push":      "Accumulator that collects values into an array.",
	"$addtoset":  "Accumulator that collects unique values into an array.",
	"$first":     "Accumulator that keeps the first value in a group.",
	"$last":      "Accumulator that keeps the last value in a group.",
	"$sum":       "Accumulator that adds numeric values.",
	"$avg":       "Accumulator that averages numeric values.",
	"$min":       "Accumulator that keeps the smallest value.",
	"$max":       "Accumulator that keeps the largest value.",
	"$regex":     "Matches a string field against a regular expression.",
	"$options":   "Regex options; i makes matching case-insensitive.",
	"$gte":       "Matches values greater than or equal to the operand.",
	"$gt":        "Matches values greater than the operand.",
	"$lte":       "Matches values less than or equal to the operand.",
	"$lt":        "Matches values less than the operand.",
	"$eq":        "Matches values equal to the operand.",
	"$ne":        "Matches values not equal to the operand.",
	"$in":        "Matches any value in an array.",
	"$nin":       "Matches no value in an array.",
	"$exists":    "Tests whether a field is present.",
	"$and":       "Combines predicates with logical AND.",
	"$or":        "Combines predicates with logical OR.",
	"$not":       "Negates a predicate.",
	"$elemmatch": "Matches an array element against a predicate.",
	"objectid":   "MongoDB 12-byte identifier literal: ObjectId(\"hex\").",
	"isodate":    "MongoDB date literal: ISODate(\"RFC3339\").",

	"tree":       "Left pane: databases, schemas, objects, and Security. Enter opens, Right expands.",
	"database":   "A database node in the tree; expanding it loads schemas or objects.",
	"schema":     "A PostgreSQL schema node that groups tables, views, and sequences.",
	"collection": "A browsable MongoDB collection. Enter opens its Data tab.",
	"view":       "A read-only saved query; v1 shows a details placeholder.",
	"sequence":   "A PostgreSQL sequence; v1 shows a details placeholder.",
	"user":       "A Security user or role node with native grants.",
	"role":       "A native role or MongoDB role node with memberships.",
	"details":    "Main tab with object identity, statistics, and structure.",
	"indexes":    "Main tab with index metadata and covered columns or fields.",
	"privileges": "Main tab with native grants, memberships, and effective checks.",
	"data":       "Main tab with a read-only window over a table or collection.",
	"results":    "Main tab with query rows or the engine error; Esc releases it.",
	"query":      "Bottom input for one SQL or MQL statement. Enter runs it.",
	"completion": "Ctrl+Space opens suggestions for the word at the cursor.",
	"history":    "Up/Down in the query bar recall executed statements for this profile.",
	"mode":       "r/o blocks writes; Ctrl+X Ctrl+W switches after confirmation.",
	"filter":     "/ filters loaded rows; Ctrl+F runs a server regexp on the focused field.",
	"security":   "Tree node that lists native users and roles.",
	"users":      "Security group that lists native user principals.",
	"roles":      "Security group that lists native role principals.",
	"schemas":    "PostgreSQL group that folds every non-public schema.",
	"status":     "Bottom line with counts, running time, and errors.",
	"help":       "Ctrl+F1 explains the focused element or word under the cursor.",
}

// Lookup returns the help text for a term. Matching is case-insensitive.
func Lookup(term string) (string, bool) {
	term = strings.TrimSpace(term)
	if term == "" {
		return "", false
	}
	text, ok := descriptions[strings.ToLower(term)]
	return text, ok
}

// Extended returns the example-driven help body for a term. The short
// description opens the help; Enter replaces it with these examples.
func Extended(term string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(term))
	examples, ok := examples[key]
	if !ok {
		return "", false
	}
	body := ""
	if description, ok := descriptions[key]; ok {
		body = description + "\n\n"
	}
	return body + "Examples\n\n" + examples, true
}

// CompletionTerms returns the dialect keywords, functions, and MQL operators
// offered by completion for one engine. UI element names are not included.
func CompletionTerms(engine domain.Engine) []string {
	var terms []string
	switch engine {
	case domain.EngineMongoDB:
		terms = append(terms, mongoTerms...)
	default:
		terms = append(terms, sqlTerms...)
		if engine == domain.EngineMySQL {
			terms = append(terms, mysqlTerms...)
		}
	}
	sort.SliceStable(terms, func(i, j int) bool {
		return strings.ToLower(terms[i]) < strings.ToLower(terms[j])
	})
	return terms
}

// UITerms returns focusable interface element names for context help.
func UITerms() []string {
	return append([]string(nil), uiTerms...)
}
