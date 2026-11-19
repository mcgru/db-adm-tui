package help

// FunctionTerm reports whether a completion term should receive an empty
// argument list when accepted. Matching is exact so a lowercase database name
// does not become a function call.
func FunctionTerm(term string) bool {
	return functionTerms[term]
}

// MQLArrayTerm reports whether a Mongo operator expects an array value and
// should receive a ": [  ]" template with the cursor inside.
func MQLArrayTerm(term string) bool {
	return mongoArrayTerms[term]
}

var functionTerms = map[string]bool{
	"COUNT":        true,
	"SUM":          true,
	"AVG":          true,
	"MIN":          true,
	"MAX":          true,
	"COALESCE":     true,
	"NOW":          true,
	"CAST":         true,
	"IFNULL":       true,
	"DATE_FORMAT":  true,
	"STR_TO_DATE":  true,
	"FIND_IN_SET":  true,
	"GROUP_CONCAT": true,
	"JSON_EXTRACT": true,
	"MATCH":        true,
	"AGAINST":      true,
	"ObjectId":     true,
	"ISODate":      true,
}

var mongoArrayTerms = map[string]bool{
	"$and": true,
	"$or":  true,
	"$nor": true,
	"$in":  true,
	"$nin": true,
}
