package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// A prompt must not carry the name of a person or the details of one test scenario: a model copies what it is shown (the
// "Dana" of an example turned up in a user's chat, 2026-10-14), and one worked scenario narrows what the model takes the task to
// be. Prompts name no person; they describe the kind of thing in words. This test reads every string literal of the production
// code and fails when one holds a first name or a scenario word from the list below. Code that must hold such a word (a fixture
// for tests, an example in a comment) is not a literal in a non-test file; if a real need appears, add the file to allowed with
// the reason.
func TestNoPersonalNamesOrScenarioWordsInStringLiterals(t *testing.T) {
	names := regexp.MustCompile(`\b(Dana|Anna|Christine|Christine_sub|Maria|Mark|Tom|Sam|Lena|Elena|Sara|Sarah|Mia|Karl|Jonas|Helen|Mike|Quillon|Brandt|Hale|Marcus|John|Jane|Emma|Alex|Lucy|Bob|Alice|Mara|Lace)\b`)
	scenario := regexp.MustCompile(`(?i)\b(fetlife)\b`)
	allowed := map[string]string{
		"tools/mock_llm_client.go": "a mock model for tests: canned answers, never sent to a user or a real model",
	}
	var hits []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if _, ok := allowed[filepath.ToSlash(path)]; ok {
			return nil
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Fatalf("%s: %v", path, perr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			text, uerr := strconv.Unquote(lit.Value)
			if uerr != nil {
				return true
			}
			if m := names.FindString(text); m != "" {
				hits = append(hits, path+":"+strconv.Itoa(fset.Position(lit.Pos()).Line)+" holds the name "+m)
			} else if m := scenario.FindString(text); m != "" {
				hits = append(hits, path+":"+strconv.Itoa(fset.Position(lit.Pos()).Line)+" holds the scenario word "+m)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(hits)
	for _, h := range hits {
		t.Error(h)
	}
}
