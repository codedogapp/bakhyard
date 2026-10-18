# AGENTS.md

The canonical rules for everyone who works on Bakhyard: the user, the lead agent and every sub-agent. If anything else disagrees with this file, this file wins. It changes only with the user's approval.

- What we build, and in which order: [`docs/plan.md`](docs/plan.md).
- How we build it: this file.

## Working agreement

- **Bit by bit.** A step is one small, focused change: one package or one component, about 150 hand-written lines or fewer, plus its tests. Never generate large batches of code.
- **Scope discipline.** Do exactly the step. No drive-by refactors. Ideas go into `docs/plan.md`, not into the code.
- **Ask, don't assume.** When something isn't decided here or in the plan, ask the user, then record the answer in the right file.
- **Verify facts.** Check versions, APIs and tool options against their source (registry, docs, `--help`) before using them. Never guess.
- **Stay in the repo.** Don't change anything outside it without the user's OK.

## Pipeline

Every step passes through these roles, in order. Each sub-agent role has its own custom agent in [`.github/agents/`](.github/agents/), which holds the role's procedure and report format; this file stays canonical. Every agent starts with a fresh context and gets this file, the step brief and the current diff.

| Role | Agent | Does | Never |
| --- | --- | --- | --- |
| Lead | the main session | Splits the work into steps; writes each step brief (goal, scope, out of scope, acceptance criteria); runs the roles below; keeps the docs in sync; opens the PR; merges after the user's OK | Writes code, config or tests |
| Senior Go engineer | `senior-go-engineer` | Implements the step with unit tests; runs every gate until green | Widens the scope; commits or pushes |
| QA | `qa-engineer` | Checks every acceptance criterion; runs every gate; probes edge cases and failure paths; adds missing tests | Changes production code |
| Code and architecture reviewer | `code-architecture-reviewer` (read-only tools) | Reviews the diff against this file: separation of concerns, package boundaries, naming, KISS, YAGNI, error handling, test quality, fit with the architecture. Marks each finding blocking or non-blocking | Edits files |
| User | | Reviews the PR; approves or rejects it | |

- QA defects and blocking findings go back to the engineer, then through QA and review again. After three rounds, the lead stops and asks the user.
- One agent at a time, in the foreground. No parallel or background agents writing code.
- Reports are short and factual: files touched, commands run and their results, open issues.
- Another language gets its own engineer role and agent when it arrives (for example, a senior Svelte and TypeScript engineer for the web app).
- The lead runs each role through its custom agent only, never through a generic one, and passes the step brief in the prompt. New or changed agent files load after a CLI restart.

## Definition of done

1. `mise run check` passes locally.
2. The PR's CI is green, including the SonarQube Cloud quality gate.
3. The user has approved the PR.

Never report a step as done before `mise run check` has passed, and say which gates actually ran.

## Git and pull requests

