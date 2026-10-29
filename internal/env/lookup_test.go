package env_test

import (
	"testing"

	"github.com/codedogapp/bakhyard/internal/env"
)

func TestLookupPeerFindsAListedPeer(t *testing.T) {
	getenv := getenvOf("BAKHYARD_PEERS=home=tcp://127.0.0.1:8081,notify=tcp://[::1]:8086")

	addr, found, err := env.LookupPeer(getenv, "notify")
	if err != nil || !found || addr != "[::1]:8086" {
		t.Errorf("got %q, %t, %v, want [::1]:8086, true", addr, found, err)
	}
}

func TestLookupPeerFindsNoPeerThatIsNotListed(t *testing.T) {
	tests := map[string]func(string) string{
		"unset":            getenvOf(),
		"empty":            getenvOf("BAKHYARD_PEERS="),
		"other peers only": getenvOf("BAKHYARD_PEERS=home=tcp://127.0.0.1:8081"),
		"a longer name":    getenvOf("BAKHYARD_PEERS=notifyd=tcp://127.0.0.1:8086"),
	}

	for name, getenv := range tests {
		t.Run(name, func(t *testing.T) {
			addr, found, err := env.LookupPeer(getenv, "notify")
			if err != nil || found || addr != "" {
				t.Errorf("got %q, %t, %v, want no peer and no error", addr, found, err)
			}
		})
	}
}

func TestLookupPeerRejects(t *testing.T) {
	for _, test := range badPeers() {
		t.Run(test.name, func(t *testing.T) {
			addr, found, err := env.LookupPeer(getenvOf("BAKHYARD_PEERS="+test.peers), "home")
			if err == nil || err.Error() != test.want || found || addr != "" {
				t.Errorf("got %q, %t, %v, want the error %q", addr, found, err, test.want)
			}
		})
	}
}
