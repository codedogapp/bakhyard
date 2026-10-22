package main

import (
	"strings"
	"testing"
)

func TestTheGatewayStartsAgainOnItsMigratedDatabase(t *testing.T) {
	vars, items := gatewayEnv(t, "http://127.0.0.1:2")

	// The second start finds the migrations done; running one again would fail.
	for range 2 {
		_, err := routes(func(key string) string { return vars[key] }, keychainWith(items))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheGatewayNeedsTheDatabasePassword(t *testing.T) {
	vars, items := gatewayEnv(t, "http://127.0.0.1:2")
	delete(items, "postgres-gateway")

	_, err := routes(func(key string) string { return vars[key] }, keychainWith(items))

	want := "read the database password: no Keychain item bakhyard/postgres-gateway"
	if err == nil || err.Error() != want {
		t.Errorf("got %v, want the error %s", err, want)
	}
}

func TestTheGatewayNeedsItsDatabase(t *testing.T) {
	vars, items := gatewayEnv(t, "http://127.0.0.1:2")
	// Nothing listens on port 1.
	vars["BAKHYARD_DATABASE"] = "postgres://bakhyard_gateway@127.0.0.1:1/bakhyard_gateway"

	_, err := routes(func(key string) string { return vars[key] }, keychainWith(items))
	if err == nil || !strings.HasPrefix(err.Error(), "reach the database: ") {
		t.Errorf("got %v, want an error that starts with reach the database", err)
	}
}
