package signin

import "github.com/coreos/go-oidc/v3/oidc"

// Google returns Google's endpoints, from its discovery document at
// https://accounts.google.com/.well-known/openid-configuration. They're fixed here, so the gateway
// starts without reaching Google.
func Google() oidc.ProviderConfig {
	//nolint:gosec // G101: TokenURL is Google's public token endpoint, not a credential.
	return oidc.ProviderConfig{
		IssuerURL:  "https://accounts.google.com",
		AuthURL:    "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:   "https://oauth2.googleapis.com/token",
		JWKSURL:    "https://www.googleapis.com/oauth2/v3/certs",
		Algorithms: []string{oidc.RS256},
	}
}
