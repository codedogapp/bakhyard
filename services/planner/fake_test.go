package main

import (
	"bufio"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestMain(m *testing.M) {
	// The tests run this binary as the Copilot CLI, under the name copilot.
	if filepath.Base(os.Args[0]) != "copilot" {
		m.Run()

		return
	}

	if err := fakeCopilot(os.Stdin, os.Stdout); err != nil {
		os.Exit(1)
	}
}

// request is a JSON-RPC 2.0 message from the client. A notification has no ID.
type request struct {
	ID     jsontext.Value `json:"id"`
	Method string         `json:"method"`
}

// fakeCopilot stands in for the Copilot CLI on in and out: it answers connect as CLI 1.0.105
// does, and any other request with an empty result. Once in closes, it leaves the empty file
// stopped in the folder it runs in.
func fakeCopilot(in io.Reader, out io.Writer) error {
	reader := textproto.NewReader(bufio.NewReader(in))

	for {
		message, err := readRequest(reader)
		if err != nil {
			break
		}

		if len(message.ID) == 0 {
			continue
		}

		if err := answer(out, message); err != nil {
			return err
		}
	}

	return os.WriteFile("stopped", nil, 0o600)
}

// readRequest reads a message from its Content-Length frame.
func readRequest(reader *textproto.Reader) (request, error) {
	var message request

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

	return message, json.Unmarshal(body, &message)
}

// answer answers message in a Content-Length frame.
func answer(out io.Writer, message request) error {
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, message.ID,
		result(message.Method))
	_, err := fmt.Fprintf(out, "Content-Length: %d\r\n\r\n%s", len(body), body)

	return err
}

// result returns the answer of CLI 1.0.105 to a request for method.
func result(method string) string {
	if method == "connect" {
		return `{"ok":true,"protocolVersion":3,"version":"1.0.105"}`
	}

	return "{}"
}
