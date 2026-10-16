package database

import (
	"database/sql"
	"fmt"
	"strings"

	"moly/models"
)

// ProjectToContacts writes the people of the memory into the contacts (and contact_relations) tables, which the readers that have not moved to
// the memory yet still read (docs/MEMORY_REDESIGN.md, step 5b). The memory is the source of truth; nothing reads the contacts back into it.
// repo must be the conversation's repository (ContactRepository.InConversation). It is safe to run after every update: it only makes the
// contacts equal to the memory.
func (s *MemoryStore) ProjectToContacts(repo *ContactRepository, userID, conversationID string) error {
	mem, err := s.Snapshot(userID, conversationID)
	if err != nil {
		return err
	}
	// the contact each person is already linked to
	links := map[string]int64{}
	rows, err := s.db.Query(`SELECT id, contact_id FROM memory_people WHERE user_id = ? AND contact_id IS NOT NULL`, userID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var cid sql.NullInt64
		if rows.Scan(&id, &cid) == nil && cid.Valid {
			links[personKey(id)] = cid.Int64
		}
	}
	rows.Close()

	contactOf := map[string]int64{}
	for _, p := range mem.People {
		c, err := s.projectPerson(repo, userID, conversationID, p, links[p.ID])
		if err != nil {
			return fmt.Errorf("%s: %w", p.ID, err)
		}
		contactOf[p.ID] = c.ID
		if pid, ok := parseKey("p", p.ID); ok && links[p.ID] != c.ID {
			if _, err := s.db.Exec(`UPDATE memory_people SET contact_id = ? WHERE id = ? AND user_id = ?`, c.ID, pid, userID); err != nil {
				return err
			}
		}
	}
	// relations, once every person has a row
	for _, p := range mem.People {
		cid := contactOf[p.ID]
		if err := repo.ClearRelations(userID, cid); err != nil {
			return err
		}
		for _, r := range p.Roles {
			of := int64(0)
			if r.Of != models.MemoryUser {
				of = contactOf[r.Of]
				if of == 0 {
					continue
				}
			}
			if err := repo.AddRelation(userID, cid, r.Role, of, r.Former); err != nil {
				return fmt.Errorf("%s role %q: %w", p.ID, r.Role, err)
			}
		}
	}
	// a person merged into another no longer has a row of their own (only a person without a name is removed)
	mrows, err := s.db.Query(`SELECT contact_id FROM memory_people WHERE user_id = ? AND conversation_id = ? AND status = 'merged' AND contact_id IS NOT NULL`, userID, conversationID)
	if err != nil {
		return err
	}
	var gone []int64
	for mrows.Next() {
		var cid int64
		if mrows.Scan(&cid) == nil {
			gone = append(gone, cid)
		}
	}
	mrows.Close()
	for _, cid := range gone {
		if _, err := s.db.Exec(`DELETE FROM contacts WHERE id = ? AND user_id = ? AND name_status != 'named'`, cid, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryStore) projectPerson(repo *ContactRepository, userID, conversationID string, p models.MemoryPerson, linked int64) (*models.Contact, error) {
	var c *models.Contact
	if linked != 0 {
		c, _ = repo.GetByID(userID, linked)
	}
	status := p.NameStatus
	if status == "declined" {
		status = NameStatusNameless
	}
	if p.Name != "" { // a named person
		switch {
		case c != nil && c.NameStatus != NameStatusNamed:
			named, err := repo.NameContact(userID, c.ID, p.Name) // the same row takes the name, or merges into the person the user already has
			if err != nil {
				return nil, err
			}
			c = named
		case c == nil:
			if known, err := repo.GetNamedByName(userID, p.Name); err == nil && known != nil {
				c = known
			} else {
				c = &models.Contact{UserID: userID, Name: p.Name, NameStatus: NameStatusNamed, Relationship: "other", CreatedVia: "conversation", Status: "active", Confidence: 0.9}
			}
		}
	} else { // a person without a name yet, stored under the user's word for them
		label := strings.TrimSpace(p.Label)
		if label == "" {
			label = "person"
		}
		if c == nil {
			if found, err := repo.GetUnresolvedByLabel(userID, label); err == nil && found != nil {
				c = found
			} else {
				c = &models.Contact{UserID: userID, Name: label, NameStatus: status, Relationship: "other", CreatedVia: "conversation", Status: "active", Confidence: 0.9, OriginConversationID: conversationID}
			}
		}
	}
	if c.ID != 0 && c.NameStatus != NameStatusNamed && status != "" && status != c.NameStatus && (status == NameStatusUnnamed || status == NameStatusAsked || status == NameStatusNameless) {
		if err := repo.SetNameStatus(userID, c.ID, status); err != nil {
			return nil, err
		}
		c.NameStatus = status
	}
	c.ContactRole = firstUserRole(p)
	c.Characteristics = append([]string(nil), p.Traits...)
	if c.Relationship == "" {
		c.Relationship = "other"
	}
	if err := repo.Save(c); err != nil {
		return nil, err
	}
	return c, nil
}

// firstUserRole is what the person is to the user: the first role of the memory that is the user's, a present one before a former one.
func firstUserRole(p models.MemoryPerson) string {
	former := ""
	for _, r := range p.Roles {
		if r.Of != models.MemoryUser {
			continue
		}
		if !r.Former {
			return r.Role
		}
		if former == "" {
			former = r.Role
		}
	}
	return former
}


// ContactIDOf is the contact row that projects a person of the memory ("p<number>"), or 0 when there is none yet.
func (s *MemoryStore) ContactIDOf(userID, person string) int64 {
	pid, ok := parseKey("p", person)
	if !ok {
		return 0
	}
	var cid sql.NullInt64
	if err := s.db.QueryRow(`SELECT contact_id FROM memory_people WHERE id = ? AND user_id = ?`, pid, userID).Scan(&cid); err != nil || !cid.Valid {
		return 0
	}
	return cid.Int64
}
