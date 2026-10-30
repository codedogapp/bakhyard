package copilot

import (
	"strings"

	sdk "github.com/github/copilot-sdk/go"
)

// askUser is the tool with which the agent asks the user a question.
const askUser = "ask_user"

// What the CLI puts before the user's answer in ask_user's result: for one of the choices, and
// for an answer of the user's own.
const (
	userSelected  = "User selected: "
	userResponded = "User responded: "
)

// askCalls are the questions of the agent's ask_user calls that have started and not yet ended,
// by the call's ID.
type askCalls map[string]Message

// see takes in event, and returns the messages that it brings of the agent's questions. A
// question joins the messages only as its call ends, with the user's answer if it got one, since
// until then it waits among the plan's questions.
func (c askCalls) see(event sdk.SessionEvent) []Message {
	switch data := event.Data.(type) {
	case *sdk.ToolExecutionStartData:
		if question, ok := questionAsked(event.ID, data); ok {
			c[data.ToolCallID] = question
		}
	case *sdk.ToolExecutionCompleteData:
		return c.end(event.ID, data)
	}

	return nil
}

// end returns the question of the ask_user call that data ends, if it's one of c, then the
// user's answer, if it got one, as the message answerID.
func (c askCalls) end(answerID string, data *sdk.ToolExecutionCompleteData) []Message {
	question, ok := c[data.ToolCallID]
	if !ok {
		return nil
	}

	delete(c, data.ToolCallID)

	answer, ok := answerOf(data)
	if !ok {
		return []Message{question}
	}

	return []Message{question, {ID: answerID, From: User, Text: answer}}
}

// questionAsked returns the question of the tool call that data starts, as the assistant's
// message id, if it's an ask_user call with a question.
func questionAsked(id string, data *sdk.ToolExecutionStartData) (Message, bool) {
	arguments, _ := data.Arguments.(map[string]any)
	text, ok := arguments["question"].(string)

	return Message{ID: id, From: Assistant, Text: text}, ok && data.ToolName == askUser
}

// answerOf returns the user's answer in ask_user's result, which data brings, if it has one:
// none when the user skipped the question or the call failed.
func answerOf(data *sdk.ToolExecutionCompleteData) (string, bool) {
	if data.Result == nil {
		return "", false
	}

	if answer, ok := strings.CutPrefix(data.Result.Content, userSelected); ok {
		return answer, true
	}

	return strings.CutPrefix(data.Result.Content, userResponded)
}
