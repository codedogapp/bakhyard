module github.com/codedogapp/bakhyard

go 1.27.2

require (
	github.com/coreos/go-oidc/v3 v3.21.0
	github.com/go-chi/chi/v5 v5.5.0
	golang.org/x/oauth2 v0.37.0
)

require github.com/go-jose/go-jose/v4 v4.1.6 // indirect

ignore ./apps/web
