// Package covers keeps the covers found for the games' titles, each for a week at most
// (docs/plan.md decisions 68, 300 and 306).
package covers

import (
	"context"
	"time"
)

// keepFor is how long a title's cover is kept: a week, well within the 30 days for which IGDB
// keeps an image that it has removed or replaced.
const keepFor = 7 * 24 * time.Hour

// Finder finds the address of a title's cover, or "" when the title has none.
type Finder interface {
	Cover(ctx context.Context, title string) (string, error)
}

// Cache keeps the covers that its Finder finds for the titles of the latest list. Its Find must
// not run while another runs, as the games' fetches don't.
type Cache struct {
	finder  Finder
	matches map[string]match
}

// match is a title's cover, or "" when it has none, and when the Finder was asked for it.
type match struct {
	cover string
	asked time.Time
}

// New returns a Cache, with nothing kept yet, of the covers that finder finds.
func New(finder Finder) *Cache {
	return &Cache{finder: finder, matches: map[string]match{}}
}

// Find returns the cover of each title in titles, in order: the one kept when the Finder was asked
// for it less than a week ago, or else the one that it finds now. It asks for the titles one after
// another, each once, and then keeps the covers of these titles only. At the Finder's first error
// it asks no more, and returns the error with the covers it has, "" for the others.
func (c *Cache) Find(ctx context.Context, titles []string) ([]string, error) {
	matches := make(map[string]match, len(titles))

	var err error

	for _, title := range titles {
		found, ok := c.kept(title, matches)
		if !ok && err == nil {
			found, err = c.ask(ctx, title)
			ok = err == nil
		}

		if ok {
			matches[title] = found
		}
	}

	c.matches = matches

	covers := make([]string, len(titles))
	for i, title := range titles {
		covers[i] = matches[title].cover
	}

	return covers, err
}

// kept returns title's match from matches, those of the Find that runs, or else from the last
// Find's when the Finder was asked for it less than a week ago.
func (c *Cache) kept(title string, matches map[string]match) (match, bool) {
	if found, ok := matches[title]; ok {
		return found, true
	}

	found, ok := c.matches[title]

	return found, ok && time.Since(found.asked) < keepFor
}

// ask asks the Finder for title's cover.
func (c *Cache) ask(ctx context.Context, title string) (match, error) {
	asked := time.Now()
	cover, err := c.finder.Cover(ctx, title)

	return match{cover: cover, asked: asked}, err
}
