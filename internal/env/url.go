package env

import (
	"fmt"
	"net"
	"net/url"
)

// SecureURL returns the URL in the setting name, to which a service sends a token. It must be an
// https URL, so that the token doesn't travel in the clear, or an http one to a loopback IP
// address, for a stand-in in tests.
func SecureURL(getenv func(string) string, name string) (*url.URL, error) {
	value, err := Required(getenv, name)
	if err != nil {
		return nil, err
	}

	address, err := url.Parse(value)
	if err != nil || address.Host == "" || !secure(address) {
		return nil, fmt.Errorf("%s: %q is not an https URL", name, value)
	}

	return address, nil
}

// secure reports whether address keeps a token on the way to it from other machines' eyes.
func secure(address *url.URL) bool {
	switch address.Scheme {
	case "https":
		return true
	case "http":
		return net.ParseIP(address.Hostname()).IsLoopback()
	default:
		return false
	}
}
