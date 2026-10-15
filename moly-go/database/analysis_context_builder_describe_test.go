package database

import (
	"testing"

	"moly/models"
)

// A named person with no role is described by the relationship and a few traits, so a description of them can be recognised
// (found live, 2026-10-15: she was named, and "the girl I like" in the next message made a second, nameless person).
func TestJudgeLinesDescribeAPersonWithNoRole(t *testing.T) {
	c := &models.Contact{Name: "Lena", NameStatus: NameStatusNamed, Relationship: "romantic", Characteristics: []string{"shy", "female", "submissive", "fourth"}}
	got := JudgeLines([]*models.Contact{c})
	if len(got) != 1 || got[0] != "Lena (romantic; shy; female; submissive)" {
		t.Fatalf("got %q", got)
	}
	plain := &models.Contact{Name: "Tom", NameStatus: NameStatusNamed, Relationship: "other"}
	if got := JudgeLines([]*models.Contact{plain}); got[0] != "Tom" {
		t.Fatalf("a person with nothing known is the bare name, got %q", got)
	}
}
