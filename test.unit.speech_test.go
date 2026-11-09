package speech

import (
	"encoding/json"
	"testing"

	messenger "github.com/slidebolt/sb-messenger-sdk"
)

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

func TestSayRoundTrip(t *testing.T) {
	msg, err := messenger.Mock()
	if err != nil {
		t.Fatal(err)
	}
	defer msg.Close()
	if _, err := msg.Subscribe(SubjectSay, func(m *messenger.Message) {
		var req SayRequest
		_ = json.Unmarshal(m.Data, &req)
		reply := SayReply{Seq: 7}
		if req.Text == "" {
			reply = SayReply{Error: "nothing to say"}
		}
		data, _ := json.Marshal(reply)
		_ = m.Respond(data)
	}); err != nil {
		t.Fatal(err)
	}
	c := NewClient(msg)
	seq, err := c.Say("s1", "hello")
	if err != nil || seq != 7 {
		t.Fatalf("Say: seq=%d err=%v", seq, err)
	}
	if _, err := c.Say("s1", ""); err == nil || err.Error() != "nothing to say" {
		t.Fatalf("Say refused: got %v", err)
	}
}

func TestSupportedInput(t *testing.T) {
	ok := []Input{{InputRTP, CodecPCMU, 8000}, {InputRTP, CodecG722, 16000}}
	bad := []Input{{"rtsp", CodecPCMU, 8000}, {InputRTP, CodecPCMU, 16000}, {InputRTP, CodecG722, 8000}, {InputRTP, "opus", 48000}}
	for _, in := range ok {
		if !SupportedInput(in) {
			t.Errorf("%+v should be supported", in)
		}
	}
	for _, in := range bad {
		if SupportedInput(in) {
			t.Errorf("%+v should not be supported", in)
		}
	}
}
