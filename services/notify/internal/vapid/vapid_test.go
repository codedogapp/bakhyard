package vapid_test

import (
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/notify/internal/vapid"
)

func TestGenerateMakesAKeyThatParses(t *testing.T) {
	private, err := vapid.Generate()
	if err != nil {
		t.Fatal(err)
	}

	key, err := vapid.Parse(private + "\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if key.Private != private || key.Public == "" || key.Public == private {
		t.Errorf("got %+v for the private key %q", key, private)
	}
}

func TestParseRefusesWhatIsNotAKey(t *testing.T) {
	for name, stored := range map[string]string{
		"empty":     "",
		"not b64":   "not base64 !",
		"too short": "AAAA",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := vapid.Parse(
				stored,
			); err == nil ||
				!strings.Contains(err.Error(), "VAPID") {
				t.Errorf("got %v, want an error about the VAPID key", err)
			}
		})
	}
}
