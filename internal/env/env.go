// Package env reads a service's environment (docs/plan.md §4.7): the address the service listens
// on, the addresses of the services it calls, its peers, and the settings it can't run without.
package env

import (
	"fmt"
	"net"
	"slices"
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

// Peer is a service that this service calls.
type Peer struct {
	Name string
	// Address is the host and port the peer listens on.
	Address string
}

// Peers returns every peer in BAKHYARD_PEERS, in the order it lists them.
func Peers(getenv func(string) string) ([]Peer, error) {
	value, err := Required(getenv, peersVar)
	if err != nil {
		return nil, err
	}

	return parsePeers(value)
}

// PeerAddress returns the host and port that the peer name listens on, from BAKHYARD_PEERS.
func PeerAddress(getenv func(string) string, name string) (string, error) {
	addr, found, err := LookupPeer(getenv, name)
	if err == nil && !found {
		err = fmt.Errorf("%s has no peer %q", peersVar, name)
	}

	return addr, err
}

// LookupPeer returns the host and port that the peer name listens on, from BAKHYARD_PEERS, and
// whether it lists name, for a service that can run without that peer. An unset BAKHYARD_PEERS
// lists no peer.
func LookupPeer(getenv func(string) string, name string) (string, bool, error) {
	peers, err := parsePeers(getenv(peersVar))
	if err != nil {
		return "", false, err
	}

	i := slices.IndexFunc(peers, func(peer Peer) bool { return peer.Name == name })
	if i < 0 {
		return "", false, nil
	}

	return peers[i].Address, true, nil
}

// parsePeers reads every entry of BAKHYARD_PEERS's value, name=tcp://host:port,…, so that a
// mistake in any entry stops the service, whichever peers it asks for.
func parsePeers(value string) ([]Peer, error) {
	if value == "" {
		return nil, nil
	}

	var peers []Peer

	for entry := range strings.SplitSeq(value, ",") {
		name, listen, found := strings.Cut(entry, "=")
		if !found || name == "" {
			return nil, fmt.Errorf("%s: %q is not name=%shost:port", peersVar, entry, tcpScheme)
		}

		addr, err := Address(listen)
		if err != nil {
			return nil, fmt.Errorf("%s: peer %s: %w", peersVar, name, err)
		}

		if slices.ContainsFunc(peers, func(peer Peer) bool { return peer.Name == name }) {
			return nil, fmt.Errorf("%s: peer %s is listed twice", peersVar, name)
		}

		peers = append(peers, Peer{Name: name, Address: addr})
	}

	return peers, nil
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
