package env_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/internal/env"
)

func TestVars(t *testing.T) {
	tests := []struct {
		name  string
		peers []env.Peer
		want  string
	}{
		{"no peers", nil, "BAKHYARD_PEERS="},
		{
			"peers in the order given",
			[]env.Peer{
				{Name: "web", Listen: "tcp://[::1]:8082"},
				{Name: "home", Listen: "tcp://127.0.0.1:8081"},
			},
			"BAKHYARD_PEERS=web=tcp://[::1]:8082,home=tcp://127.0.0.1:8081",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := env.Vars("tcp://127.0.0.1:8080", test.peers)

			want := []string{"BAKHYARD_LISTEN=tcp://127.0.0.1:8080", test.want}
			if !slices.Equal(got, want) {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestAServiceReadsWhatTheCockpitWrites(t *testing.T) {
	getenv := getenvOf(env.Vars("tcp://127.0.0.1:8080", []env.Peer{
		{Name: "home", Listen: "tcp://127.0.0.1:8081"},
		{Name: "web", Listen: "tcp://[::1]:8082"},
	})...)

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

func TestPeerAddressRejects(t *testing.T) {
	tests := []struct {
		name, peers, want string
	}{
		{"no peers", "", `BAKHYARD_PEERS has no peer "home"`},
		{"other peers only", "web=tcp://127.0.0.1:8082", `BAKHYARD_PEERS has no peer "home"`},
		{
			"an entry without a name",
			"web=tcp://127.0.0.1:8082,tcp://127.0.0.1:8081",
			`BAKHYARD_PEERS: "tcp://127.0.0.1:8081" is not name=tcp://host:port`,
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr, err := env.PeerAddress(getenvOf("BAKHYARD_PEERS="+test.peers), "home")
			if err == nil || err.Error() != test.want {
				t.Errorf("got %q, %v, want the error %q", addr, err, test.want)
			}
		})
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
