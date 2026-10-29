package jira_test

import (
	"testing"

	"github.com/codedogapp/bakhyard/internal/jira"
)

func TestIsKeyTakesAnyJiraKey(t *testing.T) {
	for _, key := range []string{
		"AB-1", "ABC-10", "A1-2", "A_B-3", "Z9_-2026", "ABCDEFGHIJ-99999",
	} {
		if !jira.IsKey(key) {
			t.Errorf("%q: got false, want true", key)
		}
	}
}

func TestIsKeyRefusesTheRest(t *testing.T) {
	for _, text := range []string{
		"", ".", "..", "abc-1", "Abc-1", "aBC-1", "A-1", "1AB-2", "_AB-2", "AB-0", "AB-01", "AB--1",
		"AB-1-2", "AB_1", "AB-", "AB", "-1", " AB-1", "AB-1 ", "AB-1\n", "AB 1", "ÄB-1", "AB-١",
		"AB-1١", "AB-1x", "AB-1/x", "../AB-1", "AB-1%2F",
	} {
		if jira.IsKey(text) {
			t.Errorf("%q: got true, want false", text)
		}
	}
}
