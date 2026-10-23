# Setup

What a machine needs before `mise run up` can run Bakhyard: the local settings, `tailscale serve`, a Google OAuth client and its secret in the Keychain, and Postgres. [The plan](plan.md) says why (§3 Auth, §3 Data, §4.7).

## Local settings

Personal settings live in `mise.local.toml`, at the repo's root. git ignores it, and mise puts its `[env]` into every task's environment.

| Variable | Service | Holds |
| --- | --- | --- |
| `BAKHYARD_NAME` | home | The name the greeting uses; "there" without it |
| `BAKHYARD_TAILSCALE_LOGIN` | gateway | The one Tailscale login let in |
| `BAKHYARD_HOST` | gateway | The ts.net name that `tailscale serve` publishes |
| `BAKHYARD_GOOGLE_CLIENT_ID` | gateway | The Google OAuth client's ID |
| `BAKHYARD_GOOGLE_EMAIL` | gateway | The one Google account let in |
| `BAKHYARD_TEST_DATABASE` | Go tests | The test database's URL, password included ([below](#the-test-database)) |

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

## Postgres

Each service keeps its data in a database of its own, owned by a role of its own, on a local PostgreSQL 18 (decisions 80 and 81). `process-compose.yaml` expects it at `127.0.0.1:5433`. So far only the gateway has one. Choose a long random password for its role, then, as a Postgres superuser in `psql`:

```sql
CREATE ROLE bakhyard_gateway LOGIN;
\password bakhyard_gateway
CREATE DATABASE bakhyard_gateway OWNER bakhyard_gateway;
```

`\password` asks for the password, so it stays out of `psql`'s history and the server's log. Then put the same password in the Keychain, where the gateway reads it when it starts:

```sh
security add-generic-password -s bakhyard -a postgres-gateway -w
```

The gateway runs its migrations when it starts. It exits 1 with `read the database password: …` without the item, and with `reach the database: …` when Postgres doesn't answer within 10 s.

## The test database

The Go tests run against a real Postgres (decision 82): a role and database of their own, in which each test makes an empty schema and drops it when it ends. Make them as for the gateway. A hex password, such as `openssl rand -hex 24` prints, needs no escaping in the URL:

```sql
CREATE ROLE bakhyard_test LOGIN;
\password bakhyard_test
CREATE DATABASE bakhyard_test OWNER bakhyard_test;
```

Then add its URL to `mise.local.toml`:

```toml
[env]
BAKHYARD_TEST_DATABASE = "postgres://bakhyard_test:<password>@127.0.0.1:5433/bakhyard_test"
```

Without it, `mise run test` fails and points here. CI runs its own Postgres for the tests.

## Run

`mise run up` builds the services and runs them with process-compose. A service that exits restarts after 1 s, so a missing setting or secret shows up in the gateway's log every second until it's fixed.

Open `https://<machine>.<tailnet>.ts.net`. Without a session the app shows its sign-in screen, whose Sign in button goes to Google; a sign-in starts a session for 7 days, renewed as the app uses it, and lands on the Home.
