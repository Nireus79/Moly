package tools

import "testing"

func TestSafeJSONParseTakesJSONOutOfFenceAndChatter(t *testing.T) {
	type out struct {
		Answered []bool `json:"answered"`
	}
	for _, in := range []string{
		"```json\n{\"answered\": [true]}\n```",
		"Here is the answer: {\"answered\": [true]} Hope it helps.",
		" {\"answered\": [true]}",
	} {
		var o out
		if err := SafeJSONParse("test", []byte(in), &o); err != nil || len(o.Answered) != 1 || !o.Answered[0] {
			t.Fatalf("%q: err=%v out=%v", in, err, o)
		}
	}
	var two out
	if err := SafeJSONParse("test", []byte(`{"answered": [true]} {"answered": [false]}`), &two); err != nil || !two.Answered[0] {
		t.Fatalf("the first of two values is taken, got %v %v", err, two)
	}
	var o out
	if err := SafeJSONParse("test", []byte("no json here at all"), &o); err == nil {
		t.Fatal("text without JSON is still an error")
	}
	if err := SafeJSONParse("test", []byte("a { broken"), &o); err == nil {
		t.Fatal("broken JSON is still an error")
	}
}
