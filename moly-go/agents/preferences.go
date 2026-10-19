package agents

import "strings"

// The user's saved preferences (Settings: communication style and tone) as instructions for the reply writer. The settings page
// stores the labels it shows ("Casual", "Formal & Respectful", "Humorous"), so they are matched by their first word, not by an
// exact lower-case string (the old switch compared with "casual" and never matched "Casual").

// styleInstruction turns a communication style into an instruction. An unknown style gets the neutral one.
func styleInstruction(style string) string {
	switch firstWord(style) {
	case "formal":
		return " Be respectful and professional. Use clear, direct language. Avoid excessive friendliness or casual expressions."
	case "casual":
		return " Be relaxed and natural. Use conversational language. Feel free to be friendly and approachable."
	case "direct":
		return " Be direct and brief. Say the point first, without padding or cushioning."
	case "authentic":
		return " Be honest and unpolished. Say what you really think, in plain words."
	case "playful":
		return " Be warm and engaging. Use light humor where appropriate. Show genuine curiosity and enthusiasm."
	default:
		return " Be warm yet respectful. Adapt your tone to match theirs."
	}
}

// toneInstruction turns the preferred tone into an instruction; "" when the tone needs none.
func toneInstruction(tone string) string {
	if firstWord(tone) == "humorous" {
		return "The user chose a humorous tone, so the reply has wit in it: this is not optional and not only for light topics. " +
			"Write like a quick, irreverent friend with a dry sense of humor: a sharp one-liner, playful exaggeration, deadpan understatement, " +
			"a gently mocking remark about the situation or about the other people in it (the boss, the system, the absurdity), at least once in every reply. " +
			"The help itself stays useful and clear; the humor rides on top of it and never replaces it. " +
			"Never joke at the expense of the user's pain or fear, and never about a death, a danger, or a crime that hurts someone. " +
			"When the user is in real distress or danger, or has just been refused, drop the jokes and be plainly warm. " +
			"A message you write for them to send to someone else keeps the register that suits that person; add a light touch of wit to it only when the user asked for something playful."
	}
	return ""
}

func firstWord(s string) string {
	f := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	if len(f) == 0 {
		return ""
	}
	return strings.Trim(f[0], ".,;:!&")
}
