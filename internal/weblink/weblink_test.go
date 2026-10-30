package weblink_test

import (
	"testing"

	"github.com/codedogapp/bakhyard/internal/weblink"
)

func TestHTTPSKeepsAnHTTPSURL(t *testing.T) {
	for _, href := range []string{
		"https://bitbucket.org/acme/web/pull-requests/7",
		"https://github.com/acme/web/pull/7?tab=files#diff",
		"https://ghe.example:8443/acme/web/issues/8",
		"HTTPS://github.com/acme/web/pull/7",
	} {
		if got := weblink.HTTPS(href); got != href {
			t.Errorf("%q: got %q, want it kept", href, got)
		}
	}
}

func TestHTTPSDropsAnyOtherLink(t *testing.T) {
	for _, href := range []string{
		"",
		"http://github.com/acme/web/pull/7",
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,hello",
		"https:///acme/web/pull/7",
		"https:github.com/acme/web/pull/7",
		"//github.com/acme/web/pull/7",
		"/acme/web/pull/7",
		" https://github.com/acme/web/pull/7",
		"https://github.com/acme/web/pull/7\n",
		"%zz",
	} {
		if got := weblink.HTTPS(href); got != "" {
			t.Errorf("%q: got %q, want nothing", href, got)
		}
	}
}
