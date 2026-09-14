# Conventions

Rules of the road so backend code, schema, and API stay consistent as the project
grows. Violating these without a decision entry is a code-review failure.

## Naming

| Thing | Pattern | Example |
|---|---|---|
| Compose project | `ft-<app>` | `ft-backend` |
| App service | `app` | — |
| DB service | `db` | — |
| Image | `financial-tracker/backend:local` | — |
| DB volume | `pgdata` | — |
| Go module | `github.com/<owner>/financial-tracker/backend` | — |
| Binary | `cmd/server` | — |
| Tables | `snake_case`, plural | `interest_entries` |
| Columns | `snake_case` | `original_amount` |
| JSON fields | `snake_case` | `account_amount` |
| Go types | PascalCase, package-level | `TransactionRepo` |
| Migration files | `<NNNNNN>_<name>.up.sql` / `.down.sql` | `000004_create_transactions.up.sql` |

## Ports & binds

- API: **8080** (HTTP). Dev binds `0.0.0.0`; anything beyond dev binds
  `127.0.0.1` only — this app is not internet-facing until auth exists (D-003).
- `$PORT` (convention on Railway and other PaaS): if set, it **overrides** the
  configured HTTP port in any environment.
- PostgreSQL: **5432 internal only**, never published to the host.
- Port changes require a decision-log entry.

## Version pinning

- Base images pinned to exact tags — **no `latest`**: `postgres:16-alpine`,
  Go runtime pinned to the toolchain version in `go.mod`.
- Go dependencies: exact versions in `go.mod`/`go.sum`, no floating
  `@latest` in commands. Major-version bumps get a decision entry.

## Environments

Model: **one binary, environment = config** (same principle as the
bins-docker environments model; our deployment decisions are D-012/D-015).

- Config via env only (`caarlos0/env`): `DATABASE_URL`, `HTTP_ADDR`,
  `LOG_LEVEL`.
- `.env.example` committed; `.env` gitignored.
- Dev, test containers, and (future) prod all run the same binary.
- No build-time environment values, no baked secrets.

## Go layout & layering

```
cmd/server/main.go          — wiring only (config → db → router → listen)
internal/
  config/                   — env struct
  database/                 — pool setup + migrations (embedded)
  server/                   — router, middleware, response helpers
  domain/                   — entities (no imports outside domain)
  repository/               — SQL via pgx (no business logic)
  service/                  — business rules (no SQL, no HTTP)
  handler/                  — HTTP + DTO validation (thin)
  dto/                      — request/response structs
```

- One `service` call per handler; one repository boundary per service concern.
- Queries are hand-written SQL, always scoped by `user_id` (R1).
- Money fields are `int64` in domain/DTOs — never `float64` (D-010).
- Dates on the wire: `YYYY-MM-DD`; timestamps: RFC 3339.

## Middleware order

Applied outer→inner in `server.New` (chi `Use` order):

1. `recoverer` — panic → 500 envelope
2. `requestID` — `X-Request-ID` in + out (must wrap the logger so it can log the id)
3. `logger` — one slog line per request
4. `auth` (Phase 8, D-013) — single-user token gate
5. `router`

## Error envelope rules

- Every error leaves the API in the `{ "error": { ... } }` shape
  (`docs/api-design.md`).
- Repository errors map to `NOT_FOUND` only when the SQL found zero rows for a
  `:id` request; everything else is `INTERNAL_ERROR` (no leaking SQL details).
- Validation failures return `details` listing every offending field — fail fast,
  list all, don't stop at the first.
- Unknown query params on list endpoints: **rejected with 400** (typo
  prevention), not silently ignored.

## Money rules (non-negotiable)

- `BIGINT` minor units, signed; income +, expense − (D-010).
- `original_amount` and `account_amount` share a sign; server validates (D-007, R3).
- Account balances change only through transactions/interest entries, always in the
  same DB transaction as the ledger row.
- No server-side rounding, no float math, ever. Derived rates are display-only.

## Migration conventions

- One file pair per change; `up` and `down` must be exact inverses.
- Every migration is idempotent-safe: migrations run once by
  `golang-migrate`'s version table; never hand-edit applied migrations —
  add a new pair instead.
- Add/alter tables → then indexes in the same migration file.
- Soft-delete columns and `user_id` go in from the first creation, never retrofitted
  (D-003, D-011).

## Testing conventions

- Unit tests: service layer with a fake repository (`pgxmock` or hand-written
  fakes).
- Integration tests: `testcontainers-go` PostgreSQL, migrations applied at suite
  start, tables truncated between tests.
- Every balance-mutating operation gets a consistency test (create/update/delete/
  transfer/interest).
- `make test` runs unit + integration; must pass from a clean clone.