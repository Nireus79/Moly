package agents

import (
	"strings"
	"unicode/utf8"

	"moly/models"
)

// What the extraction model returns is data from outside: it is checked for its shape before anything stores or uses it. The checks are
// about form (a trait is a few words, a name is not a sentence, nothing is repeated, a web address is no trait), never about meaning, so
// no word list decides anything. Found in dynamic runs and manual chats: a relationship "romantic + other", a clause as a label, and
// the form's own tag format ("USER|trait|0.8") inside a trait.
const (
	maxTraitWords  = 6
	maxTraitRunes  = 60
	maxListItems   = 30
	maxNameWords   = 6
	maxNameRunes   = 60
	maxLabelRunes  = 80
	maxRoleWords   = 5
	maxRelationLen = 5
)

func wordCount(s string) int { return len(strings.Fields(s)) }

func hasWebAddress(s string) bool { return linkPattern.MatchString(s) }

// cleanTrait returns the trait as a clean short phrase, or "" when it is not one.
func cleanTrait(raw string) string {
	t := strings.TrimSpace(raw)
	if strings.Contains(t, "|") { // the form's tag format: "USER|trait|0.8" or "CONTACT_name|trait|0.8"
		parts := strings.Split(t, "|")
		if len(parts) >= 2 {
			t = strings.TrimSpace(parts[1])
		}
	}
	t = strings.Trim(t, " \t\r\n\"'“”.,;:")
	if t == "" || wordCount(t) > maxTraitWords || utf8.RuneCountInString(t) > maxTraitRunes || strings.ContainsAny(t, "\r\n") || hasWebAddress(t) {
		return ""
	}
	return t
}

// cleanList cleans every item, drops what is not a clean phrase, removes repeats (any letter case) and caps the length. skip are values that
// are not traits (the person's own name).
func cleanList(items []string, skip ...string) []string {
	if len(items) == 0 {
		return items
	}
	seen := map[string]bool{}
	for _, s := range skip {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			seen[s] = true
		}
	}
	var out []string
	for _, it := range items {
		t := cleanTrait(it)
		key := strings.ToLower(t)
		if t == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
		if len(out) == maxListItems {
			break
		}
	}
	return out
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// sanitizeContact checks one extracted person. It returns nil when nothing is left that names the person.
func sanitizeContact(c *models.ExtractedContact) *models.ExtractedContact {
	if c == nil {
		return nil
	}
	c.Name = strings.TrimSpace(c.Name)
	c.Label = strings.TrimSpace(c.Label)
	c.Role = strings.TrimSpace(c.Role)
	if hasWebAddress(c.Name) || strings.ContainsAny(c.Name, "\r\n?") || wordCount(c.Name) > maxNameWords || utf8.RuneCountInString(c.Name) > maxNameRunes {
		// a sentence or an address is not a name: the person is kept by the user's word for them, when there is one
		c.Name, c.NameKnown = "", false
	}
	if hasWebAddress(c.Label) || strings.ContainsAny(c.Label, "\r\n") || utf8.RuneCountInString(c.Label) > maxLabelRunes {
		c.Label = ""
	}
	if c.Name == "" && c.Label != "" {
		c.Name = c.Label
	}
	if c.Name == "" {
		return nil
	}
	if wordCount(c.Role) > maxRoleWords || hasWebAddress(c.Role) || strings.ContainsAny(c.Role, "\r\n") {
		c.Role = ""
	}
	c.Traits = cleanList(c.Traits, c.Name)
	c.Confidence = clamp01(c.Confidence)
	return c
}

// sanitizeExtraction checks everything the extraction returned, in place.
func sanitizeExtraction(ec *models.ExtractedContext) {
	if ec == nil {
		return
	}
	ec.Contact = sanitizeContact(ec.Contact)
	var others []*models.ExtractedContact
	for _, o := range ec.OtherContacts {
		if o = sanitizeContact(o); o != nil {
			others = append(others, o)
		}
	}
	ec.OtherContacts = others
	ec.UserCharacteristics = cleanList(ec.UserCharacteristics)
	ec.UserInterests = cleanList(ec.UserInterests)
	ec.UserValues = cleanList(ec.UserValues)
	if len(ec.ContactCharacteristics) > 0 {
		clean := map[string][]string{}
		for who, traits := range ec.ContactCharacteristics {
			who = strings.TrimSpace(who)
			if who == "" || hasWebAddress(who) || wordCount(who) > maxNameWords {
				continue
			}
			if list := cleanList(traits, who); len(list) > 0 {
				clean[who] = append(clean[who], list...)
			}
		}
		ec.ContactCharacteristics = clean
	}
	var relations []models.ExtractedRelation
	for _, r := range ec.Relations {
		r.Person, r.Relation, r.Of = strings.TrimSpace(r.Person), strings.TrimSpace(r.Relation), strings.TrimSpace(r.Of)
		if r.Person == "" || r.Relation == "" || wordCount(r.Relation) > maxRelationLen || hasWebAddress(r.Relation) {
			continue
		}
		relations = append(relations, r)
	}
	ec.Relations = relations
}
