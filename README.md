# Cannabis COA Analyzer

A full-stack web app that extracts structured data from cannabis **Certificates of Analysis (COAs)** — the lab PDFs that report cannabinoid potency, terpene profiles, and safety-test results. Upload one or many PDFs; the server runs them through Claude concurrently, validates the output against a fixed schema, and persists the results to Postgres for browsing and editing.

Deployed as a single self-contained Go binary (SPA + API on one origin) on [Fly.io](https://fly.io).

## Features

- **Concurrent extraction pipeline** — a bounded worker pool (counting semaphore + `WaitGroup`) processes a batch of uploads in parallel; wall-clock time is bounded by the slowest single Claude call, not the sum. A 5-file batch completes in ~17s vs. ~85s sequential.
- **Structured outputs** — extraction is constrained to a typed JSON schema (`Analysis`: laboratory, sample metadata, cannabinoids, terpenes, pass/fail summary), so responses are predictable and directly storable.
- **Prompt versioning** — extraction prompts live as versioned files under `server/prompts/` (e.g. `extract_coa_v3.md`), loaded at startup rather than hardcoded.
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
- Node 22+ and pnpm
- PostgreSQL (or Docker)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (for running migrations)
- An Anthropic API key

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

7. **Run the client** — in a second terminal, start the Vite dev server (HMR):

   ```bash
   cd client
   pnpm install
   pnpm dev
   ```

   The app is then available at the URL Vite prints (default `http://localhost:5173`).

## Deployment

Ships as a single Docker image to Fly.io. Secrets (`JWT_SECRET`, `ANTHROPIC_API_KEY`, `DATABASE_URL`) are managed with `fly secrets set` — never baked into the image.

```bash
fly deploy      # build + ship
fly logs        # live logs
fly status      # machine health
```

Migrations run against Fly Postgres over a `fly proxy` tunnel. Pushes to `main` auto-deploy via GitHub Actions (`.github/workflows/fly-deploy.yml`).
