# Bakhyard

A personal dashboard for one user, hosted at home: Obsidian notes, work todos, pull requests and hours, GitHub, games, a Copilot planner, the coding agents' questions and the laptop's alerts, in one place. It's reached over Tailscale and needs a Google sign-in. It works on a Mac and on an iPhone, where it installs as a Home Screen app, and sends push notifications to both.

Its pages:

- **Home**: what needs the user, gathered from the other apps: the Jira tickets and todos to see to, the open pull requests, the review requests and issues on GitHub, the weekdays with no hours logged, the alerts that fire and the tokens about to expire. Under it, this week's hours.
- **Notes**: the Obsidian vault, rendered. On the laptop, a note can be edited, and each save is committed to the vault's repository in Gitea.
- **Todos**: todos linked to Jira tickets, with an inbox of the user's open tickets.
- **Pull requests**: the user's open pull requests on Bitbucket, with where each stands with its reviewers, and those that wait for the user's review. On weekdays at 10:00 and 15:00, a push lists those that still wait for approvals.
- **Time**: the hours logged on Jira over the last 7 days, and a form that logs a day's.
- **GitHub**: the user's open pull requests, review requests and assigned issues, across every owner.
- **Games**: the games played in the last 14 days, from Exophase, with their covers from IGDB.
- **Planner**: plans worked out with Copilot, which the vault can keep as notes.
- **Nudge**: the questions that the user's coding agents ask, answered from the phone, and the agents at work.
- **Alerts**: the laptop's alerts, those that fire and those of the last 30 days.

[The plan](docs/plan.md) has the small steps it was built in and the decisions behind them. It runs at home on an always-on Linux laptop with k3s, deployed from a private homelab repo, and on a Mac for development.

## How it's built

Each feature is a Go service of its own, in `services/`, behind a gateway that checks who's asking and passes each request on. The app is a Svelte single-page app, in `apps/web`, which the `web` service embeds and serves. Four services keep their data in PostgreSQL, each in a database of its own: the gateway its sessions, todos its todos, notify the phones' push subscriptions and the alerts, and nudge the agents' questions. The services run on k3s on the laptop, and in process-compose on a Mac. [§4 of the plan](docs/plan.md#4-system-architecture-the-microservices-experiment) has the details.

## Requirements

- macOS, because the services read their secrets from the login Keychain. On the laptop's k3s they read them from files instead ([docs/setup.md](docs/setup.md#local-settings)).
- [mise](https://mise.jdx.dev), which installs every other tool at the version the repo pins.
- [Tailscale](https://tailscale.com), on the Mac and on every device that opens the app.
- PostgreSQL 18, on `127.0.0.1:5433`.
- A Google OAuth client, for the sign-in.
- An Obsidian vault in a folder on the Mac.

## Run it

```sh
git clone https://github.com/codedogapp/bakhyard.git
cd bakhyard
mise install
cp mise.local.example.toml mise.local.toml
```

Fill in `mise.local.toml`. Then set up Tailscale, the Google client, the Keychain and Postgres as [docs/setup.md](docs/setup.md) shows, and run:

```sh
mise run up
```

It builds the app and the services, then runs them in process-compose's terminal UI until you quit it. Open `https://<machine>.<tailnet>.ts.net`, the name in `BAKHYARD_HOST`, and sign in.

## Development

```sh
mise run check
```

It runs the local quality gate for the Go code and the web app: formatting, linting, the tests with their coverage thresholds, and the vulnerability scans. Then it builds the app and runs the end-to-end tests in Chromium and WebKit, which Playwright downloads the first time. The Go tests need a database of their own, which [docs/setup.md](docs/setup.md#the-test-database) shows how to make. `mise tasks ls --all` lists every task.

The services call each other through protobuf contracts in `proto/`. After changing one, run `mise run gen`, which regenerates their Go code in `internal/gen`; the gate fails while that code is out of date.

A change is done when the gate passes and its pull request's CI is green. SonarQube Cloud runs only in CI, on demand on `main` ([AGENTS.md](AGENTS.md#sonarqube-cloud)).

## Docs

- [AGENTS.md](AGENTS.md): how we work, with the rules for every change and the quality gates.
- [docs/plan.md](docs/plan.md): what we build, in what order and why, with every decision.
- [docs/setup.md](docs/setup.md): what a Mac needs before `mise run up`. The laptop's setup is in the homelab repo's docs.
