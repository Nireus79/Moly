package agents

import "strings"

// A fact's confidence is not the average of everything extracted: each saved fact has its own (user decision,
// 2026-10-10). It is the weakest of three judgements: the model's own number, how the model reached the fact
// (stated in the user's words, implied by them, or guessed), and whether the quoted evidence is really in the message.
const (
	basisStated  = "stated"
	basisImplied = "implied"
	basisGuessed = "guessed"

	// Ceilings per basis. A stated fact keeps the model's number; an implied one cannot be surer than this; a guess
	// is always in doubt.
	capImplied = 0.7
	capGuessed = 0.4

	// doubtBelow is the confidence under which a fact is held and confirmed, never saved.
	doubtBelow = 0.6

	// unknownConfidence stands for a model that gave no number: it is not read as high.
	unknownConfidence = 0.5
)

// FactConfidence combines the three judgements into one number. modelConfidence <= 0 means the model gave none.
// A missing basis is read as implied. grounded is whether the evidence quote was found in the message.
func FactConfidence(modelConfidence float64, basis string, grounded bool) float64 {
	c := modelConfidence
	if c <= 0 {
		c = unknownConfidence
	}
	if c > 1 {
		c = 1
	}
	switch strings.ToLower(strings.TrimSpace(basis)) {
	case basisStated:
	case basisGuessed:
		c = minFloat(c, capGuessed)
	default: // implied, or not given
		c = minFloat(c, capImplied)
	}
	if !grounded {
		c = minFloat(c, capGuessed)
	}
	return c
}

// FactInDoubt reports whether a fact is too uncertain to save.
func FactInDoubt(confidence float64) bool {
	return confidence < doubtBelow
}

// QuoteInMessage reports whether a quote appears in the message, ignoring case and runs of spaces.
// An empty quote is not grounded.
func QuoteInMessage(message, quote string) bool {
	norm := func(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }
	q := norm(quote)
	return q != "" && strings.Contains(norm(message), q)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
