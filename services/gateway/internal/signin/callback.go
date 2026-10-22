package signin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// check returns the email in Google's answer to r if it's the user's own and verified, or else
// why not. Its errors never carry the code, the tokens or the secret.
func (f *Flow) check(r *http.Request) (string, error) {
	query := r.URL.Query()

	login, ok := f.logins.take(query.Get("state"))
	if !ok {
		return "", errors.New("unknown, used or expired state")
	}

	if code := query.Get("error"); code != "" {
		return "", fmt.Errorf("google sent the error %q", code)
	}

	token, err := f.exchange(r.Context(), query.Get("code"), login.verifier)
	if err != nil {
		return "", err
	}

	// go-oidc leaves the nonce to the caller.
	if token.Nonce != login.nonce {
		return "", errors.New("wrong nonce")
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

// allowed returns the token's email if it's verified and exactly the user's.
func (f *Flow) allowed(token *oidc.IDToken) (string, error) {
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
	}

	err := token.Claims(&claims)
	if err != nil {
		return "", fmt.Errorf("read the ID token's claims: %w", err)
	}

	if !claims.Verified {
		return "", fmt.Errorf("unverified email %s", claims.Email)
	}

	if claims.Email != f.email {
		return "", fmt.Errorf("someone else's email %s", claims.Email)
	}

	return claims.Email, nil
}
