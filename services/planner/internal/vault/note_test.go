package vault_test

import (
	"strings"
	"testing"
)

func TestSaveNamesTheNoteAfterThePlansTitle(t *testing.T) {
	tests := []struct {
		name string
		plan string
		slug string
	}{
		{"an H1", "# Add dark mode\n\nSteps.", "add-dark-mode"},
		{"blank lines and CRLF first", "\r\n\r\n# Fix the login\r\nSteps.", "fix-the-login"},
		{
			"apostrophes",
			"# The user's and the planner\u2019s state",
			"the-users-and-the-planners-state",
		},
		{"letters and digits of any script", "# Kávé és tea: 2 napra", "kávé-és-tea-2-napra"},
		{"punctuation", "# Plan: add `vault` (v2) — fast!", "plan-add-vault-v2-fast"},
		{
			"exactly as long as a name can be",
			"# " + strings.Repeat("abcde ", 9) + "abcdef",
			strings.Repeat("abcde-", 9) + "abcdef",
		},
		{
			"too long",
			"# Move the planner's conversations into their own store and drop the old map",
			"move-the-planners-conversations-into-their-own-store-and",
		},
		{"a first word too long", "# " + strings.Repeat("é", 70), strings.Repeat("é", 60)},
		{"no letter or digit", "# !!! ???", "plan"},
		{"no H1 first", "Steps first.\n# Add dark mode", "plan"},
		{"an H2", "## Add dark mode", "plan"},
		{"no space after the #", "#Add dark mode", "plan"},
		{"an empty H1", "#   \nSteps.", "plan"},
		{"no text", "", "plan"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, got := makingNotes(t)

			if _, err := client.Save(t.Context(), test.plan); err != nil {
				t.Fatal(err)
			}

			if path, want := (<-got).path, "Plans/2026-05-05-"+test.slug+".md"; path != want {
				t.Errorf("got %q, want %q", path, want)
			}
		})
	}
}

func TestSaveWritesTheFrontmatterAboveThePlan(t *testing.T) {
	tests := []struct {
		name  string
		plan  string
		title string
	}{
		{"an H1", "# Add dark mode\n\n1. Add a toggle.\n", `"Add dark mode"`},
		{"blank lines and CRLF first", "\r\n# Fix the login\r\nSteps.\r\n", `"Fix the login"`},
		{"quotes and a backslash", `# Say "hi" \ bye`, `"Say \"hi\" \\ bye"`},
		{"control characters", "# One\tTwo\rThree", `"One\tTwo\rThree"`},
		{"letters of any script", "# Kávé és tea", `"Kávé és tea"`},
		{"no H1", "Steps.", `"Plan"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, got := makingNotes(t)

			if _, err := client.Save(t.Context(), test.plan); err != nil {
				t.Fatal(err)
			}

			want := "---\ntitle: " + test.title + "\ntags:\n  - \"plan\"\ncreated: 2026-05-05\n" +
				"---\n\n" + test.plan
			if body := (<-got).body; body != want {
				t.Errorf("got %q, want %q", body, want)
			}
		})
	}
}
