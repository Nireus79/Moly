package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// The AI provider and the API key of a user come from ONE place: the settings manager (llm_settings.go), which reads the user's
// rows (database.UserAI), turns them into a client (tools.LLMClient) and hands that client to whoever calls a model for that
// user (tools.LLMSource). This test reads the production code and fails if anything else
//   - builds a provider client,
//   - reads the stored settings or keys of users,
//   - reads a vendor key or the provider choice from the environment, or
//   - talks to a model vendor directly.
//
// If a real need appears, add the file to the matching list below, with the reason.
func TestOneSourceOfTruthForProvidersAndKeys(t *testing.T) {
	// call name -> files that may call it
	calls := map[string][]string{
		"NewLLMClient":        {"tools/llm_client.go"},
		"NewLLMClientFromEnv": {"llm_settings.go", "tools/llm_client.go"},
		"NewLLMClientWith":    {"llm_settings.go", "tools/llm_client.go"},
		"GetUserAI":           {"llm_settings.go", "database/user_ai_settings.go"},
		"GetAppSettings":      {"database/app_settings.go"},
		"ClaimLegacy":         {"llm_settings.go", "database/user_ai_settings.go"},
		"ListModels":          {"llm_settings.go", "tools/llm_models.go"},
		"detectModel":         {"tools/llm_client.go", "tools/llm_models.go"},
	}
	// string -> files that may hold it in a string literal
	literals := map[string][]string{
		"api.anthropic.com":    {"tools/llm_client.go", "tools/llm_models.go"},
		"api.openai.com":       {"tools/llm_client.go", "tools/llm_models.go"},
		"ANTHROPIC_API_KEY":    {"tools/llm_client.go", "tools/llm_claude_cli.go"}, // the test provider removes it from the environment of the claude command
		"OPENAI_API_KEY":       {"tools/llm_client.go"},
		"LLM_PROVIDER":         {"tools/llm_client.go", "main.go", "llm_settings.go"}, // main passes the operator's choice to the manager
		"user_ai_keys":         {"database/schema.sql", "database/db.go", "database/user_ai_settings.go", "database/user_export.go"},
		"11434":                {"tools/llm_client.go", "tools/llm_models.go", "config.go", "llm_settings.go"}, // the default address of a local model server
		"/v1/messages":         {"tools/llm_client.go"},
		"/v1/chat/completions": {"tools/llm_client.go"},
	}
	var problems []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		slash := filepath.ToSlash(path)
		allowed := func(list []string) bool {
			for _, f := range list {
				if f == slash {
					return true
				}
			}
			return false
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			t.Fatalf("%s: %v", path, perr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				name := ""
				switch f := x.Fun.(type) {
				case *ast.SelectorExpr:
					name = f.Sel.Name
				case *ast.Ident:
					name = f.Name
				}
				if list, watched := calls[name]; watched && !allowed(list) {
					problems = append(problems, slash+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+" calls "+name)
				}
			case *ast.BasicLit:
				if x.Kind != token.STRING {
					return true
				}
				text, uerr := strconv.Unquote(x.Value)
				if uerr != nil {
					return true
				}
				for lit, list := range literals {
					if strings.Contains(text, lit) && !allowed(list) {
						problems = append(problems, slash+":"+strconv.Itoa(fset.Position(x.Pos()).Line)+" holds "+lit)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
}
