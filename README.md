# Bakhyard

A personal dashboard for one user, hosted at home: Obsidian notes, work todos and pull requests, GitHub, games, and a Copilot planner, in one place. It's reached over Tailscale and needs a Google sign-in. It works on a Mac and on an iPhone, where it installs as a Home Screen app.

It's early. Today Bakhyard has its app shell, the sign-in, and a Home with demo data. The modules come one small step at a time, and [the plan](docs/plan.md) has the steps, their order and the decisions behind them. It runs on a Mac for now, and later moves to an always-on Linux laptop with k3s.

## How it's built

Each feature is a Go service of its own, in `services/`, behind a gateway that checks who's asking and passes each request on. The app is a Svelte single-page app, in `apps/web`, which the `web` service embeds and serves. The gateway keeps its sessions, and todos its todos, in PostgreSQL, and process-compose runs the services. [§4 of the plan](docs/plan.md#4-system-architecture-the-microservices-experiment) has the details.

## Requirements

- macOS, because the services read their secrets from the login Keychain.
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

It runs the local quality gate for the Go code and the web app: formatting, linting, the tests with their coverage thresholds, and the vulnerability scans. The Go tests need a database of their own, which [docs/setup.md](docs/setup.md#the-test-database) shows how to make. `mise tasks ls --all` lists every task.

A change is done when the gate passes and its pull request's CI is green, including SonarQube Cloud's quality gate. SonarQube Cloud runs only in CI.

## Docs

- [AGENTS.md](AGENTS.md): how we work, with the rules for every change and the quality gates.
- [docs/plan.md](docs/plan.md): what we build, in what order and why, with every decision.
- [docs/setup.md](docs/setup.md): what a machine needs before `mise run up`.
