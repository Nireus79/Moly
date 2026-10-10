package tools

import (
	"encoding/json"
	"testing"
)

func TestLenientBool(t *testing.T) {
	cases := map[string]bool{`true`: true, `false`: false, `"true"`: true, `"false"`: false, `"yes"`: true, `1`: true, `0`: false, `null`: false, `"maybe"`: false}
	for in, want := range cases {
		var v struct {
			B LenientBool `json:"b"`
		}
		if err := json.Unmarshal([]byte(`{"b":`+in+`}`), &v); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if bool(v.B) != want {
			t.Errorf("%s: got %v want %v", in, v.B, want)
		}
	}
}
