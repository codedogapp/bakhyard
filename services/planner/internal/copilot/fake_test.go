package copilot

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"sync"
	"testing"
)

// incoming is a JSON-RPC 2.0 message from the client: a request if it has a method, else an
// answer to the fake's request.
type incoming struct {
	ID     jsontext.Value `json:"id"`
	Method string         `json:"method"`
	Params jsontext.Value `json:"params"`
	Result jsontext.Value `json:"result"`
}

// outgoing is a JSON-RPC 2.0 message from the fake.
type outgoing struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      jsontext.Value `json:"id"`
	Method  string         `json:"method,omitzero"`
	Params  any            `json:"params,omitzero"`
	Result  any            `json:"result,omitzero"`
	Error   *failure       `json:"error,omitzero"`
}

// internalError is JSON-RPC's code for a server's internal error.
const internalError = -32603

// failure is a JSON-RPC 2.0 error.
type failure struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

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

// fakeCLI is a stand-in Copilot CLI that a client reaches over TCP. It answers a plan's requests,
// with an error for the method in failing, and notes their parameters. Once it has the prompt,
// it brings plan with exitPlanMode.request, and passes the client's answer to answers.
type fakeCLI struct {
	plan, failing string
	answers       chan jsontext.Value

	mu    sync.Mutex
	asked map[string]jsontext.Value
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

		if err := f.answer(conn, message); err != nil {
			return
		}
	}
}

// answer notes request and answers it. After the prompt, it brings the plan.
func (f *fakeCLI) answer(out io.Writer, request incoming) error {
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

	if err := write(out, reply); err != nil || reply.Error != nil {
		return err
	}

	if request.Method != "session.send" {
		return nil
	}

	return write(out, f.bring(named.SessionID))
}

// bring returns the request that brings the plan to the client of session id.
func (f *fakeCLI) bring(id string) outgoing {
	request := exitRequest{
		SessionID: id, Summary: "A plan", PlanContent: f.plan,
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

// result returns the CLI's answer to a request for method in session.
func result(method string, named session) any {
	switch method {
	case "connect":
		return map[string]any{"ok": true, "protocolVersion": 3, "version": "1.0.105"}
	case "session.create":
		return map[string]string{"sessionId": named.SessionID, "workspacePath": "/workspace"}
	case "session.send":
		return map[string]string{"messageId": "message-1"}
	case "session.detach":
		return map[string]bool{"success": true}
	default:
		return map[string]any{}
	}
}

// read reads a message from its Content-Length frame.
func read(reader *textproto.Reader) (incoming, error) {
	var message incoming

	header, err := reader.ReadMIMEHeader()
	if err != nil {
		return message, err
	}

	length, err := strconv.Atoi(header.Get("Content-Length"))
	if err != nil {
		return message, err
	}

	body := make([]byte, length)
	if _, err := io.ReadFull(reader.R, body); err != nil {
		return message, err
	}

	err = json.Unmarshal(body, &message)

	return message, err
}

// write writes message in a Content-Length frame.
func write(out io.Writer, message outgoing) error {
	message.JSONRPC = "2.0"

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "Content-Length: %d\r\n\r\n%s", len(body), body)

	return err
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
