package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"

	"moly/tools"
)

// A new conversation is created as "Direct Message". Once Moly has locked a goal, the conversation is named after it with a
// short title (a few words, written by the model in the goal's own language), so the list of previous conversations tells them
// apart. When the user confirms a switch to another goal, the title follows. The work runs in the background, after the reply.

const (
	defaultConversationName = "Direct Message"
	titleMaxRunes           = 48
	titleTimeout            = 5 * time.Minute
)

// titledGoal remembers, for this run, which goal each conversation's title was made from.
var titledGoal sync.Map // conversationID -> goal

// nameConversation names the conversation after its locked goal, unless the title already matches that goal.
func (srv *APIServer) nameConversation(userID, conversationID, goal string) {
	goal = strings.TrimSpace(goal)
	if srv == nil || srv.database == nil || userID == "" || conversationID == "" || goal == "" {
		return
	}
	conn := srv.database.GetConnection()
	var name string
	if err := conn.QueryRow(`SELECT COALESCE(name, '') FROM conversations WHERE id = ? AND user_id = ?`, conversationID, userID).Scan(&name); err != nil {
		return
	}
	if last, ok := titledGoal.Load(conversationID); ok && last.(string) == goal {
		return
	}
	// A title that is not the default and that this run did not make belongs to an earlier run's goal: leave it until the goal changes.
	if _, ok := titledGoal.Load(conversationID); !ok && name != defaultConversationName && name != "" {
		titledGoal.Store(conversationID, goal)
		return
	}

	title := ""
	if srv.llmClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), titleTimeout)
		resp, err := srv.llmClient.Call(ctx, &tools.LLMRequest{
			SystemPrompt: "You name conversations. Write a title of at most five words for the goal below, in the same language as the goal. " +
				"Reply with the title only: no quotes, no full stop, no explanation.",
			UserPrompt:  goal,
			Temperature: 0.2,
			MaxTokens:   30,
			Retries:     1,
			Timeout:     titleTimeout,
		})
		cancel()
		if err == nil && resp != nil {
			title = cleanTitle(resp.Content)
		} else if err != nil {
			log.Printf("[Title] the model gave no title: %v", err)
		}
	}
	if title == "" {
		title = fallbackTitle(goal)
	}
	if title == "" {
		return
	}
	if _, err := conn.Exec(`UPDATE conversations SET name = ? WHERE id = ? AND user_id = ?`, title, conversationID, userID); err != nil {
		log.Printf("[Title] could not save the title: %v", err)
		return
	}
	titledGoal.Store(conversationID, goal)
	log.Printf("[Title] conversation %s named after its goal", conversationID)
}

// cleanTitle takes the model's answer to one clean line, or "" when it is unusable (empty, an explanation, far too long).
func cleanTitle(raw string) string {
	line := strings.TrimSpace(strings.SplitN(strings.TrimSpace(raw), "\n", 2)[0])
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "Title:"), "title:"))
	line = strings.Trim(line, "\"'`*“”‘’ ")
	line = strings.TrimRight(line, ".!:;, ")
	if line == "" || len([]rune(line)) > 4*titleMaxRunes || len(strings.Fields(line)) > 12 {
		return ""
	}
	return clip(line)
}

// fallbackTitle is the start of the goal itself, cut at a word, for when the model gives nothing usable.
func fallbackTitle(goal string) string {
	words := strings.Fields(goal)
	if len(words) == 0 {
		return ""
	}
	if len(words) > 6 {
		words = words[:6]
	}
	return clip(strings.TrimRight(strings.Join(words, " "), ".!?:;, "))
}

// clip shortens to titleMaxRunes at a word boundary and capitalises the first letter.
func clip(s string) string {
	r := []rune(s)
	if len(r) > titleMaxRunes {
		cut := string(r[:titleMaxRunes])
		if i := strings.LastIndex(cut, " "); i > titleMaxRunes/2 {
			cut = cut[:i]
		}
		r = []rune(cut)
	}
	if len(r) > 0 {
		r[0] = unicode.ToUpper(r[0])
	}
	return string(r)
}
