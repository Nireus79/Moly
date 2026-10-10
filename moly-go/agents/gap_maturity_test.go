package agents

import "testing"

func TestGapMaturityCountsAnsweredOverAll(t *testing.T) {
	cases := []struct {
		answered, open int
		want           float64
	}{
		{0, 0, 0}, {0, 3, 0}, {1, 1, 0.5}, {4, 1, 0.8}, {3, 0, 1},
	}
	for _, c := range cases {
		if got := GapMaturity(c.answered, c.open); got != c.want {
			t.Errorf("GapMaturity(%d, %d) = %v, want %v", c.answered, c.open, got, c.want)
		}
	}
	if ConversationGapMaturity(nil, "c") != 0 {
		t.Fatal("no store, no maturity")
	}
}
