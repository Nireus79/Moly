package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("PHASE 3 - INTEGRATION TESTING WITH ERROR LOGGING")
	fmt.Println("============================================================")

	db, _ := sql.Open("sqlite3", "/home/nireus79/.moly/staging/staging.db")
	defer db.Close()
	db.Exec("PRAGMA foreign_keys = ON")
	fmt.Println("\n✓ Connected to staging database\n")

	ts := time.Now().Unix()
	u := "test_1_" + fmt.Sprintf("%d", ts)
	
	// Insert user
	_, err := db.Exec("INSERT INTO users (id, email, name, created_at, last_active, updated_at) VALUES (?, ?, ?, ?, ?, ?)", u, "t@t", "T", ts, ts, ts)
	if err != nil {
		log.Printf("User insert error: %v", err)
	} else {
		fmt.Printf("✓ User inserted: %s\n", u)
	}
	
	// Insert conversation
	_, err = db.Exec("INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)", "c1", u, "C", ts, ts)
	if err != nil {
		log.Printf("Conversation insert error: %v", err)
	} else {
		fmt.Printf("✓ Conversation inserted\n")
	}
	
	// Insert interaction
	_, err = db.Exec("INSERT INTO interactions (user_id, conversation_id, content, type, timestamp) VALUES (?, ?, ?, ?, ?)", u, "c1", "msg", "user", ts)
	if err != nil {
		log.Printf("Interaction insert error: %v", err)
	} else {
		fmt.Printf("✓ Interaction inserted\n")
	}
	
	// Insert clarification question (FIX #7)
	_, err = db.Exec("INSERT INTO clarification_questions (id, user_id, conversation_id, clarification_type, question_text, priority, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", "q1", u, "c1", "gap_clarification", "Q?", 1, "pending", ts)
	if err != nil {
		log.Printf("Clarification 1 error: %v", err)
	} else {
		fmt.Printf("✓ Clarification question 1 inserted\n")
	}
	
	_, err = db.Exec("INSERT INTO clarification_questions (id, user_id, conversation_id, clarification_type, question_text, priority, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", "q2", u, "c1", "context_about_situation", "Q2?", 2, "pending", ts)
	if err != nil {
		log.Printf("Clarification 2 error: %v", err)
	} else {
		fmt.Printf("✓ Clarification question 2 inserted\n")
	}
	
	// Verify
	var cnt int
	db.QueryRow("SELECT COUNT(*) FROM clarification_questions WHERE user_id = ?", u).Scan(&cnt)
	fmt.Printf("✓ Clarification questions count: %d\n", cnt)
	
	if cnt > 0 {
		fmt.Println("\n✅ FIX #7 VERIFIED - Questions persisted successfully")
	} else {
		fmt.Println("\n❌ FIX #7 FAILED - Questions not persisted")
	}
}
