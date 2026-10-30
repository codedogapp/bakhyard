package copilot

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net"
	"net/textproto"
	"sync"
	"testing"
	"time"

	sdk "github.com/github/copilot-sdk/go"
)

// exitRequest is the request in which the CLI brings the plan.
type exitRequest struct {
	SessionID         string   `json:"sessionId"`
	Summary           string   `json:"summary"`
	PlanContent       string   `json:"planContent"`
	Actions           []string `json:"actions"`
	RecommendedAction string   `json:"recommendedAction"`
}

// session is the part of a request's parameters that names its session.
type session struct {
	SessionID string `json:"sessionId"`
}

// sessionEvent is the notification of an event of a session.
type sessionEvent struct {
	SessionID string           `json:"sessionId"`
	Event     sdk.SessionEvent `json:"event"`
}

// fakeCLI is a stand-in Copilot CLI that a client reaches over TCP. It answers a plan's requests,
// with an error for the method in failing, and notes their parameters. Once it has the prompt,
// it brings plan, unless it's empty, with exitPlanMode.request, and passes the client's answer to
// answers.
type fakeCLI struct {
	plan, failing string
	answers       chan jsontext.Value

	mu    sync.Mutex
	asked map[string]jsontext.Value
	conn  io.Writer
}

func newFakeCLI(plan, failing string) *fakeCLI {
	return &fakeCLI{
		plan: plan, failing: failing,
		answers: make(chan jsontext.Value, 1), asked: map[string]jsontext.Value{},
	}
}

// serve starts f on a free port, for one connection, and returns its address.
func (f *fakeCLI) serve(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		f.mu.Lock()
		f.conn = conn
		f.mu.Unlock()

		f.talk(conn)
		_ = conn.Close()
	}()

	return listener.Addr().String()
}

// talk answers what comes over conn until it closes.
func (f *fakeCLI) talk(conn io.ReadWriter) {
	reader := textproto.NewReader(bufio.NewReader(conn))

	for {
		message, err := read(reader)
		if err != nil {
			return
		}

		if message.Method == "" {
			f.answers <- message.Result

			continue
		}

		if err := f.answer(message); err != nil {
			return
		}
	}
}

// answer notes request and answers it. After the prompt, it brings the plan.
func (f *fakeCLI) answer(request incoming) error {
	f.mu.Lock()
	f.asked[request.Method] = request.Params
	f.mu.Unlock()

	named, err := sessionOf(request.Params)
	if err != nil {
		return err
	}

	reply := outgoing{ID: request.ID, Result: result(request.Method, named)}
	if request.Method == f.failing {
		reply = outgoing{ID: request.ID, Error: &failure{internalError, "failing on purpose"}}
	}

	if err := f.send(reply); err != nil || reply.Error != nil {
		return err
	}

	if request.Method != "session.send" || f.plan == "" {
		return nil
	}

	return f.send(bring(named.SessionID, f.plan))
}

// send sends message to the client.
func (f *fakeCLI) send(message outgoing) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return write(f.conn, message)
}

// replied returns the result of the client's next answer to a request of the fake's, which is
// empty for an error, or fails when no answer comes.
func (f *fakeCLI) replied(t *testing.T) jsontext.Value {
	t.Helper()

	select {
	case result := <-f.answers:
		return result
	case <-time.After(wait):
		t.Fatal("the client didn't answer")
	}

	return nil
}

// emit sends the client events of session id, as the CLI does while the agent works.
func (f *fakeCLI) emit(t *testing.T, id string, events ...sdk.SessionEvent) {
	t.Helper()

	for _, event := range events {
		notification := outgoing{Method: "session.event", Params: sessionEvent{id, event}}
		if err := f.send(notification); err != nil {
			t.Fatal(err)
		}
	}
}

// bring returns the request that brings plan to the client of session id.
func bring(id, plan string) outgoing {
	request := exitRequest{
		SessionID: id, Summary: "A plan", PlanContent: plan,
		Actions:           []string{"autopilot", "interactive", "exit_only"},
		RecommendedAction: "autopilot",
	}

	return outgoing{ID: jsontext.Value("1"), Method: "exitPlanMode.request", Params: request}
}

// sessionOf returns the session that params name, if any.
func sessionOf(params jsontext.Value) (session, error) {
	var named session
	if len(params) == 0 {
		return named, nil
	}

	err := json.Unmarshal(params, &named)

	return named, err
}

// record is the CLI's record of a plan's session: the user's prompt, a message that the CLI adds,
// a message of the assistant that only calls a tool, one with text, and the agent's question
// through ask_user, which the user answered with one of its choices.
const record = `{"events": [
	{"id": "1", "parentId": null, "timestamp": "2026-01-02T03:04:05Z", "type": "user.message",
		"data": {"content": "Plan a trip"}},
	{"id": "2", "parentId": "1", "timestamp": "2026-01-02T03:04:06Z", "type": "user.message",
		"data": {"content": "Mind the plan", "source": "system"}},
	{"id": "3", "parentId": "2", "timestamp": "2026-01-02T03:04:07Z",
		"type": "assistant.message", "data": {"messageId": "m1", "content": ""}},
	{"id": "4", "parentId": "3", "timestamp": "2026-01-02T03:04:08Z",
		"type": "assistant.message", "data": {"messageId": "m2", "content": "Where to?"}},
	{"id": "5", "parentId": "4", "timestamp": "2026-01-02T03:04:09Z",
		"type": "tool.execution_start", "data": {"toolCallId": "call-1", "toolName": "ask_user",
			"arguments": {"question": "Which city?", "choices": ["Rome", "Paris"]}}},
	{"id": "6", "parentId": "5", "timestamp": "2026-01-02T03:04:10Z",
		"type": "tool.execution_complete", "data": {"toolCallId": "call-1", "success": true,
			"result": {"content": "User selected: Rome", "detailedContent": "User selected: Rome"}}}
]}`

// result returns the CLI's answer to a request for method in session.
func result(method string, named session) any {
	switch method {
	case "connect":
		return map[string]any{"ok": true, "protocolVersion": 3, "version": "1.0.105"}
	case "session.create":
		return map[string]string{"sessionId": named.SessionID, "workspacePath": "/workspace"}
	case "session.send":
		return map[string]string{"messageId": "message-1"}
	case "session.getMessages":
		return jsontext.Value(record)
	case "session.detach":
		return map[string]bool{"success": true}
	default:
		return map[string]any{}
	}
}

// asks reports whether the client asked for method, and decodes the parameters it last sent
// into params.
func (f *fakeCLI) asks(t *testing.T, method string, params any) bool {
	t.Helper()

	f.mu.Lock()
	sent, found := f.asked[method]
	f.mu.Unlock()

	if found && params != nil {
		if err := json.Unmarshal(sent, params); err != nil {
			t.Fatal(err)
		}
	}

	return found
}
