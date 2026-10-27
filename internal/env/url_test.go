package env_test

import (
	"fmt"
	"testing"

	"github.com/codedogapp/bakhyard/internal/env"
)

func TestSecureURLAccepts(t *testing.T) {
	for value, want := range map[string]string{
		"https://api.atlassian.com/ex/jira/cloud-id": "https://api.atlassian.com/ex/jira/cloud-id",
		"HTTPS://API.BITBUCKET.ORG/2.0":              "https://API.BITBUCKET.ORG/2.0",
		"http://127.0.0.1:8090":                      "http://127.0.0.1:8090",
		"http://127.0.0.1":                           "http://127.0.0.1",
		"http://127.1.2.3:8090/2.0":                  "http://127.1.2.3:8090/2.0",
		"http://[::1]:8090":                          "http://[::1]:8090",
	} {
		got, err := env.SecureURL(getenvOf("BAKHYARD_API="+value), "BAKHYARD_API")
		if err != nil || got.String() != want {
			t.Errorf("%s: got %v, %v, want %s", value, got, err, want)
		}
	}
}

func TestSecureURLRejects(t *testing.T) {
	for _, value := range []string{
		"http://example.com",
		"http://localhost:8090",
		"http://192.168.0.2:8090",
		"http://0.0.0.0:8090",
		"ftp://127.0.0.1",
		"https://",
		"https:api.atlassian.com",
		"api.atlassian.com",
		"//127.0.0.1:8090",
		"https://api.atlassian.com/%zz",
	} {
		got, err := env.SecureURL(getenvOf("BAKHYARD_API="+value), "BAKHYARD_API")

		want := fmt.Sprintf("BAKHYARD_API: %q is not an https URL", value)
		if err == nil || err.Error() != want {
			t.Errorf("got %v, %v, want the error %s", got, err, want)
		}
	}
}

func TestSecureURLNeedsTheSetting(t *testing.T) {
	for _, getenv := range []func(string) string{getenvOf(), getenvOf("BAKHYARD_API=")} {
		got, err := env.SecureURL(getenv, "BAKHYARD_API")
		if err == nil || err.Error() != "BAKHYARD_API is not set" {
			t.Errorf("got %v, %v, want the error BAKHYARD_API is not set", got, err)
		}
	}
}
