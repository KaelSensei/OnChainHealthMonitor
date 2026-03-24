# 🚀 CI/CD Guide - OnChainHealthMonitor

> **Who this is for:** Anyone working on the project who wants to understand how code gets from a local branch to a Docker image in the cloud - including beginners who've never heard of GitHub Actions or container registries.

---

## Overview

Every time code is pushed to this repository, an automated pipeline takes over. It checks the code for problems, runs tests, packages the code into a Docker image, and stores that image in a registry where it can be pulled and deployed anywhere.

Here is the full pipeline in text form:

```
Developer pushes code
        │
        ▼
GitHub Actions detects the push
        │
        ├─ Pull Request? ──► pr-checks.yml
        │                    commitlint + markdownlint
        │
        ├─ services/api/** changed? ──► ci-api.yml
        │                              go vet → staticcheck → go test → docker build (PR) / push GHCR (main)
        │
        ├─ services/collector/** changed? ──► ci-collector.yml
        │                                    go vet → staticcheck → go test → docker build (PR) / push GHCR (main)
        │
        ├─ services/analyzer/** changed? ──► ci-analyzer.yml
        │                                   go vet → staticcheck → go test → docker build (PR) / push GHCR (main)
        │
        ├─ services/notifier/** changed? ──► ci-notifier.yml
        │                                   go vet → staticcheck → go test → docker build (PR) / push GHCR (main)
        │
        ├─ services/subscription/** changed? ──► ci-subscription.yml
        │                                        go vet → staticcheck → go test → docker build (PR) / push GHCR (main)
        │
        ├─ dashboard/** changed? ──► ci-dashboard.yml
        │                            eslint → next build → vitest → docker build (PR) / push GHCR (main)
        │
        ├─ docker-compose / Kong / OpenAPI changed? ──► ci-infra.yml
        │                                              validate configs
        │
        └─ git tag v*.*.* pushed? ──► release.yml
                                      matrix build all services → push semver tags to GHCR
```

---

## Workflow Files

All workflows live in `.github/workflows/`. Here is what each one does:

| File | Trigger | What it does |
|------|---------|--------------|
| `ci-api.yml` | Push/PR touching `services/api/**` | Lint, test, build image (push to GHCR on main only) |
| `ci-collector.yml` | Push/PR touching `services/collector/**` | Lint, test, build image (push to GHCR on main only) |
| `ci-analyzer.yml` | Push/PR touching `services/analyzer/**` | Lint, test, build image (push to GHCR on main only) |
| `ci-notifier.yml` | Push/PR touching `services/notifier/**` | Lint, test, build image (push to GHCR on main only) |
| `ci-subscription.yml` | Push/PR touching `services/subscription/**` | Lint, test, build image (push to GHCR on main only) |
| `ci-dashboard.yml` | Push/PR touching `dashboard/**` | ESLint, next build, vitest, build image (push to GHCR on main only) |
| `ci-infra.yml` | Push/PR touching infra files | Validate docker-compose, Kong config, OpenAPI spec |
| `ci-e2e.yml` | Push/PR touching `e2e/**` | End-to-end smoke tests |
| `release.yml` | Push of a `v*.*.*` tag | Matrix build of all services with semver tags |
| `pr-checks.yml` | Every pull request | commitlint + markdownlint |

---

## Path-Based Triggers (The Smart Part)

This is a monorepo - four services live in the same repository. Without path-based triggers, pushing a one-line change to `collector` would rebuild and republish `api`, `analyzer`, and `notifier` too. That wastes time and produces misleading image versions.

Each service workflow uses `paths:` to watch only its own directory:

```yaml
on:
  push:
    branches: [main]
    paths:
      - 'services/collector/**'
      - '.github/workflows/ci-collector.yml'
```

**Result:** Change a file in `services/collector/` → only `ci-collector.yml` runs. The other three services are untouched.

The workflow file itself is also in the `paths` list - so if you change the CI configuration for a service, the pipeline re-runs to validate the change.

---

## How to Read a GitHub Actions Run

