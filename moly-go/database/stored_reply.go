package database

import "encoding/json"

// StoredReplyText returns the reply text of a stored assistant message.
// Rows written before the text was stored plainly hold {"phase": ..., "response": ...}; those are unwrapped.
func StoredReplyText(content string) string {
	if len(content) == 0 || content[0] != '{' {
		return content
	}
	var wrapped struct {
		Response *string `json:"response"`
	}
	if err := json.Unmarshal([]byte(content), &wrapped); err != nil || wrapped.Response == nil {
		return content
	}
	return *wrapped.Response
}
