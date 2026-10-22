# Setup

What a machine needs before `mise run up` can run Bakhyard: the local settings, `tailscale serve`, a Google OAuth client, and its secret in the Keychain. [The plan](plan.md) says why (§3 Auth, §4.7).

## Local settings

Personal settings live in `mise.local.toml`, at the repo's root. git ignores it, and mise puts its `[env]` into every task's environment.

| Variable | Service | Holds |
| --- | --- | --- |
| `BAKHYARD_NAME` | home | The name the greeting uses; "there" without it |
| `BAKHYARD_TAILSCALE_LOGIN` | gateway | The one Tailscale login let in |
| `BAKHYARD_HOST` | gateway | The ts.net name that `tailscale serve` publishes |
| `BAKHYARD_GOOGLE_CLIENT_ID` | gateway | The Google OAuth client's ID |
| `BAKHYARD_GOOGLE_EMAIL` | gateway | The one Google account let in |

For example:

```toml
[env]
BAKHYARD_TAILSCALE_LOGIN = "you@example.com"
BAKHYARD_HOST = "machine.tailnet.ts.net"
BAKHYARD_GOOGLE_CLIENT_ID = "1234567890-abcdef.apps.googleusercontent.com"
BAKHYARD_GOOGLE_EMAIL = "you@example.com"
```

The gateway doesn't start without its four settings: it exits 1 and says which one is missing.

## Tailscale

Tailscale runs from the Homebrew formula (decision 62). Publish the gateway on the tailnet:

```sh
tailscale serve --bg 8080
```

The app is then at `https://<machine>.<tailnet>.ts.net`, the name in `BAKHYARD_HOST`. `tailscale serve status` must show the tailnet only, never Funnel.

## Google OAuth client

In the [Google Auth Platform](https://console.developers.google.com/auth/clients), in a project of your own:

1. Under Audience, the app is External and in Testing, with your Google account as a test user.
2. Under Clients, a Web application client has the authorized redirect URI `https://<machine>.<tailnet>.ts.net/auth/google/callback`. A change to the URIs can take from 5 minutes to a few hours to work.
3. The client's ID goes into `BAKHYARD_GOOGLE_CLIENT_ID`.
4. Google shows a client secret only when it makes one, so put it straight into the Keychain (below).

To replace the secret, press Add secret on the client's page, store the new secret with `-U` (below), and restart the gateway. Once a sign-in works, disable the old secret, then delete it.

## The client secret in the Keychain

The gateway reads the client secret from the login Keychain when it starts (decision 79). Add it with:

```sh
security add-generic-password -s bakhyard -a google-client-secret -w
```

With `-w` last, `security` asks for the secret, so the secret stays out of the shell's history. To replace it, add `-U`. To check the item without showing the secret:

```sh
security find-generic-password -s bakhyard -a google-client-secret
```

Without the item, the gateway exits 1 with `read the Google client secret: no Keychain item with service bakhyard and account google-client-secret`.

## Run

`mise run up` builds the services and runs them with process-compose. A service that exits restarts after 1 s, so a missing setting or secret shows up in the gateway's log every second until it's fixed.

To sign in, open `https://<machine>.<tailnet>.ts.net/auth/google/login`. Until `signin-session` adds sessions, a sign-in only logs `signed in` and lands on the Home.
