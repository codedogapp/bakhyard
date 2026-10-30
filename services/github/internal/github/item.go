package github

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Item is an open pull request or issue on GitHub.
type Item struct {
	// ID is the pull request's or issue's number, unique only within its repository.
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Repo is the full name of the item's repository, such as acme/web.
	Repo string `json:"repo"`
	// URL is the item's page on GitHub, and is empty when GitHub's link to it isn't an https URL.
	URL string `json:"url,omitempty"`
	// Draft says whether the item is a draft pull request.
	Draft   bool      `json:"draft"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// issue is the part of an issue or pull request in GitHub's search results that the client reads.
type issue struct {
	// ID is unique across GitHub, unlike Number.
	ID            int64     `json:"id"`
	Number        int       `json:"number"`
	Title         string    `json:"title"`
	RepositoryURL string    `json:"repository_url"`
	HTMLURL       string    `json:"html_url"`
	Draft         bool      `json:"draft"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// itemsOf returns the Items of the issues and pull requests found, in their order, each once: an
// item that moves between two pages while the client reads them shows up on both.
func itemsOf(found []issue) ([]Item, error) {
	items := []Item{}
	seen := map[int64]bool{}

	for _, each := range found {
		if seen[each.ID] {
			continue
		}

		seen[each.ID] = true

		item, err := each.item()
		if err != nil {
			return nil, fmt.Errorf("read GitHub's answer: %w", err)
		}

		items = append(items, item)
	}

	return items, nil
}

// item returns the Item of an issue or pull request in GitHub's answer.
func (found issue) item() (Item, error) {
	repo, err := repoOf(found.RepositoryURL)
	if err != nil {
		return Item{}, err
	}

	return Item{
		ID:      found.Number,
		Title:   found.Title,
		Repo:    repo,
		URL:     webPage(found.HTMLURL),
		Draft:   found.Draft,
		Created: found.CreatedAt.UTC(),
		Updated: found.UpdatedAt.UTC(),
	}, nil
}

// repoOf returns the full name of the repository at address in GitHub's API, such as acme/web for
// https://api.github.com/repos/acme/web. Search results hold no other trace of it.
func repoOf(address string) (string, error) {
	found, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("no repository in %q", address)
	}

	// The path ends in /repos/{owner}/{name}, after /api/v3 on GitHub Enterprise Server.
	rest, name, _ := strings.CutLast(found.Path, "/")
	rest, owner, _ := strings.CutLast(rest, "/")

	if !strings.HasSuffix(rest, "/repos") || owner == "" || name == "" {
		return "", fmt.Errorf("no repository in %q", address)
	}

	return owner + "/" + name, nil
}

// webPage returns href when it's an https URL, and an empty string when it isn't, so that the
// app never links a javascript: URL or the like.
func webPage(href string) string {
	page, err := url.Parse(href)
	if err != nil || page.Scheme != "https" || page.Host == "" {
		return ""
	}

	return href
}
