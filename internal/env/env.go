// Package env reads a service's environment (docs/plan.md §4.7): the address the service listens
// on, the addresses of the services it calls, its peers, and the settings it can't run without.
package env

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	listenVar = "BAKHYARD_LISTEN"
	peersVar  = "BAKHYARD_PEERS"
	tcpScheme = "tcp://"
)

// Required returns the value of the variable name, or an error if it is unset or empty.
func Required(getenv func(string) string, name string) (string, error) {
	value := getenv(name)
	if value == "" {
		return "", fmt.Errorf("%s is not set", name)
	}

	return value, nil
}

// ListenAddress returns the host and port to listen on, from BAKHYARD_LISTEN.
func ListenAddress(getenv func(string) string) (string, error) {
	listen, err := Required(getenv, listenVar)
	if err != nil {
		return "", err
	}

	addr, err := Address(listen)
	if err != nil {
		return "", fmt.Errorf("%s: %w", listenVar, err)
	}

	return addr, nil
}

// PeerAddress returns the host and port that the peer name listens on, from BAKHYARD_PEERS.
func PeerAddress(getenv func(string) string, name string) (string, error) {
	peers := getenv(peersVar)
	if peers == "" {
		return "", noPeer(name)
	}

	for entry := range strings.SplitSeq(peers, ",") {
		peer, listen, found := strings.Cut(entry, "=")
		if !found {
			return "", fmt.Errorf("%s: %q is not name=%shost:port", peersVar, entry, tcpScheme)
		}

		if peer == name {
			addr, err := Address(listen)
			if err != nil {
				return "", fmt.Errorf("%s: peer %s: %w", peersVar, name, err)
			}

			return addr, nil
		}
	}

	return "", noPeer(name)
}

func noPeer(name string) error {
	return fmt.Errorf("%s has no peer %q", peersVar, name)
}

// Address returns the host and port of a listen address, tcp://host:port. The port is a number
// from 1 to 65535, since peers couldn't find a service that listens on any free port.
func Address(listen string) (string, error) {
	addr, found := strings.CutPrefix(listen, tcpScheme)
	host, port, err := net.SplitHostPort(addr)
	number, portErr := strconv.ParseUint(port, 10, 16)

	if !found || err != nil || host == "" || portErr != nil || number == 0 {
		return "", fmt.Errorf("%q is not %shost:port", listen, tcpScheme)
	}

	return addr, nil
}
