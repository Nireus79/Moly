package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Which model serves which task. Moly does not let a person pick a Claude or OpenAI model: the judges and the safety stage are
// calibrated per model, and a wrong pick breaks them without an error. It uses the models it was tested with (the approved
// table, shipped with the program), newest first, among those the person's key can use (the provider's own list).

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
