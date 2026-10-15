package agents

import (
	"context"
	"os"
	"regexp"
	"testing"

	"moly/tools"
)

// A live check (off by default): the notice appears when the reply does not already say that Moly does not read the page, and not when it
// does. The replies are ones Moly gave in manual chats.
func TestLiveLinkNotice(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") == "" {
		t.Skip()
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Skip(err)
	}
	cases := []struct {
		name, msg, reply string
		wantNotice       bool
	}{
		{"offers a summary", "What do you think of this? https://example.org/article/9921",
			"What would be most useful to you: a short summary of the article, your overall reaction, a check of its claims or credibility, or feedback on something specific in it?", true},
		{"offers to look", "Please read https://example.org/profile/nora_k and summarise what she is like, then help me write her a first message.",
			"Happy to look at her profile and help you draft that first message. Before I do, what's your reason for reaching out: is it dating, work or networking, or something else?", true},
		{"ignores the link", "She wrote her view here: https://example.org/forum/thread/4821 and I think she is wrong. Use what she wrote.",
			"What is causing the project delay, and how long do you expect it to last?", true},
		{"asks for a paste but does not say why", "My friend posted this about our trip: https://example.org/blog/trip and I want to answer her.",
			"Could you paste the part of the post you want to respond to?", true},
		{"already says it", "Look at https://example.org/forum/thread/4821 for her view.",
			"I don't open or read web pages, so I can't use what is behind that link. Could you paste the part of her post you disagree with?", false},
		{"greek", "Δες αυτό: https://example.org/post/12 και πες μου τη γνώμη σου.",
			"Θέλεις μια σύντομη περίληψη του άρθρου ή τη γνώμη μου για το θέμα;", true},
		{"no link", "She wrote her view of it in a long post and I think she is wrong.", "What is the main point you disagree with?", false},
	}
	for _, k := range cases {
		ok := 0
		var last string
		for i := 0; i < 3; i++ {
			last = LinkNotice(context.Background(), c, k.msg, k.reply)
			if (last != "") == k.wantNotice {
				ok++
			}
		}
		t.Logf("%d/3 notice=%v for %s -> %q", ok, k.wantNotice, k.name, last)
		if ok < 2 {
			t.Errorf("%s: notice expected=%v, got that %d of 3 times", k.name, k.wantNotice, ok)
		}
		if k.name == "greek" && !regexp.MustCompile(`\p{Greek}`).MatchString(last) {
			t.Errorf("the notice for a Greek message is not Greek: %q", last)
		}
	}
}
