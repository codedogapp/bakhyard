package env_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/env"
)

func TestAServiceReadsItsEnvironment(t *testing.T) {
	getenv := getenvOf(
		"BAKHYARD_LISTEN=tcp://127.0.0.1:8080",
		"BAKHYARD_PEERS=home=tcp://127.0.0.1:8081,web=tcp://[::1]:8082",
	)

	addr, err := env.ListenAddress(getenv)
	if err != nil || addr != "127.0.0.1:8080" {
		t.Errorf("listen address: got %q, %v, want 127.0.0.1:8080", addr, err)
	}

	for name, want := range map[string]string{"home": "127.0.0.1:8081", "web": "[::1]:8082"} {
		addr, err := env.PeerAddress(getenv, name)
		if err != nil || addr != want {
			t.Errorf("peer %s: got %q, %v, want %s", name, addr, err, want)
		}
	}
}

func TestRequiredReturnsTheValue(t *testing.T) {
	value, err := env.Required(getenvOf("BAKHYARD_HOST=machine.tailnet.ts.net"), "BAKHYARD_HOST")
	if err != nil || value != "machine.tailnet.ts.net" {
		t.Errorf("got %q, %v, want machine.tailnet.ts.net", value, err)
	}
}

func TestRequiredRejects(t *testing.T) {
	tests := map[string]func(string) string{
		"an unset variable": getenvOf(),
		"an empty variable": getenvOf("BAKHYARD_HOST="),
	}

	for name, getenv := range tests {
		t.Run(name, func(t *testing.T) {
			value, err := env.Required(getenv, "BAKHYARD_HOST")
			if err == nil || err.Error() != "BAKHYARD_HOST is not set" {
				t.Errorf("got %q, %v, want the error BAKHYARD_HOST is not set", value, err)
			}
		})
	}
}

func TestListenAddressRejects(t *testing.T) {
	tests := []struct {
		name, listen, want string
	}{
		{"no address", "", "BAKHYARD_LISTEN is not set"},
		{
			"an address without the scheme",
			"127.0.0.1:8080",
			`BAKHYARD_LISTEN: "127.0.0.1:8080" is not tcp://host:port`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr, err := env.ListenAddress(getenvOf("BAKHYARD_LISTEN=" + test.listen))
			if err == nil || err.Error() != test.want {
				t.Errorf("got %q, %v, want the error %q", addr, err, test.want)
			}
		})
	}
}

func TestPeersListsThePeersInOrder(t *testing.T) {
	peers, err := env.Peers(
		getenvOf("BAKHYARD_PEERS=todos=tcp://127.0.0.1:8084,pulls=tcp://[::1]:8085"),
	)

	want := []env.Peer{
		{Name: "todos", Address: "127.0.0.1:8084"},
		{Name: "pulls", Address: "[::1]:8085"},
	}
	if err != nil || !slices.Equal(peers, want) {
		t.Errorf("got %v, %v, want %v", peers, err, want)
	}
}

func TestPeersRejects(t *testing.T) {
	tests := append(badPeers(), badPeer{"no peers", "", "BAKHYARD_PEERS is not set"})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			peers, err := env.Peers(getenvOf("BAKHYARD_PEERS=" + test.peers))
			if err == nil || err.Error() != test.want {
				t.Errorf("got %v, %v, want the error %q", peers, err, test.want)
			}
		})
	}
}

func TestPeerAddressRejects(t *testing.T) {
	tests := append(
		badPeers(),
		badPeer{"no peers", "", `BAKHYARD_PEERS has no peer "home"`},
		badPeer{
			"other peers only",
			"web=tcp://127.0.0.1:8082",
			`BAKHYARD_PEERS has no peer "home"`,
		},
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr, err := env.PeerAddress(getenvOf("BAKHYARD_PEERS="+test.peers), "home")
			if err == nil || err.Error() != test.want {
				t.Errorf("got %q, %v, want the error %q", addr, err, test.want)
			}
		})
	}
}

// badPeer is a value of BAKHYARD_PEERS that a reader rejects, and the error it gives.
type badPeer struct {
	name, peers, want string
}

// badPeers returns the values of BAKHYARD_PEERS that every reader rejects, whichever peer it asks
// for: the tests ask for home.
func badPeers() []badPeer {
	return []badPeer{
		{
			"an entry without a name",
			"web=tcp://127.0.0.1:8082,tcp://127.0.0.1:8081",
			`BAKHYARD_PEERS: "tcp://127.0.0.1:8081" is not name=tcp://host:port`,
		},
		{
			"an empty name",
			"=tcp://127.0.0.1:8081",
			`BAKHYARD_PEERS: "=tcp://127.0.0.1:8081" is not name=tcp://host:port`,
		},
		{
			"an empty entry",
			"web=tcp://127.0.0.1:8082,",
			`BAKHYARD_PEERS: "" is not name=tcp://host:port`,
		},
		{
			"a bad address",
			"home=127.0.0.1:8081",
			`BAKHYARD_PEERS: peer home: "127.0.0.1:8081" is not tcp://host:port`,
		},
		{
			"another peer's bad address",
			"home=tcp://127.0.0.1:8081,web=8082",
			`BAKHYARD_PEERS: peer web: "8082" is not tcp://host:port`,
		},
		{
			"a peer listed twice",
			"home=tcp://127.0.0.1:8081,home=tcp://127.0.0.1:8083",
			"BAKHYARD_PEERS: peer home is listed twice",
		},
	}
}

func TestAddressAccepts(t *testing.T) {
	tests := map[string]string{
		"tcp://127.0.0.1:8080":  "127.0.0.1:8080",
		"tcp://localhost:65535": "localhost:65535",
		"tcp://[::1]:1":         "[::1]:1",
	}

	for listen, want := range tests {
		got, err := env.Address(listen)
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v, want %q", listen, got, err, want)
		}
	}
}

func TestAddressRejects(t *testing.T) {
	tests := []string{
		"",
		"127.0.0.1:8080",
		"unix:///tmp/home.sock",
		"TCP://127.0.0.1:8080",
		"tcp://127.0.0.1",
		"tcp://:8080",
		"tcp://127.0.0.1:",
		"tcp://127.0.0.1:http",
		"tcp://127.0.0.1:0",
		"tcp://127.0.0.1:65536",
		"tcp://127.0.0.1:-1",
		"tcp://127.0.0.1:8080/",
		"tcp://::1:8080",
	}

	for _, listen := range tests {
		got, err := env.Address(listen)

		want := fmt.Sprintf("%q is not tcp://host:port", listen)
		if err == nil || err.Error() != want {
			t.Errorf("got %q, %v, want the error %q", got, err, want)
		}
	}
}

// getenvOf returns a getenv that knows only the variables given, as KEY=value.
func getenvOf(vars ...string) func(string) string {
	values := make(map[string]string, len(vars))

	for _, variable := range vars {
		key, value, _ := strings.Cut(variable, "=")
		values[key] = value
	}

	return func(key string) string {
		return values[key]
	}
}
