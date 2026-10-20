package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Which model serves which task. Moly does not let a person pick a Claude or OpenAI model: the judges and the safety stage are
// calibrated per model, and a wrong pick breaks them without an error. The choice is made from what the person's key can use
// (the provider's own list), in this order, so that a retired or a new model never needs an update of the program:
//  1. a model Moly was tested with (the approved table, shipped with the program), newest first;
//  2. otherwise the newest model of the right kind, found from the list itself (dynamicPick: the provider's own family names,
//     "haiku" for the small checks and "sonnet" for the replies; "mini" and the plain model for OpenAI), marked untested;
//  3. otherwise the provider's suggestion.
// No model name is written in the code: the table is data, and a name that is not in the list is simply skipped.

//go:embed approved_models.json
var approvedModelsJSON []byte

// ApprovedModel is one model Moly was tested with.
type ApprovedModel struct {
	ID       string `json:"id"`
	Approved string `json:"approved"` // the date the live batteries passed on it; empty: listed, not measured yet
	Notes    string `json:"notes"`
}

// ApprovedTable maps a provider to its approved models per role.
type ApprovedTable map[string]map[string][]ApprovedModel

// ApprovedModels is the table shipped with this version.
func ApprovedModels() ApprovedTable {
	var t ApprovedTable
	_ = json.Unmarshal(approvedModelsJSON, &t) // the file is part of the program and tested; "_comment" is not a provider
	delete(t, "_comment")
	return t
}

// RoleModels are the models chosen for the two roles.
type RoleModels struct {
	Fast   string `json:"fast"`
	Reply  string `json:"reply"`
	Tested bool   `json:"tested"` // both are models Moly was tested with; false: the provider's own suggestion is used for what is missing
}

// ResolveRoleModels chooses the model of each role from what the provider lists and the approved table. An approved model that
// the list does not hold (retired, or the key has no access) is skipped for the next approved one. A role with none available
// falls back to the provider's suggestion (list.Suggested) and the result is marked untested: Moly still works, and says so.
func ResolveRoleModels(provider string, list *ModelList, table ApprovedTable) (RoleModels, error) {
	if list == nil || len(list.Models) == 0 {
		return RoleModels{}, fmt.Errorf("the provider lists no model for this key")
	}
	available := map[string]bool{}
	for _, m := range list.Models {
		available[m.ID] = true
	}
	pick := func(role string) (string, bool) {
		for _, a := range table[strings.ToLower(provider)][role] {
			if a.Approved != "" && available[a.ID] {
				return a.ID, true
			}
		}
		// a model listed but not measured yet is used only when it is all there is among the approved ones
		for _, a := range table[strings.ToLower(provider)][role] {
			if available[a.ID] {
				return a.ID, false
			}
		}
		// not in the table at all (a new model, or the table is out of date): the newest model of the right kind in the list
		if id := dynamicPick(provider, role, list); id != "" {
			return id, false
		}
		return "", false
	}
	var out RoleModels
	fastOK, replyOK := false, false
	out.Fast, fastOK = pick("fast")
	out.Reply, replyOK = pick("reply")
	if out.Fast == "" {
		out.Fast = list.Suggested
	}
	if out.Reply == "" {
		out.Reply = list.Suggested
	}
	if out.Fast == "" || out.Reply == "" {
		return RoleModels{}, fmt.Errorf("the provider lists no usable model for this key")
	}
	out.Tested = fastOK && replyOK
	return out, nil
}

// dynamicPick chooses the model of a role from the list alone, by the family words the providers themselves use, never by a
// full model name. The list is newest first for both providers. It returns "" when nothing in the list fits.
//   - Claude: the provider says the line of each model ("haiku", "sonnet", "opus"); an older reply without it is read from the id.
//     The small checks use the newest haiku, the replies the newest sonnet; a list without the wanted line gives "".
//   - OpenAI: ids look like gpt-<version>[-mini|-nano][-<date>]. The replies use the highest version of the plain model, the small
//     checks the highest version of "mini" (or "nano"). Variants that are special products (pro, codex, preview, search, ...)
//     and dated snapshots of a model that has an undated name are left out.
func dynamicPick(provider, role string, list *ModelList) string {
	switch strings.ToLower(provider) {
	case "claude":
		want := "sonnet"
		if role == "fast" {
			want = "haiku"
		}
		for _, m := range list.Models {
			line := strings.ToLower(m.Line)
			if line == "" {
				line = strings.ToLower(m.ID)
			}
			if strings.Contains(line, want) {
				return m.ID
			}
		}
	case "openai":
		return pickOpenAI(role, list)
	}
	return ""
}

var openAIPlain = regexp.MustCompile(`^gpt-(\d+(?:\.\d+)?)(o)?(?:-(mini|nano))?$`)
var openAIDated = regexp.MustCompile(`^(.*)-\d{4}-\d{2}-\d{2}$|^(.*)-\d{4}$`)

func pickOpenAI(role string, list *ModelList) string {
	have := map[string]bool{}
	for _, m := range list.Models {
		have[m.ID] = true
	}
	type cand struct {
		id      string
		version float64
		small   int // 0 plain, 1 mini, 2 nano
	}
	var cands []cand
	for _, m := range list.Models {
		id := m.ID
		if g := openAIDated.FindStringSubmatch(id); g != nil {
			base := g[1]
			if base == "" {
				base = g[2]
			}
			if have[base] {
				continue // a dated snapshot of a model that has its own name
			}
			id = base
		}
		g := openAIPlain.FindStringSubmatch(id)
		if g == nil {
			continue
		}
		v, _ := strconv.ParseFloat(g[1], 64)
		small := 0
		switch g[3] {
		case "mini":
			small = 1
		case "nano":
			small = 2
		}
		cands = append(cands, cand{m.ID, v, small})
	}
	best := func(smalls ...int) string {
		bestID, bestV := "", -1.0
		for _, size := range smalls {
			for _, c := range cands { // newest first: a tie keeps the first
				if c.small == size && c.version > bestV {
					bestID, bestV = c.id, c.version
				}
			}
			if bestID != "" {
				return bestID
			}
		}
		return ""
	}
	if role == "fast" {
		return best(1, 2, 0)
	}
	return best(0)
}
