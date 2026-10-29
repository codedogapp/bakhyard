package bitbucket

import (
	"net/url"
	"time"
)

// Pull is an open pull request: one of the user's, or one that waits for the user's review.
type Pull struct {
	// ID is the pull request's number, unique only within its repository.
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Repo is the full name of the repository the pull request merges into, such as acme/web.
	Repo string `json:"repo"`
	// URL is the pull request's page on Bitbucket, and is empty when Bitbucket's link to it isn't
	// an https URL.
	URL     string    `json:"url,omitempty"`
	Draft   bool      `json:"draft"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// pullRequest is the part of a pull request in Bitbucket's answer that the client reads.
type pullRequest struct {
	ID          int         `json:"id"`
	Title       string      `json:"title"`
	Draft       bool        `json:"draft"`
	CreatedOn   time.Time   `json:"created_on"`
	UpdatedOn   time.Time   `json:"updated_on"`
	Destination destination `json:"destination"`
	Links       links       `json:"links"`
}

// destination is the branch that a pull request merges into, in Bitbucket's answer.
type destination struct {
	Repository repository `json:"repository"`
}

type repository struct {
	FullName string `json:"full_name"`
}

// links are a pull request's links in Bitbucket's answer.
type links struct {
	HTML link `json:"html"`
}

type link struct {
	Href string `json:"href"`
}

// pull returns the Pull of a pull request in Bitbucket's answer.
func (found pullRequest) pull() Pull {
	return Pull{
		ID:      found.ID,
		Title:   found.Title,
		Repo:    found.Destination.Repository.FullName,
		URL:     webPage(found.Links.HTML.Href),
		Draft:   found.Draft,
		Created: found.CreatedOn.UTC(),
		Updated: found.UpdatedOn.UTC(),
	}
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
