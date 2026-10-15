package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
)

// What a person is, and of whom (table contact_relations). "Alex is the father of Kyle" and "Alex is the former boss of the user"
// are two relations of one person. A relation belongs to the person it is said of, so it is as visible as that person is: an
// unresolved person's relations stay in the conversation of the person, and a relation to a person this repository cannot see is
// not returned.

// AddRelation records that the person (contactID) is the relation of ofContactID, or of the user when ofContactID is 0. The same
// relation again only updates whether it is still so (former). Both people must be ones this repository sees.
func (r *ContactRepository) AddRelation(userID string, contactID int64, relation string, ofContactID int64, former bool) error {
	relation = strings.TrimSpace(relation)
	if userID == "" || contactID == 0 || relation == "" {
		return fmt.Errorf("userId, person and relation required")
	}
	if ofContactID == contactID {
		return fmt.Errorf("a person is not their own relation")
	}
	if c, err := r.GetByID(userID, contactID); err != nil || c == nil {
		return fmt.Errorf("no such person")
	}
	var of interface{}
	if ofContactID != 0 {
		if c, err := r.GetByID(userID, ofContactID); err != nil || c == nil {
			return fmt.Errorf("no such person")
		}
		of = ofContactID
	}
	f := 0
	if former {
		f = 1
	}
	_, err := r.db.Exec(
		`INSERT INTO contact_relations (user_id, contact_id, relation, of_contact_id, former, created_at) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(contact_id, relation, IFNULL(of_contact_id, 0)) DO UPDATE SET former = excluded.former`,
		userID, contactID, relation, of, f, time.Now().Unix(),
	)
	return err
}

// attachRelations loads the relations of the people into them. A failure is logged and leaves the people without relations:
// they are context for the models, never a reason to fail a turn.
func (r *ContactRepository) attachRelations(userID string, contacts ...*models.Contact) {
	ids := make([]interface{}, 0, len(contacts))
	byID := map[int64]*models.Contact{}
	marks := make([]string, 0, len(contacts))
	for _, c := range contacts {
		if c == nil || c.ID == 0 {
			continue
		}
		c.Relations = nil
		byID[c.ID] = c
		ids = append(ids, c.ID)
		marks = append(marks, "?")
	}
	if len(ids) == 0 {
		return
	}
	args := append([]interface{}{userID}, ids...)
	args = append(args, r.conversationID)
	rows, err := r.db.Query(
		`SELECT r.contact_id, r.relation, r.of_contact_id, IFNULL(o.name, ''), r.former
		 FROM contact_relations r LEFT JOIN contacts o ON o.id = r.of_contact_id
		 WHERE r.user_id = ? AND r.contact_id IN (`+strings.Join(marks, ",")+`)
		   AND (r.of_contact_id IS NULL OR (o.status = 'active' AND (o.name_status = 'named' OR (o.origin_conversation_id != '' AND o.origin_conversation_id = ?))))
		 ORDER BY r.id`,
		args...,
	)
	if err != nil {
		log.Printf("[Database] ContactRepository: relations not loaded: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var contactID int64
		var relation, ofName string
		var of *int64
		var former int
		if err := rows.Scan(&contactID, &relation, &of, &ofName, &former); err != nil {
			log.Printf("[Database] ContactRepository: relation not read: %v", err)
			return
		}
		if c := byID[contactID]; c != nil {
			c.Relations = append(c.Relations, models.ContactRelation{Relation: relation, OfName: ofName, Former: former == 1})
		}
	}
}

// moveRelations hands the relations of one person to another (two records of one person were found to be the same person):
// what was said of the first, and what the first was said to be, now hold for the second. A relation the second already has is
// dropped, not duplicated.
func (r *ContactRepository) moveRelations(userID string, from, to int64) error {
	for _, q := range []string{
		`UPDATE OR IGNORE contact_relations SET contact_id = ? WHERE contact_id = ? AND user_id = ?`,
		`UPDATE OR IGNORE contact_relations SET of_contact_id = ? WHERE of_contact_id = ? AND user_id = ?`,
	} {
		if _, err := r.db.Exec(q, to, from, userID); err != nil {
			return err
		}
	}
	// whatever could not be moved (it would repeat one the target has, or make the person their own relation) goes with the old row
	_, err := r.db.Exec(`DELETE FROM contact_relations WHERE user_id = ? AND (contact_id = ? OR of_contact_id = ? OR contact_id = of_contact_id)`, userID, from, from)
	return err
}

// RemoveUserRelation removes a relation of the person TO THE USER (a role the user corrected: "no wait, she is my sister-in-law").
func (r *ContactRepository) RemoveUserRelation(userID string, contactID int64, relation string) error {
	if userID == "" || contactID == 0 || strings.TrimSpace(relation) == "" {
		return fmt.Errorf("userId, person and relation required")
	}
	_, err := r.db.Exec(`DELETE FROM contact_relations WHERE user_id = ? AND contact_id = ? AND of_contact_id IS NULL AND LOWER(relation) = LOWER(?)`,
		userID, contactID, strings.TrimSpace(relation))
	return err
}
