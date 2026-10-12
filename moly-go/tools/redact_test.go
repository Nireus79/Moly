package tools

import (
	"strings"
	"testing"
)

func TestRedactStructuredRemovesCardsAndIBANsOnly(t *testing.T) {
	cases := map[string]string{
		"My card is 4111 1111 1111 1111, charge was wrong.":    "My card is [card number removed], charge was wrong.",
		"4111-1111-1111-1111":                                  "[card number removed]",
		"4111111111111111":                                     "[card number removed]",
		"IBAN GR16 0110 1250 0000 0001 2300 695 for the bank.": "IBAN [IBAN removed] for the bank.",
		"Order number 1234 5678 9012 3456 was late.":           "Order number 1234 5678 9012 3456 was late.", // fails Luhn: not a card
		"I was born in 1984 and have 3 sisters.":               "I was born in 1984 and have 3 sisters.",
		"Call me on 210 123 4567 tomorrow":                     "Call me on 210 123 4567 tomorrow",
	}
	for in, want := range cases {
		if got, _ := RedactStructured(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestRedactItemsCutsExactSecretsAndIgnoresInventions(t *testing.T) {
	got, n := RedactItems("My password is hunter2 and my PIN is 4921.", []string{"hunter2", "4921", "not in the text", "ab"})
	if got != "My password is [removed] and my PIN is [removed]." || n != 2 {
		t.Fatalf("got %q (%d)", got, n)
	}
	if got, _ := RedactItems("nothing secret here", []string{"secret"}); !strings.Contains(got, "[removed]") {
		t.Fatalf("an item that occurs is cut, got %q", got)
	}
}
