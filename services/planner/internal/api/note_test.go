package api_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/codedogapp/bakhyard/services/planner/internal/api"
	"github.com/codedogapp/bakhyard/services/planner/internal/copilot"
)

// noteTarget is the route that saves the plan of plan-1 in the vault.
const noteTarget = "/planner/plans/plan-1/note"

// vaultStandIn is a stand-in for the vault: it notes the plans that it gets, and answers each
// with path, or with its failure.
type vaultStandIn struct {
	path    string
	failure error
	plans   []string
}

func (v *vaultStandIn) Save(_ context.Context, plan string) (string, error) {
	v.plans = append(v.plans, plan)

	return v.path, v.failure
}

// newSavingAPI returns the API of a planner whose plan-1 has plan, which saves plans in vault.
func newSavingAPI(plan string, vault api.Saver) plannerAPI {
	planner := &standIn{talk: copilot.Conversation{Plan: plan}}

	return newAPI(api.Door{Planner: planner, Vault: vault})
}

func TestSavingAPlanMakesItANote(t *testing.T) {
	vault := &vaultStandIn{path: "Plans/2026-10-30-kávé 1.md"}
	plan := "# Kávé\n\n1. Grind the beans.\n"

	rec := newSavingAPI(plan, vault).serve(t, http.MethodPost, noteTarget, "")
	if want := `{"path":"Plans/2026-10-30-kávé 1.md"}`; rec.Code != http.StatusCreated ||
		rec.Body.String() != want || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("got %d %q as %q, want 201 %s as application/json", rec.Code, rec.Body,
			rec.Header().Get("Content-Type"), want)
	}

	if want := []string{plan}; !slices.Equal(vault.plans, want) {
		t.Errorf("the vault got %q, want %q", vault.plans, want)
	}
}

func TestSavingAPlanIsNotFoundWithoutTheVault(t *testing.T) {
	planner := newSavingAPI("# Kávé", nil)

	rec := planner.serve(t, http.MethodPost, noteTarget, "")
	if want := "Saving to the vault is not set up\n"; rec.Code != http.StatusNotFound ||
		rec.Body.String() != want {
		t.Errorf("got %d %q, want 404 %q", rec.Code, rec.Body, want)
	}
}

func TestSavingAnUnknownPlanIsNotFound(t *testing.T) {
	vault := &vaultStandIn{}
	planner := newSavingAPI("# Kávé", vault)

	rec := planner.serve(t, http.MethodPost, "/planner/plans/plan-2/note", "")
	if rec.Code != http.StatusNotFound || rec.Body.String() != "no such plan\n" ||
		len(vault.plans) != 0 || planner.logs.Len() != 0 {
		t.Errorf("got %d %q, the vault got %q and logged %q, want 404 no such plan alone",
			rec.Code, rec.Body, vault.plans, planner.logs)
	}
}

func TestSavingAPlanWaitsForCopilotToWriteIt(t *testing.T) {
	for _, plan := range []string{"", " \n\t"} {
		vault := &vaultStandIn{}

		rec := newSavingAPI(plan, vault).serve(t, http.MethodPost, noteTarget, "")
		if want := "Copilot hasn't written the plan yet\n"; rec.Code != http.StatusConflict ||
			rec.Body.String() != want || len(vault.plans) != 0 {
			t.Errorf("%q: got %d %q and the vault got %q, want 409 %q alone", plan, rec.Code,
				rec.Body, vault.plans, want)
		}
	}
}

func TestSavingAPlanFailsWhenCopilotFails(t *testing.T) {
	vault := &vaultStandIn{}
	failing := &standIn{failure: errors.New("no connection")}
	planner := newAPI(api.Door{Planner: failing, Vault: vault})

	rec := planner.serve(t, http.MethodPost, noteTarget, "")
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" ||
		len(vault.plans) != 0 {
		t.Errorf("got %d %q and the vault got %q, want 502 Bad Gateway alone", rec.Code, rec.Body,
			vault.plans)
	}

	want := `msg="read a plan" err="no connection"`
	if logs := planner.logs.String(); !strings.Contains(logs, want) {
		t.Errorf("logged %q, want the cause", logs)
	}
}

func TestSavingAPlanFailsWhenTheVaultFails(t *testing.T) {
	failure := errors.New("notes answered 405 Method Not Allowed")
	planner := newSavingAPI("# Kávé", &vaultStandIn{failure: failure})

	rec := planner.serve(t, http.MethodPost, noteTarget, "")
	if rec.Code != http.StatusBadGateway || rec.Body.String() != "Bad Gateway\n" {
		t.Errorf("got %d %q, want 502 Bad Gateway", rec.Code, rec.Body)
	}

	want := `msg="save a plan" err="notes answered 405 Method Not Allowed"`
	if logs := planner.logs.String(); !strings.Contains(logs, want) {
		t.Errorf("logged %q, want the cause", logs)
	}
}
