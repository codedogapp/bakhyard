package copilot

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
)

// incoming is a JSON-RPC 2.0 message from the client: a request if it has a method, else an
// answer to the fake's request.
type incoming struct {
	ID     jsontext.Value `json:"id"`
	Method string         `json:"method"`
	Params jsontext.Value `json:"params"`
	Result jsontext.Value `json:"result"`
}

// outgoing is a JSON-RPC 2.0 message from the fake: a notification if it has a method but no ID.
type outgoing struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      jsontext.Value `json:"id,omitzero"`
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
