package speech

import "testing"

func TestValidSession(t *testing.T) {
	for _, id := range []string{"abc", "a-b_c1"} {
		if !ValidSession(id) {
			t.Errorf("%q should be valid", id)
		}
	}
	for _, id := range []string{"", "a.b", "a*", "a>", "a b", "a/b"} {
		if ValidSession(id) {
			t.Errorf("%q should be invalid", id)
		}
	}
}

func TestEventSubject(t *testing.T) {
	if got := EventSubject("s1"); got != "speech.sessions.s1.event" {
		t.Fatalf("got %q", got)
	}
}
