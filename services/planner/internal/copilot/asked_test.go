package copilot

import (
	"reflect"
	"testing"

	sdk "github.com/github/copilot-sdk/go"
)

// calling returns the event id that starts the call callID of tool, with arguments as the SDK
// decodes them.
func calling(id, callID, tool string, arguments any) sdk.SessionEvent {
	data := &sdk.ToolExecutionStartData{ToolCallID: callID, ToolName: tool, Arguments: arguments}

	return sdk.SessionEvent{ID: id, Data: data}
}

// ended returns the event id that ends the call callID with the result content.
func ended(id, callID, content string) sdk.SessionEvent {
	result := &sdk.ToolExecutionCompleteResult{Content: content, DetailedContent: new(content)}
	data := &sdk.ToolExecutionCompleteData{ToolCallID: callID, Success: true, Result: result}

	return sdk.SessionEvent{ID: id, Data: data}
}

func TestQuestionsAsked(t *testing.T) {
	failed := &sdk.ToolExecutionCompleteData{
		ToolCallID: "c5",
		Error:      &sdk.ToolExecutionCompleteError{Message: "Tool execution was interrupted"},
	}
	events := []sdk.SessionEvent{
		calling("s1", "c1", "ask_user", map[string]any{
			"question": "Which city?", "choices": []any{"Rome", "Paris"},
		}),
		calling("s2", "c2", "view", map[string]any{"question": "Which file?"}),
		ended("e2", "c2", "User selected: README.md"),
		ended("e1", "c1", "User selected: Rome"),
		ended("e1b", "c1", "User selected: Paris"),
		calling(
			"s3",
			"c3",
			"ask_user",
			map[string]any{"question": "Budget?", "allow_freeform": true},
		),
		ended("e3", "c3", "User responded: About 2000 euros,\nfor two"),
		calling("s4", "c4", "ask_user", map[string]any{"question": "Hotel or flat?"}),
		ended("e4", "c4", "User skipped question"),
		calling("s5", "c5", "ask_user", map[string]any{"question": "When?"}),
		{ID: "e5", Data: failed},
		calling("s6", "c6", "ask_user", map[string]any{"message": "How long?"}),
		ended("e6", "c6", "User responded: A week"),
		calling("s7", "c7", "ask_user", "How many?"),
		ended("e7", "c7", "User responded: Two"),
		calling("s8", "c8", "ask_user", map[string]any{"question": "Which month?"}),
		ended("e9", "c9", "User selected: May"),
	}

	want := []Message{
		{ID: "s1", From: Assistant, Text: "Which city?"},
		{ID: "e1", From: User, Text: "Rome"},
		{ID: "s3", From: Assistant, Text: "Budget?"},
		{ID: "e3", From: User, Text: "About 2000 euros,\nfor two"},
		{ID: "s4", From: Assistant, Text: "Hotel or flat?"},
		{ID: "s5", From: Assistant, Text: "When?"},
	}
	if got := messages(events); !reflect.DeepEqual(got, want) {
		t.Errorf("messages = %+v, want %+v", got, want)
	}
}

func TestSeeTellsTheQuestionsAsked(t *testing.T) {
	planned := newPlan(nil)
	updates, _ := planned.follow()

	planned.see(calling("s1", "c1", "ask_user", map[string]any{"question": "Which city?"}))

	if len(updates) != 0 {
		t.Errorf("%d updates as the question is asked, want none while it waits", len(updates))
	}

	planned.see(ended("e1", "c1", "User responded: Florence"))
	planned.unfollow(updates)

	want := []Update{
		{Change: ChangeMessage, Message: Message{ID: "s1", From: Assistant, Text: "Which city?"}},
		{Change: ChangeMessage, Message: Message{ID: "e1", From: User, Text: "Florence"}},
	}
	if got := closed(t, updates); !reflect.DeepEqual(got, want) {
		t.Errorf("updates = %+v, want %+v", got, want)
	}
}
