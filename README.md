# Cannabis COA Analyzer

A full-stack web app that extracts structured data from cannabis **Certificates of Analysis (COAs)** — the lab PDFs that report cannabinoid potency, terpene profiles, and safety-test results. Upload one or many PDFs; the server runs them through Claude concurrently, validates the output against a fixed schema, and persists the results to Postgres for browsing and editing.

Deployed as a single self-contained Go binary (SPA + API on one origin) on [Fly.io](https://fly.io).

## Features

- **Concurrent extraction pipeline** — a bounded worker pool (counting semaphore + `WaitGroup`) processes a batch of uploads in parallel; wall-clock time is bounded by the slowest single Claude call, not the sum. A 5-file batch completes in ~17s vs. ~85s sequential.
- **Structured outputs** — extraction is constrained to a typed JSON schema (`Analysis`: laboratory, sample metadata, cannabinoids, terpenes, pass/fail summary), so responses are predictable and directly storable.
- **Prompt versioning** — extraction prompts live as versioned files under `server/prompts/` (e.g. `extract_coa_v3.md`), loaded at startup rather than hardcoded.
- **Accuracy evals** — a hand-verified golden dataset (`server/testdata/`) and a `cmd/eval` harness score extraction field by field, with float tolerance for LLM rounding — so prompt changes are measured, not guessed.
- **Token & cost tracking** — input/output tokens and per-request USD cost are logged for every analysis.
- **JWT auth** — bcrypt-hashed passwords, JWT issued in an `HttpOnly` / `Secure` / `SameSite=Strict` cookie; protected API routes behind auth middleware, and a client-side route guard that probes `/user/me`.
- **Edit workflow** — extracted analyses are reviewable and editable in the UI, persisted via `PUT`.

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go (stdlib `net/http`), `pgx/v5` connection pooling |
| AI | Anthropic Go SDK — Files API (beta) + Messages API, `claude-haiku-4-5` |
| Database | PostgreSQL (golang-migrate migrations) |
| Frontend | React 19, TanStack Router + Query, Tailwind v4, shadcn/ui |
| Deploy | Docker multi-stage build, Fly.io, Fly Postgres |

## Architecture

The Go server serves the built React SPA via `//go:embed` and mounts the API on the same origin:

```
Browser ──▶ Go server (:8080)
             ├─ /                      → embedded React SPA (SPA fallback to index.html)
             ├─ POST /user/register    → bcrypt + insert
             ├─ POST /user/login       → JWT in HttpOnly cookie
             ├─ GET  /user/me          → auth probe
             └─ /coa/*  (auth middleware)
                 ├─ POST /coa/analyze          → concurrent extraction pipeline
                 ├─ GET  /coa/analyses         → paginated list
                 ├─ GET  /coa/analyses/{id}    → detail
                 └─ PUT  /coa/analyses/{id}    → edit
```

Same-origin serving means the `Secure` / `SameSite=Strict` auth cookie works unchanged in production with no CORS handling. Go 1.22 ServeMux precedence lets the `/` SPA catch-all coexist with the more-specific API routes.

**Extraction flow per file:** upload PDF to Anthropic Files API → analyze with Claude against the schema → store structured result + token usage → delete the uploaded file.

## Project Structure

```
.
├── Dockerfile              # 3-stage: build SPA → build Go (embeds SPA) → minimal alpine
├── docker-compose.yml      # local Postgres + server
├── fly.toml                # Fly.io deploy config
├── client/                 # React SPA (Vite)
│   └── src/
│       ├── api/            # auth + coa fetch clients
│       ├── routes/         # TanStack file-based routes (_authenticated guard)
│       └── components/
└── server/
    ├── cmd/main.go         # entrypoint: wiring, graceful shutdown
    ├── db/                 # pgx pool + migrations
    ├── internal/auth/      # register, login, JWT, bcrypt
    ├── internal/coa/       # extraction service, handler (worker pool), store, schema
    ├── prompts/            # versioned extraction prompts
    └── web/                # //go:embed of the built SPA
```

## Getting Started

### Prerequisites

- Go 1.26+
- Node 22+ (ships with Corepack, used to install pnpm)
- PostgreSQL (or Docker)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (see below)
- An Anthropic API key

### Installing golang-migrate

Install the CLI with the Postgres driver compiled in:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

> The `-tags 'postgres'` is required. Without it the CLI builds without Postgres support and fails at runtime with `unknown driver postgres`. This installs to `$(go env GOPATH)/bin` — make sure that's on your `PATH`.
>
> Alternatives: `brew install golang-migrate` (macOS), the AUR `migrate` package (Arch), or a prebuilt binary from the [releases page](https://github.com/golang-migrate/migrate/releases) (all drivers bundled, no build tag needed).

### Local development

1. **Clone the repo:**

   ```bash
   git clone git@github.com:BurrowedInCode/cannabis_coa_analyzer.git
   cd cannabis_coa_analyzer
   ```

2. **Start Postgres** (via Docker):

   ```bash
   docker compose up -d postgres
   ```

3. **Configure the server** — copy the example env and fill in the values:

   ```bash
   cp server/.env.example server/.env
   ```

   ```
   DATABASE_URL=postgres://admin:password@localhost:5432/cannabis_coa_analyzer?sslmode=disable
   JWT_SECRET=<openssl rand -hex 32>
   ANTHROPIC_API_KEY=<your key>
   ```

   (The `docker-compose.yml` Postgres defaults are `admin` / `password` / db `cannabis_coa_analyzer`.)

4. **Apply migrations** with [golang-migrate](https://github.com/golang-migrate/migrate):

   ```bash
   migrate -path server/db/migrations \
     -database "postgres://admin:password@localhost:5432/cannabis_coa_analyzer?sslmode=disable" up
   ```

5. **Run the server** — installs Go deps on first build, then serves the API + embedded SPA on `:8080`:

   ```bash
   cd server
   go mod download
   go run ./cmd/main.go
   ```

6. **Configure the client** — the SPA reads its API base URL from `VITE_API_URL`. This file is gitignored, so create it for local development pointing at the server:

   ```bash
   echo 'VITE_API_URL=http://localhost:8080' > client/.env
   ```

7. **Run the client** — in a second terminal, enable Corepack (provisions pnpm at the version pinned in `client/package.json`), then start the Vite dev server (HMR):

   ```bash
   corepack enable
   cd client
   pnpm install
   pnpm dev
   ```

   The app is then available at the URL Vite prints (default `http://localhost:5173`).

## Using the App

1. **Register** at `/register`, then **log in** at `/login`. Login sets an `HttpOnly` auth cookie; the protected pages are guarded and will bounce you to `/login` if you're not authenticated.
2. **Upload** at `/upload` — select one or more COA PDFs and submit. They're processed concurrently; on success you're routed to the analyses list.
3. **Browse** at `/analyses` — a paginated list of extracted COAs (sample name, seed-to-sale number, test date, pass/fail).
4. **Inspect & edit** at `/analyses/{id}` — the full extraction (laboratory, cannabinoids, terpenes, safety-test summary). Edit mode lets you correct any field and save it back via `PUT`.

### Sample COAs

The repo ships 10 real-world COA PDFs under `server/testdata/coas/` (`sample01.pdf` … `sample10.pdf`) — use these to try the upload flow without hunting for your own documents.

## Evaluating Extraction Accuracy

Because the extraction is LLM-driven, correctness is measured with an **eval harness** rather than assumed. Each sample COA has a hand-verified "golden" answer key in `server/testdata/expected/` (e.g. `sample01.json`), and `cmd/eval` scores the model's output against it field by field.

Run it from the `server/` directory (needs `ANTHROPIC_API_KEY` set — it calls the live API):

```bash
cd server
go run ./cmd/eval
```

Output is per-document and overall accuracy, with a breakdown of every field that missed. The format looks like this (numbers illustrative):

```
sample01.pdf             41/42
    terpene: Myrcene           want "0.82"  got "0.81"
...
OVERALL  <correct>/<total>  (<pct>%)
```

Flags let you point at other data or swap the prompt (handy for prompt iteration):

```bash
go run ./cmd/eval -coas testdata/coas -expected testdata/expected -prompt prompts/extract_coa_v3.md
```

How scoring works (`internal/eval`):
- **Scalar fields** (lab name/address, sample name, test date, pass/fail, …) are compared case-insensitively after trimming.
- **Cannabinoids & terpenes** are matched by name, then values compared with a **tolerance** (`|want − got| ≤ 2% + 0.01`) to absorb LLM rounding.
- **Safety-test summary** entries are matched by name and compared on pass/fail status.
- A missing field counts as a miss and is reported as `(not found)`.

### Unit tests

The scoring logic itself is unit-tested (no API calls, runs offline):

```bash
cd server
go test ./...
```

## Deployment

Ships as a single Docker image to Fly.io. Secrets (`JWT_SECRET`, `ANTHROPIC_API_KEY`, `DATABASE_URL`) are managed with `fly secrets set` — never baked into the image.

```bash
fly deploy      # build + ship
fly logs        # live logs
fly status      # machine health
```

Migrations run against Fly Postgres over a `fly proxy` tunnel. Pushes to `main` auto-deploy via GitHub Actions (`.github/workflows/fly-deploy.yml`).
