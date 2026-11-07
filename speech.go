// Package speech defines the contract between SlideBolt binaries and sb-speech.
// A caller opens a session and sends it audio, and gets text back one utterance at
// a time. Control travels over the messenger, audio does not.
package speech

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	messenger "github.com/slidebolt/sb-messenger-sdk"
)

// Messenger subjects.
const (
	SubjectStart          = "speech.sessions.start"
	SubjectStop           = "speech.sessions.stop"
	SubjectSay            = "speech.sessions.say"
	SubjectServiceStarted = "speech.service.started"
)

// Input types and codecs.
const (
	InputRTP  = "rtp"
	CodecPCMU = "pcmu"
)

const requestTimeout = 5 * time.Second

var sessionPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidSession reports whether a session ID is safe to use in a subject.
func ValidSession(id string) bool {
	return sessionPattern.MatchString(id)
}

// EventSubject returns the subject a session's events are published on.
func EventSubject(sessionID string) string {
	return "speech.sessions." + sessionID + ".event"
}

type Input struct {
	Type       string `json:"type"`
	Codec      string `json:"codec"`
	SampleRate int    `json:"sampleRate"`
}

type StartRequest struct {
	Input Input `json:"input"`
}

type StartReply struct {
	SessionID string `json:"sessionId,omitempty"`
	RTPHost   string `json:"rtpHost,omitempty"`
	RTPPort   int    `json:"rtpPort,omitempty"`
	Error     string `json:"error,omitempty"`
}

type StopRequest struct {
	SessionID string `json:"sessionId"`
}

type StopReply struct {
	Error string `json:"error,omitempty"`
}

// Event types.
// MaxSayLength is the longest text one Say request may speak.
const MaxSayLength = 600

// SayRequest speaks text into a session, back to the sender of its audio.
// Saying something new interrupts anything still being spoken.
type SayRequest struct {
	SessionID string `json:"sessionId"`
	Text      string `json:"text"`
}

// SayReply accepts or refuses a SayRequest. Seq identifies the utterance in the
// speaking and spoken events that follow.
type SayReply struct {
	Seq   int    `json:"seq,omitempty"`
	Error string `json:"error,omitempty"`
}

const (
	EventTranscript = "transcript"
	EventError      = "error"
	// EventSpeaking and EventSpoken bracket one utterance from Say. Text is
	// what is being said; EventSpoken carries Error if it could not be finished.
	EventSpeaking = "speaking"
	EventSpoken   = "spoken"
)

// Event reports progress on a session.
type Event struct {
	SessionID string `json:"sessionId"`
	Type      string `json:"type"`
	Seq       int    `json:"seq,omitempty"`
	Text      string `json:"text,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Client is used by other binaries to talk to sb-speech.
type Client struct {
	msg messenger.Messenger
}

// NewClient creates a Client on an existing messenger connection.
func NewClient(msg messenger.Messenger) *Client {
	return &Client{msg: msg}
}

// Start opens a session and returns where to send audio.
func (c *Client) Start(req StartRequest) (StartReply, error) {
	var reply StartReply
	if err := c.request(SubjectStart, req, &reply); err != nil {
		return StartReply{}, err
	}
	if reply.Error != "" {
		return StartReply{}, errors.New(reply.Error)
	}
	return reply, nil
}

// Stop ends a session.
func (c *Client) Stop(sessionID string) error {
	var reply StopReply
	if err := c.request(SubjectStop, StopRequest{SessionID: sessionID}, &reply); err != nil {
		return err
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	return nil
}

// Say speaks text into a session. It returns once the request is accepted; the
// speaking and spoken events report progress.
func (c *Client) Say(sessionID, text string) (int, error) {
	var reply SayReply
	if err := c.request(SubjectSay, SayRequest{SessionID: sessionID, Text: text}, &reply); err != nil {
		return 0, err
	}
	if reply.Error != "" {
		return 0, errors.New(reply.Error)
	}
	return reply.Seq, nil
}

// OnEvent subscribes to one session's events.
func (c *Client) OnEvent(sessionID string, handler func(Event)) (messenger.Subscription, error) {
	return c.msg.Subscribe(EventSubject(sessionID), func(m *messenger.Message) {
		var ev Event
		if err := json.Unmarshal(m.Data, &ev); err == nil {
			handler(ev)
		}
	})
}

// OnServiceStarted is called each time sb-speech starts, when all sessions are gone.
func (c *Client) OnServiceStarted(handler func()) (messenger.Subscription, error) {
	return c.msg.Subscribe(SubjectServiceStarted, func(*messenger.Message) { handler() })
}

func (c *Client) request(subject string, req, reply any) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("speech: marshal: %w", err)
	}
	resp, err := c.msg.Request(subject, data, requestTimeout)
	if err != nil {
		return fmt.Errorf("speech: %s: %w", subject, err)
	}
	if err := json.Unmarshal(resp.Data, reply); err != nil {
		return fmt.Errorf("speech: decode %s reply: %w", subject, err)
	}
	return nil
}
