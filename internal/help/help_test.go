package help

import (
	"strings"
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestEveryCompletionTermHasHelp(t *testing.T) {
	for _, engine := range []domain.Engine{
		domain.EnginePostgreSQL, domain.EngineMySQL, domain.EngineMongoDB,
	} {
		for _, term := range CompletionTerms(engine) {
			if _, ok := Lookup(term); !ok {
				t.Fatalf("%s completion term %q has no help", engine, term)
			}
		}
	}
}

func TestEveryUITermHasHelp(t *testing.T) {
	for _, term := range UITerms() {
		if _, ok := Lookup(term); !ok {
			t.Fatalf("UI term %q has no help", term)
		}
	}
}

func TestFindExtendedHelpShowsExamples(t *testing.T) {
	body, ok := Extended("find")
	if !ok {
		t.Fatal("no extended find help")
	}
	for _, want := range []string{"$and", "Projection", "_id: 0", "filter"} {
		if !strings.Contains(body, want) {
			t.Fatalf("extended find help missing %q:\n%s", want, body)
		}
	}
}

func TestFunctionAndOperatorTemplates(t *testing.T) {
	for _, term := range []string{"COUNT", "COALESCE", "ObjectId", "ISODate"} {
		if !FunctionTerm(term) {
			t.Fatalf("%q is not a function term", term)
		}
	}
	if FunctionTerm("count") {
		t.Fatal("lowercase dynamic identifier treated as a function")
	}
	for _, term := range []string{"$and", "$or", "$in", "$nin"} {
		if !MQLArrayTerm(term) {
			t.Fatalf("%q is not an array term", term)
		}
	}
}

func TestLookupIsCaseInsensitive(t *testing.T) {
	if _, ok := Lookup("SeLeCt"); !ok {
		t.Fatal("mixed-case lookup failed")
	}
	if _, ok := Lookup("$MATCH"); !ok {
		t.Fatal("MQL operator lookup failed")
	}
	if _, ok := Lookup("nope"); ok {
		t.Fatal("unknown term found")
	}
}
