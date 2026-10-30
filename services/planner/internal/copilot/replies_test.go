package copilot

import "encoding/json/jsontext"

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

// earlier is the plan of an earlier run that the fake keeps, besides those that the client starts.
const earlier = "plan-1"

// result returns the CLI's answer to a request for method in session.
func (f *fakeCLI) result(method string, named session) any {
	switch method {
	case "connect":
		return map[string]any{"ok": true, "protocolVersion": 3, "version": "1.0.105"}
	case "session.create":
		return map[string]string{"sessionId": named.SessionID, "workspacePath": "/workspace"}
	case "session.getMetadata":
		return metadata(named)
	case "session.resume":
		return map[string]string{"sessionId": named.SessionID, "workspacePath": f.workspace}
	case "session.send":
		return map[string]string{"messageId": "message-1"}
	case "session.getMessages":
		return jsontext.Value(record)
	case "session.list":
		return jsontext.Value(listed)
	case "session.detach":
		return map[string]bool{"success": true}
	default:
		return map[string]any{}
	}
}

// metadata returns the CLI's answer to a request for the metadata of the session named, which
// has none unless it's the plan of an earlier run.
func metadata(named session) any {
	if named.SessionID != earlier {
		return map[string]any{}
	}

	return map[string]any{"session": map[string]any{"sessionId": earlier, "isRemote": false}}
}
