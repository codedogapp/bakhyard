package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	// maxRepositories bounds the repositories in which the client looks for the pull requests
	// that wait for the user's review, so that the polls stay within Bitbucket's rate limit
	// (decision 171). It's also the size of the page of a workspace's repositories.
	maxRepositories = 100
	// listingRepositories and listingReviews start the errors of the listings that Reviews makes.
	listingRepositories = "list the Bitbucket repositories"
	listingReviews      = "list the Bitbucket pull requests that wait for my review"
)

// workspaceRepository is the part of a repository in a workspace's list that the client reads.
type workspaceRepository struct {
	// FullName is the repository's workspace and slug, such as acme/web, as in Bitbucket's URLs.
	FullName  string    `json:"full_name"`
	UpdatedOn time.Time `json:"updated_on"`
}

// Reviews returns the open pull requests that wait for the user's review in the workspaces' 100
// latest updated repositories, the latest updated first.
func (c *Client) Reviews(ctx context.Context) ([]Pull, error) {
	uuid, err := c.me(ctx)
	if err != nil {
		return nil, err
	}

	repos, err := c.latestRepositories(ctx)
	if err != nil {
		return nil, err
	}

	query := url.Values{
		"q":       {`state="OPEN" AND reviewers.uuid="` + uuid + `"`},
		"pagelen": {strconv.Itoa(pageSize)},
	}.Encode()

	pulls := []Pull{}

	for _, repo := range repos {
		target, err := c.pullRequestsOf(repo)
		if err != nil {
			return nil, err
		}

		target.RawQuery = query

		found, err := c.pages(ctx, target, listingReviews)
		if err != nil {
			return nil, err
		}

		pulls = append(pulls, found...)
	}

	latestFirst(pulls)

	return pulls, nil
}

// latestRepositories returns the workspaces' latest updated repositories, the latest first and
// maxRepositories at most, from the first page of each workspace's repositories. Repositories
// updated at the same time keep their order.
func (c *Client) latestRepositories(ctx context.Context) ([]workspaceRepository, error) {
	var repos []workspaceRepository

	for _, workspace := range c.settings.Workspaces {
		target := c.settings.API.JoinPath("repositories", url.PathEscape(workspace))
		target.RawQuery = url.Values{
			"sort": {"-updated_on"}, "pagelen": {strconv.Itoa(maxRepositories)},
		}.Encode()

		var found page[workspaceRepository]

		err := c.get(ctx, target, listingRepositories, &found)
		if err != nil {
			return nil, err
		}

		repos = append(repos, found.Values...)
	}

	slices.SortStableFunc(repos, func(a, b workspaceRepository) int {
		return b.UpdatedOn.Compare(a.UpdatedOn)
	})

	return repos[:min(len(repos), maxRepositories)], nil
}

// pullRequestsOf returns the address of repo's pull requests, from its full name, whose workspace
// and slug the address mustn't lose.
func (c *Client) pullRequestsOf(repo workspaceRepository) (*url.URL, error) {
	workspace, slug, _ := strings.Cut(repo.FullName, "/")
	if lost(workspace) || lost(slug) {
		return nil, fmt.Errorf("%s: %q is not a repository's full name", listingRepositories,
			repo.FullName)
	}

	return c.settings.API.JoinPath(
		"repositories", url.PathEscape(workspace), url.PathEscape(slug), "pullrequests",
	), nil
}

// lost reports whether url.JoinPath would clean part out of a path, as it does "", "." and "..".
func lost(part string) bool {
	return part == "" || part == "." || part == ".."
}
