package tui

import (
	"testing"

	"github.com/mcgru/db-adm-tui/internal/domain"
)

func TestCompletionTextMQLArrayOperator(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		start  int
		end    int
		item   string
		want   string
		cursor int
	}{
		{"adds key and array", "$o", 0, 2, "$or", "$or: [  ]", 7},
		{"adds array after existing colon", "$o:", 0, 2, "$or", "$or:[  ]", 6},
		{"keeps existing array", "$or: []", 0, 3, "$or", "$or: []", 3},
		{"keeps existing array after partial key", "$o: []", 0, 2, "$or", "$or: []", 3},
		{"preserves following bracket", "{$o}", 1, 3, "$or", "{$or: [  ]}", 8},
		{"fills empty value", "{$o: }", 1, 3, "$or", "{$or: [  ]}", 8},
	}
	for _, tc := range cases {
		got, cursor := completionText(tc.value, tc.start, tc.end, tc.item, domain.EngineMongoDB)
		if got != tc.want || cursor != tc.cursor {
			t.Fatalf("%s: got %q cursor %d, want %q cursor %d", tc.name, got, cursor, tc.want, tc.cursor)
		}
	}
}

func TestCompletionTextFunction(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		start  int
		end    int
		item   string
		want   string
		cursor int
	}{
		{"adds parentheses", "cou", 0, 3, "COUNT", "COUNT()", 6},
		{"keeps existing parentheses", "COUNT(", 0, 5, "COUNT", "COUNT(", 5},
		{"keeps spaced parentheses", "COUNT (", 0, 5, "COUNT", "COUNT (", 5},
		{"adds mongo function", "Obj", 0, 3, "ObjectId", "ObjectId()", 9},
	}
	for _, tc := range cases {
		got, cursor := completionText(tc.value, tc.start, tc.end, tc.item, domain.EnginePostgreSQL)
		if got != tc.want || cursor != tc.cursor {
			t.Fatalf("%s: got %q cursor %d, want %q cursor %d", tc.name, got, cursor, tc.want, tc.cursor)
		}
	}
}
