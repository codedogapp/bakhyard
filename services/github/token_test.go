package main

import (
	"log/slog"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/codedogapp/bakhyard/internal/githubapi"
	"github.com/codedogapp/bakhyard/services/github/internal/github"
)

func TestGitHubRemindsToRenewTheTokenBeforeItExpires(t *testing.T) {
	expires := time.Now().Add(3 * 24 * time.Hour)

	got := listItems(t, envOf(map[string]string{githubAPIVar: githubServer(t, expires)}),
		secretsWith(map[string]string{"github-token": "secret"}))

	want := []string{
		"acme/web#1 acme/web #1",
		"acme/api#1 acme/api #1 · Review requested",
		"acme/docs#1 acme/docs #1 · Assigned",
		"github-token ",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGitHubTokenExpiresWhenGitHubSays(t *testing.T) {
	expires := time.Date(2027, time.October, 27, 22, 0, 0, 0, time.UTC)

	api, err := url.Parse(githubServer(t, expires))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name   string
		client *github.Client
		want   time.Time
	}{
		{"GitHub's answer", github.New(githubapi.Settings{API: api, Token: "secret"}), expires},
		{"a failed ask", github.New(githubapi.Settings{API: api, Token: "wrong"}), time.Time{}},
		{"GitHub off", nil, time.Time{}},
	} {
		token := githubToken(t.Context(), test.client, slog.New(slog.DiscardHandler))

		got, err := token.Expires(t.Context())
		if err != nil || !got.Equal(test.want) {
			t.Errorf("%s: got %v and %v, want %v", test.name, got, err, test.want)
		}
	}
}
