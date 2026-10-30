// Package weblink keeps the links to pages on the web that the app may show, so that it never
// links a javascript: URL or the like (decision 201).
package weblink

import "net/url"

// HTTPS returns href when it's an https URL with a host, and an empty string when it isn't.
func HTTPS(href string) string {
	page, err := url.Parse(href)
	if err != nil || page.Scheme != "https" || page.Host == "" {
		return ""
	}

	return href
}