1. Go to the repository on GitHub
2. Click the **Actions** tab at the top
3. You'll see a list of workflow runs - each is one pipeline execution
4. Click a run to open it
5. On the left: **Jobs** (e.g. `Lint & Test`, `Build & Push Docker`)
6. Click a job to see its **Steps** (checkout, setup-go, install staticcheck, go vet…)
7. Click a step to expand its **Logs** - raw terminal output

Jobs run in order. `Build & Push Docker` only starts if `Lint & Test` succeeds (`needs: lint-and-test`). If tests fail, no broken image is ever pushed.

---

## How Images Are Tagged

Each service workflow uses `docker/metadata-action` to generate image tags automatically. Two tags are produced for every push to `main`:

| Tag format | Example | When |
|------------|---------|------|
| `sha-<short-commit>` | `sha-98c164f` | Every push to `main` |
| `latest` | `latest` | Every push to `main` |
| `v1.2.3` | `v1.2.3` | When a git tag is pushed |
| `v1.2` | `v1.2` | Same - minor alias |
| `v1` | `v1` | Same - major alias |

The `sha-` tag is the most important for traceability: you can always look up which exact commit produced a given image.

---

## What is GHCR? (Beginner Explanation)

**GHCR** stands for **GitHub Container Registry**. It is a place to store Docker images - like a warehouse for packaged versions of your software.

A **container registry** is similar to npm (for JavaScript packages) or PyPI (for Python packages), but for Docker images. Instead of `npm install express`, you do `docker pull ghcr.io/.../api:latest`.

Why do we push images to a registry?

- **Sharing:** Any server, CI runner, or teammate can pull the exact same image
- **Versioning:** Each tag (`sha-abc123`, `v1.0.0`) is an immutable snapshot - you can always roll back
- **Deployment:** Kubernetes pulls images from the registry at deploy time

GHCR is the natural choice here because the repository is already on GitHub, no extra account is needed, and it's free for public repos. Authentication uses the built-in `GITHUB_TOKEN` - no secret to rotate.

---

## How to Pull an Image from GHCR

All images are published under `ghcr.io/kaelsensei/onchainhealthmonitor/<service>`.

```bash
# Pull the latest api image
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/api:latest

# Pull a specific commit
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/api:sha-98c164f

# Pull a release version
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/api:v1.0.0

# Same for other services
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/collector:latest
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/analyzer:latest
docker pull ghcr.io/kaelsensei/onchainhealthmonitor/notifier:latest
```

If the image is on a private repository, log in first:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

---

## How to Trigger a Release

A release is triggered by pushing a git tag that starts with `v`:

```bash
# Tag the current commit
git tag v1.0.0

# Push the tag to GitHub
git push origin v1.0.0

# Or both in one line
git tag v1.0.0 && git push origin v1.0.0
```

This triggers `release.yml`, which runs a matrix build: all four services are built in parallel and pushed to GHCR with three tags each (`v1.0.0`, `v1.0`, `v1`).

**Semantic versioning convention:**

- `v1.0.0` → major release, breaking changes possible
- `v1.1.0` → new features, backwards compatible
- `v1.1.1` → bug fix only

---

## PR Checks (commitlint + markdownlint)

Every pull request triggers `pr-checks.yml`, which runs two validators:

### commitlint