- Repo: [`codedogapp/bakhyard`](https://github.com/codedogapp/bakhyard), public. Default branch `main`.
- One step, one branch (`step/<id>`, ids from plan §0) and one PR. The lead squash-merges after the user approves, with an explicit subject and body: `gh pr merge --squash --subject "…" --body "…"`.
- Commit messages and PR titles follow Conventional Commits (`feat(home): …`, `fix: …`, `build: …`, `ci: …`, `docs: …`): an imperative subject of 72 characters or fewer, and a body that says why when it isn't obvious.
- **No AI attribution.** Never add `Co-authored-by` trailers for Copilot or any other AI agent.
- Never force-push `main` or rewrite pushed history.
- Secrets never go into git or chat. Local secrets live in `mise.local.toml` (gitignored); CI secrets are GitHub organisation secrets.
- The repo is public: no personal identifiers, work-project names or third-party material in tracked files.

## Engineering principles

- Clean code, KISS, YAGNI and separation of concerns.
- **One package, one job.** The package name says what it does. No `util`, `common`, `helpers` or `misc` packages; no god packages or god structs.
- **One concern per file**, and files stay short (see the gates).
- Small functions, early returns, shallow nesting.
- Domain states and enums get their own types, never bare strings.
- Prefer the standard library; never re-implement it.
- No speculative abstractions: an interface needs two implementations or a test seam; shared code needs two users.
- Pass dependencies explicitly. No package-level mutable state.
- Wrap errors with context (`fmt.Errorf("load config: %w", err)`) and handle each error once.
- Comment only where the code needs clarification. No heavy commenting.
- Tests check behaviour, not implementation details. Table-driven where it helps; no sleeps.

## Architecture

- **Lean microservices.** An experiment the user chose: each feature is its own service (process) behind a gateway, with one SPA and no micro-frontends. It starts with the gateway and one service. Infrastructure (an event bus, metrics, tracing, more services, cockpit features) arrives only when a step needs it.
- **One Go module** at the root: `github.com/codedogapp/bakhyard`. Each service lives in `services/<name>/`, with its `main.go` and its own `internal/` packages, so Go's `internal` rule keeps services out of each other's code. Code that two or more services need moves to the root `internal/`.
- Services talk only through their APIs. No service reads another service's data.

## Toolchain

- **mise** manages every tool and task, except tools that macOS and the CI runners already ship (curl, jq); those run at the OS's version. Every task has a description.
- **Latest versions.** Everything we install runs on its newest release.
  - mise tools use `latest` in `mise.toml`. mise skips releases younger than 24 h (its `minimum_release_age` default), and mise-action does the same for mise itself. CI installs mise and the tools afresh on every run, with mise-action's cache off. Update local tools with `mise upgrade --local`; a plain `mise upgrade` also changes the user's global tools.
  - The rest is pinned exactly at its newest release, checked against its registry: GitHub Actions by commit SHA with a `# vX.Y.Z` comment, Go modules in `go.mod`, npm packages in `package.json`. Renovate keeps these current.
- **Nothing deprecated.** Never use a deprecated tool, API, endpoint, parameter, field or flag; use its replacement. Check the source's deprecation notes before picking one.
- **Go**: the latest release. Before editing Go code, run the `use-modern-go` skill's `list` for the file and follow its guidelines.
- **Go lint and format**: golangci-lint, based on the user's labrys config; gofumpt and golines at 100 columns.
- **Web** (when it arrives): Svelte 5, Vite and TypeScript; Biome configured like the user's reference frontend project (the lead supplies the path); plain CSS custom properties, no Tailwind; visual identity per plan §7.
- **Mirror the user's repos** before inventing a convention: labrys (Go lint), jirascrap (CI, Sonar, Renovate) and the reference frontend project (mise monorepo, Biome, hooks).

## Quality gates

Measure everything we can with lightweight static tools. Thresholds change only with the user's OK. Every suppression (`//nolint`, a Sonar "won't fix") needs a reason next to it and the user's OK.

| Measure | Limit | Tool |
| --- | --- | --- |
| File length | ≤ 200 lines, not counting blank lines and comments | revive `file-length-limit` |
| Function length | ≤ 50 lines and ≤ 30 statements; tests exempt | `funlen` |
| Cognitive complexity | ≤ 15 per function | `gocognit` |
| Cyclomatic complexity | ≤ 10 per function | `gocyclo` |
| Nesting | ≤ 3 levels of control flow | revive `max-control-nesting` |
| Parameters, results | ≤ 4 parameters, ≤ 3 results | revive `argument-limit`, `function-result-limit` |
| Duplication | no clone of 100 tokens or more; tests exempt | `dupl` |
| Test coverage | ≥ 80% total, ≥ 70% per package; generated code and `main.go` wiring excluded | go-test-coverage |
| Security | no findings | `gosec` in golangci-lint |
| Vulnerabilities | none known | `govulncheck` |
| Formatting | no diff | gofumpt, golines |
| GitHub workflows | no findings | jactionlint (with shellcheck for `run:` scripts) |
| Dependencies | current | Renovate (GitHub App) |
| New code overall | quality gate passes | SonarQube Cloud |

The local gates run through `mise run check`. CI runs the same task and, in a parallel job, SonarQube Cloud.

### SonarQube Cloud

- It runs only in CI, on every PR and every push to `main`, and fails the build when the quality gate is red (`-Dsonar.qualitygate.wait=true`). Its verdict is final: `mise run check` is necessary but not sufficient.
- Never run a scanner locally. Scanning an uncommitted tree publishes it as a `main` analysis and corrupts the new-code baseline.
- Fix a red gate. Never explain it away, and never widen the exclusions in `sonar-project.properties` to turn it green.
- `mise run sonar-issues [pr]` lists the open issues, on `main` or on a pull request. Security hotspots show only in the SonarQube Cloud UI, because their API is deprecated.

## Communication

- With the user: terse (caveman style). Keep the review load small: what changed, where and why, in a few lines.
- Code, docs, commits and PRs: plain, normal prose.
- Docs live in `docs/`; diagrams use Mermaid. Refer to issues in other repos as `owner/repo#123`.
