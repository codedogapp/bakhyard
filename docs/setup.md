# Setup

What a machine needs before `mise run up` can run Bakhyard: the local settings, `tailscale serve`, a Google OAuth client and its secret in the Keychain, and Postgres; Jira and Bitbucket are optional. [The plan](plan.md) says why (§3 Auth, §3 Data, §4.7).

## Local settings

Personal settings live in `mise.local.toml`, at the repo's root. git ignores it, and mise puts its `[env]` into every task's environment. Start it from the example, which lists every setting:

```sh
cp mise.local.example.toml mise.local.toml
```

| Variable | Service | Holds |
| --- | --- | --- |
| `BAKHYARD_NAME` | home | The name the greeting uses; "there" without it |
| `BAKHYARD_TAILSCALE_LOGIN` | gateway | The one Tailscale login let in |
| `BAKHYARD_HOST` | gateway | The ts.net name that `tailscale serve` publishes |
| `BAKHYARD_GOOGLE_CLIENT_ID` | gateway | The Google OAuth client's ID |
| `BAKHYARD_GOOGLE_EMAIL` | gateway | The one email let in; its first sign-in pins its Google account ([below](#the-pinned-google-account)) |
| `BAKHYARD_VAULT` | notes | The Obsidian vault's folder, as an absolute path |
| `BAKHYARD_JIRA_SITE` | todos | The Jira site, `https://<site>.atlassian.net`, whose open tickets todos lists ([below](#jira)) |
| `BAKHYARD_JIRA_API` | todos | Jira's REST API for a token with scopes, `https://api.atlassian.com/ex/jira/<cloud ID>` |
| `BAKHYARD_JIRA_EMAIL` | todos | The email of the Atlassian account that owns the token |
| `BAKHYARD_BITBUCKET_API` | pulls | Bitbucket's REST API, `https://api.bitbucket.org/2.0`, where pulls lists your open pull requests ([below](#bitbucket)) |
| `BAKHYARD_BITBUCKET_EMAIL` | pulls | The email of the Atlassian account that owns the Bitbucket token |
| `BAKHYARD_BITBUCKET_WORKSPACES` | pulls | The slugs of the workspaces to list your open pull requests in, separated by commas |
| `BAKHYARD_TEST_DATABASE` | Go tests | The test database's URL, password included ([below](#the-test-database)) |

The gateway doesn't start without its four settings, nor notes without its vault: each exits 1 and says what's missing. The gateway also waits for notes, notify, nudge, pulls and todos to start. Todos runs without Jira when none of the three Jira settings is set, and pulls without Bitbucket when none of the three Bitbucket settings is set.

On the Mac, the gateway, todos and pulls read their secrets from the login Keychain, as the sections below show. On the laptop, which has no Keychain, `BAKHYARD_SECRETS` names a folder of files instead, each named as its secret's Keychain account (`google-client-secret`, `postgres-gateway`, `postgres-todos`, `postgres-notify`, `postgres-nudge`, `vapid-key`, `jira-token`, `bitbucket-token`) and holding the secret exactly, with no newline at the end: the files of the Kubernetes Secrets mounted there (decision 154). Leave it unset on the Mac.

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

Each service keeps its data in a database of its own, owned by a role of its own, on a local PostgreSQL 18 (decisions 80 and 81). `process-compose.yaml` expects it at `127.0.0.1:5433`. So far the gateway and todos have one. For the gateway, choose a long random password for its role, then, as a Postgres superuser in `psql`:

```sql
CREATE ROLE bakhyard_gateway LOGIN;
\password bakhyard_gateway
CREATE DATABASE bakhyard_gateway OWNER bakhyard_gateway;
```

`\password` asks for the password, so it stays out of `psql`'s history and the server's log. Then put the same password in the Keychain, where the gateway reads it when it starts:

```sh
security add-generic-password -s bakhyard -a postgres-gateway -w
```

For todos, do the same with a password of its own:

```sql
CREATE ROLE bakhyard_todos LOGIN;
\password bakhyard_todos
CREATE DATABASE bakhyard_todos OWNER bakhyard_todos;
```

```sh
security add-generic-password -s bakhyard -a postgres-todos -w
```

Each service runs its migrations when it starts. It exits 1 with `read the database password: …` without its item, and with `reach the database: …` when Postgres doesn't answer within 10 s.

Notify and nudge each have a database and a password of their own, made as for todos with the roles `bakhyard_notify` and `bakhyard_nudge`, and the Keychain accounts `postgres-notify` and `postgres-nudge`.

Notify also needs the app's VAPID key, which signs the Web Push messages (decision 162). Make one, put it in the Keychain, and keep a copy: a new key makes every phone subscribe again.

```sh
security add-generic-password -s bakhyard -a vapid-key -w "$(openssl rand -base64 32 | tr '+/' '-_' | tr -d '=\n')"
```

Then open Nudge in the app on the phone, add the app to the Home Screen and turn notifications on. The agents' side, `tools/nudge`, is in [Nudge](#nudge).

## The test database

The Go tests run against a real Postgres (decision 82): a role and database of their own, in which each test makes an empty schema and drops it when it ends. Make them as for the gateway. A hex password, such as `openssl rand -hex 24` prints, needs no escaping in the URL:

```sql
CREATE ROLE bakhyard_test LOGIN;
\password bakhyard_test
CREATE DATABASE bakhyard_test OWNER bakhyard_test;
```

Then put the password in `BAKHYARD_TEST_DATABASE`'s URL in `mise.local.toml`, in place of `<password>`.

Without it, `mise run test` fails and points here. CI runs its own Postgres for the tests.

## Jira

Todos lists your open Jira tickets: those assigned to you whose status isn't in Jira's Done category (decision 121). It also looks up the tickets that your todos left to do name, to find those that are done (decision 133). It reads them with an API token with scopes, as Atlassian recommends (decision 103). To set it up:

1. Find the site's cloud ID. This prints `{"cloudId":"…"}`:

   ```sh
   curl https://<site>.atlassian.net/_edge/tenant_info
   ```

2. On [your Atlassian account's API tokens](https://id.atlassian.com/manage-profile/security/api-tokens), select Create API token with scopes. Pick an expiry date, at most a year away, the Jira app, and the scopes `read:jira-work`, `read:jira-user` and `write:jira-work`; time tracking needs the last two later.
3. Put the token in the Keychain. With `-w` last, `security` asks for it:

   ```sh
   security add-generic-password -s bakhyard -a jira-token -w
   ```

4. In `mise.local.toml`, set `BAKHYARD_JIRA_SITE` to `https://<site>.atlassian.net`, `BAKHYARD_JIRA_API` to `https://api.atlassian.com/ex/jira/<cloud ID>` and `BAKHYARD_JIRA_EMAIL` to your account's email, then restart todos.

Both URLs must be https, so the token never travels in the clear; only a loopback IP address, such as `http://127.0.0.1:8090`, may take http, for a stand-in Jira in tests. With only some of the settings, a URL that isn't https or no token in the Keychain, todos exits 1 and says which: `BAKHYARD_JIRA_EMAIL is not set`, `BAKHYARD_JIRA_API: "…" is not an https URL` or `read the Jira token: no Keychain item with service bakhyard and account jira-token`. Without any of the settings, `GET /api/todos/tickets` answers 404 `Jira is not set up`. Todos polls Jira as it starts and every 5 minutes after (decisions 122 and 133): it searches your open tickets, then fetches the other tickets that your todos left to do name. When Jira refuses the token, for example once it has expired, `GET /api/todos/tickets` keeps answering what the last poll to succeed found, adds Jira's answer as its `failure`, such as `search Jira: 401 Unauthorized` or `fetch Jira tickets: 401 Unauthorized`, and todos logs `poll Jira` with it. To replace the token, add `-U` to the command in step 3, then restart todos.

## Bitbucket

Pulls lists open pull requests on Bitbucket Cloud, in the workspaces you name: those you opened, and those that wait for your review (decisions 143, 146 and 171). It reads them with an API token with scopes, which Bitbucket's API requires (plan §2). To set it up:

1. On [your Atlassian account's API tokens](https://id.atlassian.com/manage-profile/security/api-tokens), select Create API token with scopes. Pick an expiry date, at most a year away, the Bitbucket app, and the scopes `read:pullrequest:bitbucket`, `read:user:bitbucket` and `read:repository:bitbucket`; the pull requests that wait for your review need the last one.
2. Put the token in the Keychain. With `-w` last, `security` asks for it:

   ```sh
   security add-generic-password -s bakhyard -a bitbucket-token -w
   ```

3. In `mise.local.toml`, set `BAKHYARD_BITBUCKET_API` to `https://api.bitbucket.org/2.0`, `BAKHYARD_BITBUCKET_EMAIL` to your Atlassian account's email and `BAKHYARD_BITBUCKET_WORKSPACES` to the slugs of the workspaces that hold your pull requests, separated by commas, such as `acme,globex`, then restart pulls. A workspace's slug, its workspace ID, is the part of its repositories' URLs right after `bitbucket.org/`; its Workspace settings show it too.

The API's URL must be https, as Jira's must; only a loopback IP address may take http, for a stand-in Bitbucket in tests. Spaces around a slug don't count, and a workspace named twice is listed once. With only some of the settings, a URL that isn't https, an empty slug, `.` or `..` among the slugs, or no token in the Keychain, pulls exits 1 and says which: `BAKHYARD_BITBUCKET_EMAIL is not set`, `BAKHYARD_BITBUCKET_API: "…" is not an https URL`, `BAKHYARD_BITBUCKET_WORKSPACES: "acme,,globex" is not a list of workspace slugs` or `read the Bitbucket token: no Keychain item with service bakhyard and account bitbucket-token`. Without any of the settings, `GET /api/pulls/mine` and `GET /api/pulls/reviews` answer 404 `Bitbucket is not set up`. Pulls polls Bitbucket as it starts and every 5 minutes after: it asks who you are, then lists your open pull requests in each workspace. It looks for the pull requests that wait for your review as it starts and every 15 minutes after: it lists each workspace's repositories, the latest updated first, then asks each of the 100 latest updated across the workspaces for its open pull requests with you among the reviewers, and `GET /api/pulls/reviews` answers them; a pull request in an older repository doesn't show. When Bitbucket refuses the token, for example once it has expired, `GET /api/pulls/mine` keeps answering what the last poll to succeed found, adds Bitbucket's answer as its `failure`, such as `find me on Bitbucket: 401 Unauthorized`, and pulls logs `poll Bitbucket` with it; `GET /api/pulls/reviews` does the same, and pulls logs `poll Bitbucket reviews`. To replace the token, add `-U` to the command in step 2, then restart pulls.

## Run

`mise run up` builds the services and runs them with process-compose. A service that exits restarts after 1 s, so a missing setting or secret shows up in that service's log every second until it's fixed.

Open `https://<machine>.<tailnet>.ts.net`. Without a session the app shows its sign-in screen, whose Sign in button goes to Google; a sign-in starts a session for 7 days, renewed as the app uses it, and lands on the Home. Sign out, at the header's right end, ends the session on that device only (decision 88). To sign out everywhere, for example after losing a phone, delete every session in `psql`, connected to `bakhyard_gateway`:

```sql
DELETE FROM sessions;
```

## The pinned Google account

The first sign-in with `BAKHYARD_GOOGLE_EMAIL` pins its Google account to the email, and every later sign-in needs that account too (decision 87). Another account with the same email gets a 403, and the gateway logs `refused a sign-in` with `another account with the email …`. The pins are in the gateway's database. To see them, in `psql`, connected to `bakhyard_gateway`:

```sql
SELECT email, pinned_at FROM pins;
```

To let another account in, for example after the address moved to a new Google account, delete the email's pin. The next sign-in with the email pins its account.

```sql
DELETE FROM pins WHERE email = 'you@example.com';
```

## Nudge

`mise run build` builds `tools/nudge` into `bin/tools/nudge`, the agents' command: `nudge ask`, `nudge mcp` and `nudge setup SERVER`. Run `nudge setup` with the app's `https://` address on each machine where an agent runs; it writes the `server` into `settings.json`, adds the MCP server and the skill to the agent. The agents reach the server's `/agent` path with the machine's Tailscale login, with no other key (decision 164).
