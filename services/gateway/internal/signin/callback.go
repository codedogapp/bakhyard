package signin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// account is a Google account in an ID token: its email, and its sub, which Google never changes
// or reuses.
type account struct {
	email, sub string
}

// check returns the account in Google's answer to r if its email is the user's own and verified,
// or else why not. Its errors never carry the code, the tokens or the secret.
func (f *Flow) check(r *http.Request) (account, error) {
	query := r.URL.Query()

	login, ok := f.logins.take(query.Get("state"))
	if !ok {
		return account{}, errors.New("unknown, used or expired state")
	}

	if code := query.Get("error"); code != "" {
		return account{}, fmt.Errorf("google sent the error %q", code)
	}

	token, err := f.exchange(r.Context(), query.Get("code"), login.verifier)
	if err != nil {
		return account{}, err
	}

	// go-oidc leaves the nonce to the caller.
	if token.Nonce != login.nonce {
		return account{}, errors.New("wrong nonce")
	}

	return f.allowed(token)
}

// exchange trades code for Google's tokens and returns the ID token once it's verified: its
// signature, issuer, audience and expiry.
func (f *Flow) exchange(ctx context.Context, code, verifier string) (*oidc.IDToken, error) {
	ctx = oidc.ClientContext(ctx, f.client)

	token, err := f.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchange the code: %w", err)
	}

	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no ID token in Google's answer")
	}

	idToken, err := f.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("verify the ID token: %w", err)
	}

	return idToken, nil
}

// allowed returns the token's account if its email is verified and exactly the user's, and it
// has a sub.
func (f *Flow) allowed(token *oidc.IDToken) (account, error) {
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}

	err := token.Claims(&claims)
	if err != nil {
		return account{}, fmt.Errorf("read the ID token's claims: %w", err)
	}

	if !claims.Verified {
		return account{}, fmt.Errorf("unverified email %s", claims.Email)
	}

	if claims.Email != f.email {
		return account{}, fmt.Errorf("someone else's email %s", claims.Email)
	}

	// go-oidc doesn't check that the sub is there.
	if token.Subject == "" {
		return account{}, errors.New("no sub in the ID token")
	}

	return account{email: claims.Email, sub: token.Subject}, nil
}
