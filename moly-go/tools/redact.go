package tools

import (
	"regexp"
	"strings"
)

// Redaction keeps secrets out of everything Moly stores. Two sources:
//   - the shape of the secret (RedactStructured): payment card numbers pass the Luhn check, IBANs pass the mod-97 check. These are
//     formats, not meanings, so no model is needed and nothing is guessed;
//   - the model's reading of the message (RedactItems): the extraction call lists the secrets the user typed (a password, a PIN, an
//     ID number, a key), copied exactly, and each is cut out of the text.

var (
	cardCandidate = regexp.MustCompile(`\d(?:[ -]?\d){12,18}`)
	ibanCandidate = regexp.MustCompile(`\b[A-Z]{2}\d{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,3})?\b`)
)

const (
	cardMask = "[card number removed]"
	ibanMask = "[IBAN removed]"
	itemMask = "[removed]"
)

func luhnValid(digits string) bool {
	sum, double := 0, false
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return len(digits) >= 13 && sum%10 == 0
}

func ibanValid(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	s = s[4:] + s[:4]
	rem := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			rem = (rem*10 + int(r-'0')) % 97
		case r >= 'A' && r <= 'Z':
			v := int(r-'A') + 10
			rem = (rem*100 + v) % 97
		default:
			return false
		}
	}
	return rem == 1
}

// RedactStructured replaces payment card numbers and IBANs in the text by a mask and returns how many it replaced.
func RedactStructured(s string) (string, int) {
	n := 0
	out := cardCandidate.ReplaceAllStringFunc(s, func(m string) string {
		digits := strings.NewReplacer(" ", "", "-", "").Replace(m)
		if luhnValid(digits) {
			n++
			return cardMask
		}
		return m
	})
	out = ibanCandidate.ReplaceAllStringFunc(out, func(m string) string {
		if ibanValid(m) {
			n++
			return ibanMask
		}
		return m
	})
	return out, n
}

// RedactItems cuts each item (copied exactly from the message by the model) out of the text. An item that does not occur in the text
// is ignored, so a model cannot make the text lose words it did not contain; very short items are ignored.
func RedactItems(s string, items []string) (string, int) {
	n := 0
	for _, it := range items {
		it = strings.TrimSpace(it)
		if len([]rune(it)) < 3 || !strings.Contains(s, it) {
			continue
		}
		s = strings.ReplaceAll(s, it, itemMask)
		n++
	}
	return s, n
}