Enforces [Conventional Commits](https://www.conventionalcommits.org/) format for commit messages:

```
<type>(<scope>): <description>
```

Valid types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `ci`

**Examples of valid commits:**

```
feat(collector): add rate limiting for RPC calls
fix(analyzer): handle nil score on startup
docs: update README with CI badges
chore(deps): bump go version to 1.23
```

**Examples of invalid commits (will fail):**

```
fixed stuff
update
WIP - dont review
```

Configuration lives in `.commitlintrc.json` at the repo root.

### markdownlint

Enforces consistent Markdown style across all `.md` files:

- No trailing spaces
- Blank line before headings
- Consistent list markers (`-` not `*`)
- No bare URLs (use `[label](url)` format)

Configuration lives in `.markdownlint.json` at the repo root.

---

## How to Run Lint Locally Before Pushing

Run these commands inside each service directory to catch issues before CI does:

```bash
# Navigate to a service
cd services/api   # or collector, analyzer, notifier

# 1. Go vet - catches obvious errors (wrong argument types, unreachable code, etc.)
go vet ./...

# 2. staticcheck - deeper linter (deprecated APIs, unused code, correctness issues)
#    Install once: go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...

# 3. Run tests with race detector
go test ./... -race

# 4. Verify the binary compiles
go build ./...
```

**Run all services at once from the repo root:**

```bash
for svc in api collector analyzer notifier; do
  echo "=== Checking services/$svc ==="
  (cd services/$svc && go vet ./... && staticcheck ./... && go test ./... -race)
done
```

**Lint Markdown files locally:**

```bash
# Install: npm install -g markdownlint-cli
markdownlint '**/*.md' --ignore node_modules

# Fix auto-fixable issues
markdownlint '**/*.md' --ignore node_modules --fix
```

---

## What to Do If a Build Fails

### Step 1: Find the failing step

Go to **Actions** → click the failing run → click the failing job → expand the failing step.

### Step 2: Read the error message

**Common Go build errors and fixes:**

| Error | Cause | Fix |
|-------|-------|-----|
| `undefined: SomeType` | Missing import or typo | Check `import` block; run `go build ./...` locally |
| `cannot use X as type Y` | Wrong type passed to function | Check function signature |
| `go.sum file has uncommitted changes` | New dependency added but not committed | Run `go mod tidy && git add go.sum go.mod` |
| `staticcheck: SA1019: deprecated` | Using a deprecated API | Update to the suggested replacement |
| `race detected` | Data race in tests | Fix concurrent access; use mutex or channels |
| `docker build: COPY failed` | File referenced in Dockerfile doesn't exist | Check the `COPY` paths in `Dockerfile` |

### Step 3: Fix locally, push again

```bash
# Fix the issue, then:
go vet ./...
staticcheck ./...
go test ./... -race
git add .
git commit -m "fix: correct type error in handler"
git push
```

The workflow will re-run automatically on the new push.

### Step 4: If the GHCR push fails

The `build-and-push` job only runs on `main` branch pushes. If it fails with a 403:
- Ensure the repo's **Settings → Actions → General → Workflow permissions** is set to "Read and write permissions"
- The `GITHUB_TOKEN` is automatic; no manual secret is needed

---

## Infra Validation (`ci-infra.yml`)

This workflow runs when infrastructure files change:

```yaml
paths:
  - 'docker-compose.yml'
  - 'gateway/**'
  - 'openapi.yaml'
  - '.github/workflows/ci-infra.yml'
```

It performs three checks:

1. **docker-compose validate** - `docker compose config` checks for YAML syntax errors and missing fields
2. **Kong config** - `deck validate` checks the Kong declarative config (`gateway/kong.yml`) against the Kong schema
3. **OpenAPI spec** - `redocly lint openapi.yaml` validates the spec for correctness and style

---

## Summary: The Happy Path

```
git checkout -b feat/my-feature
# ... make changes to services/api ...
git add .
git commit -m "feat(api): add /api/v1/protocols/{id}/history endpoint"
git push origin feat/my-feature
# → pr-checks.yml runs (commitlint ✅, markdownlint ✅)
# → ci-api.yml runs on the PR (lint ✅, test ✅, build ✅) - no image push on PRs

# PR is reviewed and merged to main
# → ci-api.yml runs again on main
# → image pushed: ghcr.io/kaelsensei/onchainhealthmonitor/api:latest
# → image pushed: ghcr.io/kaelsensei/onchainhealthmonitor/api:sha-abc1234

# Ready to release
git tag v1.2.0 && git push origin v1.2.0
# → release.yml builds all 4 services in parallel
# → all images pushed with v1.2.0, v1.2, v1 tags
```

---

*For local setup, see [LOCAL_SETUP.md](LOCAL_SETUP.md). For architecture decisions, see [../architecture/DECISIONS.md](../architecture/DECISIONS.md).*
