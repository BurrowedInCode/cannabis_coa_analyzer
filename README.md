# Cannabis COA Analyzer

A full-stack web app that extracts structured data from cannabis **Certificates of Analysis (COAs)** — the lab PDFs that report cannabinoid potency, terpene profiles, and safety-test results. Upload one or many PDFs; the server runs them through a configurable OpenRouter model concurrently, validates the output locally against a versioned JSON Schema, and persists the results to Postgres for browsing and editing.

Deployed as a single self-contained Go binary (SPA + API on one origin) on [Fly.io](https://fly.io).

## Features

- **Concurrent extraction pipeline** — semaphore-bounded goroutine fan-out processes uploaded files concurrently, with at most 10 active analyses shared across requests.
- **Bounded PDF uploads** — requests are limited to 5 PDFs, 10 MiB per file, and 52 MiB total; server-side signature and MIME checks reject invalid content before a paid model call.
- **Structured outputs** — requests supply a strict, versioned JSON Schema upstream, and responses are independently validated against it locally before decoding into `Analysis`.
- **Prompt versioning** — extraction prompts live as versioned files under `server/prompts/` (e.g. `extract_coa_v5.md`), loaded at startup rather than hardcoded.
- **Accuracy evals** — a curated golden dataset (`server/testdata/`) and a `cmd/eval` harness score extraction field by field, with float tolerance for LLM rounding — so prompt changes are measured, not guessed.
- **Token & cost tracking** — provider-reported input/output tokens and per-request USD cost are logged after successful extractions.
- **JWT auth** — bcrypt-hashed passwords, JWT issued in an `HttpOnly` / `Secure` / `SameSite=Strict` cookie; protected API routes behind auth middleware, and a client-side route guard that probes `/user/me`.
- **Edit workflow** — extracted analyses are reviewable and editable in the UI, persisted via `PUT`.

## Tech Stack

| Layer | Tech |
|-------|------|
| Backend | Go (stdlib `net/http`), `pgx/v5` connection pooling |
| AI | Official OpenRouter Go SDK with configurable model routing |
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

Same-origin serving means the `Secure` / `SameSite=Strict` auth cookie works unchanged in production without cross-origin configuration. The installed CORS middleware supports the separate local Vite origin through `CORS_ORIGIN`. Go 1.22 ServeMux precedence lets the `/` SPA catch-all coexist with the more-specific API routes.

**Extraction flow per file:** encode the PDF as a data URL → request strict structured output from the configured OpenRouter model → validate locally → decode and normalize the response → store the result + token usage.

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
    ├── internal/coa/       # extraction service, HTTP handlers, normalization, persistence
    ├── internal/llm/       # provider-neutral interface + OpenRouter adapter
    ├── prompts/            # versioned extraction prompts
    ├── schemas/            # versioned extraction JSON Schemas
    └── web/                # //go:embed of the built SPA
```

## Getting Started

### Prerequisites

- Go 1.26+
- Node 22+ (ships with Corepack, used to install pnpm)
- PostgreSQL (or Docker)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (see below)
- An [OpenRouter](https://openrouter.ai/) API key

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
   OPENROUTER_API_KEY=<your key>
   LLM_MODEL=openai/gpt-5.6-luna
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
2. **Upload** at `/upload` — select up to 5 COA PDFs (10 MiB each) and submit. They're processed concurrently; on success you're routed to the analyses list.
3. **Browse** at `/analyses` — a paginated list of extracted COAs (sample name, seed-to-sale number, test date, pass/fail).
4. **Inspect & edit** at `/analyses/{id}` — the full extraction (laboratory, cannabinoids, terpenes, safety-test summary). Edit mode lets you correct any field and save it back via `PUT`.

### Sample COAs

The repo ships 10 real-world COA PDFs under `server/testdata/coas/` (`sample01.pdf` … `sample10.pdf`) — use these to try the upload flow without hunting for your own documents.

## Evaluating Extraction Accuracy

Because the extraction is LLM-driven, behavior is measured with an **eval harness** rather than assumed. Each sample COA has a curated reference answer in `server/testdata/expected/` (e.g. `sample01.json`), and `cmd/eval` scores the model's output against it field by field. The resulting score demonstrates extraction performance against those expected-field conventions; it is not a claim of definitive scientific accuracy.

Run it from the `server/` directory (needs `OPENROUTER_API_KEY` and `LLM_MODEL` set — it calls the live API):

```bash
cd server
go run ./cmd/eval
```

Output records the model, prompt, schema, per-document and overall accuracy, provider token usage, cost, and every field that missed. The format looks like this (numbers illustrative):

```
MODEL   openai/gpt-5.6-luna
PROMPT  prompts/extract_coa_v5.md
SCHEMA  schemas/coa_analysis_v2.json

sample01.pdf             41/42  tokens=3042/629 cost=$0.001234
    terpene: Myrcene           want "0.82"  got "0.81"
...
DOCUMENTS <scored>/<evaluated> scored
OVERALL  <correct>/<total>  (<pct>%)
USAGE    tokens=<input>/<output> cost=$<total>
```

Flags let you point at other data or swap the prompt (handy for prompt iteration):

```bash
go run ./cmd/eval -coas testdata/coas -expected testdata/expected -prompt prompts/extract_coa_v5.md
```

Change `LLM_MODEL` to any compatible OpenRouter model ID to compare providers without changing code. The current evaluated default is `openai/gpt-5.6-luna`.

How scoring works (`internal/eval`):
- **Scalar fields** (lab name/address, sample name, test date, pass/fail, …) are compared case-insensitively after trimming.
- **Cannabinoids & terpenes** are matched by name, then values compared with a **tolerance** (`|want − got| ≤ 2% + 0.01`) to absorb LLM rounding.
- **Safety-test summary** entries are matched by name and compared on pass/fail status.
- A missing field counts as a miss and is reported as `(not found)`.
- A document rejected during extraction or validation contributes zero correct fields while its expected fields remain in the overall denominator.

### Unit tests

The scoring logic itself is unit-tested (no API calls, runs offline):

```bash
cd server
go test ./...
```

## Deployment

Ships as a single Docker image to Fly.io. Secrets (`JWT_SECRET`, `OPENROUTER_API_KEY`, `DATABASE_URL`) are managed with `fly secrets set` — never baked into the image. Set `LLM_MODEL` in the Fly environment to select the upstream model.

```bash
fly deploy      # build + ship
fly logs        # live logs
fly status      # machine health
```

Migrations run against Fly Postgres over a `fly proxy` tunnel. Deployments are started manually with `fly deploy` after verification.
