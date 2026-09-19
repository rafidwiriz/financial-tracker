# Decision Log

ADR-style record of every non-obvious decision. Add an entry **before** implementing
when something ambiguous comes up.

## Format

```markdown
## D-<NNN> — <title>

- **Date:** YYYY-MM-DD
- **Status:** proposed | accepted | superseded by D-<NNN>
- **Context:** what forced the decision
- **Decision:** what we chose
- **Consequences:** trade-offs, follow-ups
```

---

## D-001 — Go + PostgreSQL

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner preference for the backend language and database; no framework
  requirements beyond that.
- **Decision:** Backend in Go, database PostgreSQL. HTTP routers/drivers chosen later
  (D-002).
- **Consequences:** Single static binary, excellent PostgreSQL driver ecosystem,
  strong standard library. Frontend is a separate app (Phase 8) and can use any stack.

## D-002 — chi + pgx, raw SQL, no ORM

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Learning-first and long-term viable: ORMs hide SQL and add magic that
  is hard to undo later. Financial queries need explicit control over transactions
  and joins.
- **Decision:** `chi` for routing, `pgx/v5` for the database driver, hand-written SQL
  in a repository layer, `golang-migrate` for migrations. No ORM.
- **Consequences:** More verbose data-access code, but every query is explicit and
  optimizable. Migrations stay plain SQL, reviewable and reversible.

## D-003 — Single-user MVP, `user_id` foreign keys from day one

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner wants a working single-user app first, multi-user later without
  a rewrite.
- **Decision:** Every table carries a `user_id` FK. The MVP boots with one seeded
  default user; a middleware injects that fixed user into every request. When
  multi-user arrives, only the middleware (JWT auth) and registration/login endpoints
  change.
- **Consequences:** Slightly more columns and index space now; near-zero cost later.
  All queries must be scoped by user from the first commit so no leaky endpoints
  exist to fix later.

## D-004 — Category tree as adjacency list

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Category hierarchy needed (parent/child). Options: adjacency list,
  nested sets, materialized paths.
- **Decision:** Adjacency list (`parent_id` column). Personal-finance trees are
  shallow (2–3 levels), so recursive CTEs are fast and simple.
- **Consequences:** Trivial inserts/updates; tree reads need a recursive CTE, which
  PostgreSQL handles well at this scale.

## D-005 — Tags as a first-class many-to-many entity

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner wants custom free-text labels per transaction (e.g.
  "tax-deductible"), multiple per transaction, and the ability to filter on them.
- **Decision:** A `tags` table plus a `transaction_tags` junction table
  (many-to-many). Tag names are unique per user.
- **Consequences:** Tags are queryable (filter by tag), reusable across transactions,
  and can be referenced by budgets (FR5). Renaming a tag updates all its references
  naturally through the single `tags` row.

## D-006 — Condition-based budgets via scope junction tables

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Budgets must target arbitrary combinations of categories, tags, and
  accounts — not just "one budget per category".
- **Decision:** A `budgets` table with a `scope_type` (`all` | `categories` | `tags` |
  `accounts` | `mixed`) and three junction tables (`budget_categories`,
  `budget_tags`, `budget_accounts`) holding the scope members.
- **Consequences:** Flexible scoping; progress queries get more complex (see
  `data-model.md` §Budget progress), but the model stays normalized and extensible.

## D-007 — Dual-amount multi-currency, no stored exchange rate

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Originally planned a stored exchange rate per transaction (amount,
  currency, rate → account amount). Owner clarified: they want to *track* the
  original currency amount and the account impact amount — both entered explicitly.
- **Decision:** Store `original_amount` (original currency, signed) and
  `account_amount` (account currency, signed, same sign) per transaction. Currency
  of the account comes from the account row. Exchange rate is **not stored**; it is
  derived as `|account_amount| ÷ |original_amount|` when display needs it.
- **Consequences:** Nothing is computed or rounded server-side — the numbers are
  exactly what the user entered, which matters for real bank statements. If
  auto-conversion is wanted later (auto exchange rates), a rate source can be added
  as a display aid without touching stored data.

## D-008 — Transfers are linked transaction pairs

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Requirement added: transfer money between accounts. Considered a
  dedicated `transfers` table vs. modeling as transactions.
- **Decision:** A transfer creates **two transaction rows** linked by a shared
  `transfer_pair_id`, created atomically (single DB transaction): one negative leg on
  the source account, one positive leg on the destination account. `category_id` is
  NULL for both legs. Transfers are excluded from budgets and income/expense reports.
- **Consequences:** Transfers are invisible to spending analysis without special
  casing; the single-table model keeps balance logic uniform. Update/delete must
  operate on both legs atomically.

## D-009 — Interest entries are a separate table, posted manually

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner wants compound interest tracked, posted manually, with reports.
  Considered reusing transactions with a special category.
- **Decision:** A dedicated `interest_entries` table (account_id, amount, date, note).
  Posting an entry updates the account balance atomically, same pattern as
  transactions. Reports aggregate directly off this table.
- **Consequences:** Interest reporting is trivial (`SUM(amount)` per account/period).
  There is a small conceptual overlap with transactions — the two are intentionally
  kept separate so reports are unambiguous.

## D-010 — Money as signed BIGINT in minor units

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Floating point cannot represent money exactly; the app handles
  multiple currencies with different minor-unit scales.
- **Decision:** All money columns are `BIGINT` in the currency's minor unit (cents for
  most fiat). Income is positive, expense negative. The API returns raw integers;
  the frontend formats.
- **Consequences:** Exact arithmetic, no rounding surprises. Care needed at the API
  boundary (document unit in every field; the frontend must divide by 100 for
  display).

## D-011 — Soft deletes for accounts and categories

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Historical transactions reference accounts/categories; hard deletion
  would orphan or rewrite history.
- **Decision:** Accounts get `is_closed`; categories get a nullable `deleted_at`.
  Transactions referencing them remain intact and filterable.
- **Consequences:** Slightly more query care (always filter `is_closed`/`deleted_at`
  in list endpoints); data is never silently lost.

## D-012 — Railway as the primary deployment target

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner wants the app deployable but explicitly does not want a VPS
  for this project.
- **Decision:** Primary deployment target is **Railway** (PaaS): GitHub-connected
  deploys, free `*.up.railway.app` domain, env/secret management. The app itself
  stays platform-agnostic (D-015).
- **Consequences:** Railway conventions apply at deploy time (`$PORT`, healthcheck
  path, env-only config), but nothing Railway-specific enters the code. Cost:
  Railway's usage-based billing; free domain keeps cost at the platform minimum.

## D-013 — Shared-secret access gate for single-user deployment

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** `AGENTS.md` forbids exposing the API beyond localhost without auth;
  owner wants the single-user app deployable now. Full user auth is deliberately
  deferred (D-003).
- **Decision:** A middleware requires `Authorization: Bearer <API_KEY>` on all
  `/api/v1/*` routes. `API_KEY` comes from env; if it is unset, the server
  **refuses to start** (fail closed). This gate is replaced by real user auth
  (JWT) when multi-user lands — the middleware boundary is the only thing that
  changes.
- **Consequences:** Safe public deployment now; one shared credential (rotate-able
  via env). Not per-user — acceptable for single-user mode. No schema impact.

## D-014 — Migrations run in the container entrypoint

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Deploys must never run against an unmigrated (or half-migrated)
  schema. A separate "manual migrate" step is easy to forget.
- **Decision:** The container entrypoint runs `migrate up` (embedded
  `golang-migrate`) after waiting for DB readiness, and the server **refuses to
  start if migrations fail**. Single instance → no migration race.
- **Consequences:** Deploy = safe schema state by construction. Dev keeps `make
  migrate` for local runs; the entrypoint path is the same code, so behavior
  matches.

## D-015 — Platform-agnostic deployment artifacts

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner wants the app deployable to any service if needed — Railway
  today, something else tomorrow — without a rewrite.
- **Decision:** Deployment artifacts are portable by design:
  - one multi-stage `Dockerfile` (pinned base, `scratch`/alpine runtime, small
    image);
  - configuration exclusively via env (`DATABASE_URL`, `HTTP_ADDR`, `API_KEY`,
    …) — the app never reads platform metadata;
  - `docker-compose.yml` stays valid for dev **and** any Docker host;
  - no Railway SDKs, no platform-specific features in code;
  - per-platform deploy steps live in a runbook (`docs/deployment.md`, Phase 8).
- **Consequences:** Moving platforms later = new runbook + env values, zero code
  changes. Slightly more config discipline now.

## D-016 — Managed Postgres on Railway, containerized locally

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner weighed Railway-managed Postgres vs containerized; the app is
  DB-agnostic (D-015), so this is purely an ops choice per environment.
- **Decision:**
  - **Deployed environments (Railway):** managed Postgres (Railway plugin) —
    backups/upserves handled platform-side. The app connects via `DATABASE_URL`.
  - **Local dev:** containerized `postgres:16-alpine` in Docker Compose — same
    pinned image as before, zero behavior change.
  - The app never distinguishes the two; only the env value differs. Railway's
    per-environment setup (e.g. dev/stage/prod) maps each environment to its own
    managed database instance.
- **Consequences:** Platform-side backup/restore tooling on Railway; local dev
  stays self-contained. If we ever move hosts, export the dump and point
  `DATABASE_URL` elsewhere — no code change. Railway environments also give us a
  place to try stage/prod flows cheaply later.

## D-017 — MIT license

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** No license file existed, so the code was all-rights-reserved by
  default. The owner wants to publish and share the project (portfolio /
  learning-first); considered MIT, Apache-2.0, AGPL-3.0, and public-domain
  options.
- **Decision:** The project is licensed under the **MIT License** (`LICENSE` at
  the repository root). Copyright line uses "Financial Tracker contributors"
  (SPDX: `MIT`).
- **Consequences:** Anyone may use, modify, and redistribute the code for any
  purpose — including commercial — provided the copyright notice is preserved
  in copies. No copyleft or network-share obligations, which suits a
  single-user personal app (D-003): there is no competitive value to protect.
  Apache-2.0's patent grant and AGPL's share-alike were evaluated and rejected
  as unnecessary friction for this project.

## D-018 — Default user bootstrap with a fixed UUID

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** D-003 requires one seeded user in single-user mode. Need a stable,
  reproducible identity for it and a place to create it.
- **Decision:**
  - Fixed UUID `00000000-0000-0000-0000-000000000001`, overridable via
    `DEFAULT_USER_ID` env.
  - Seeded at server startup (idempotent `INSERT … ON CONFLICT DO NOTHING`),
    **never** in a migration — schema migrations stay data-free.
  - Email `default@local`, empty password hash (no login in MVP); real values
    arrive with multi-user auth.
- **Consequences:** Single binary bootstraps itself; migrations stay pure schema.
  The fixed UUID is a convention the auth middleware (D-013) will use later.

## D-019 — Case-insensitive tag names

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** D-005 requires unique tag names per user, but PostgreSQL's plain
  `UNIQUE (user_id, name)` is case-sensitive — `Travel` and `travel` would be
  two tags. Owner confirmed case-insensitive uniqueness is desired while the
  stored name should keep the user's original casing.
- **Decision:**
  - Column stays `name VARCHAR(100)`, storing the casing as first entered (display
    fidelity).
  - Uniqueness enforced with a functional index: `UNIQUE (user_id, LOWER(name))`.
  - All tag-name lookups (create, autocomplete, rename checks) normalize with
    `LOWER()` in SQL; tag references in budgets/filters keep using `tag_id`, so
    they are unaffected.
  - Rejected: `citext` extension (extra managed dependency in every DB) and
    storing lowercased-only names (loses display casing).
- **Consequences:** Duplicates blocked regardless of case (API still returns 409
  CONFLICT, now case-blind); display preserves first-entered casing; every name
  query must consistently apply `LOWER()` — one convention to follow in Phase 3.
- **Follow-up:** pattern can be reused for category names if the owner wants the
  same behavior there (not decided).

## D-020 — Category names unique among siblings, case-insensitive

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Same ambiguity as D-019, but categories are a typed tree — a name
  like `Food` legitimately exists under different parents (expense vs income).
  Global uniqueness would be too strict; no rule invites sprawl.
- **Decision:** Unique **among siblings**, case-insensitive: two children of the
  same parent may not share a name, regardless of case. Implemented as
  `UNIQUE (user_id, parent_id, LOWER(name))`.
- **Consequences:** `Food` under `Expenses` and `Food` under `Income` coexist;
  `Food` + `food` under the same parent → 409. Updates/moves through the tree
  re-check the constraint.
- **Implementation wrinkle:** PostgreSQL treats NULLs as distinct in unique
  indexes, so root categories (`parent_id IS NULL`) need a partial index or a
  `COALESCE(parent_id, <sentinel>)` key — handled in the Phase 3 migration.

## D-021 — DELETE on accounts always soft-closes; no hard delete in v1

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Account deletion interacts with the ledger: hard deleting an
  account breaks `transactions.account_id` (CASCADE would delete money history,
  RESTRICT would force an error anyway). Owner weighed soft-close vs hard
  delete (history, reversibility, query cost, audit norms).
- **Decision:** `DELETE /api/v1/accounts/:id` always sets `is_closed = true` —
  with or without transactions. No hard delete path in v1; zero-transaction
  accounts take the same code path. Categories already behave likewise
  (`deleted_at`, api-design.md).
- **Consequences:** One rule, no special cases; transaction history and balance
  stay intact; an accidentally closed account can be reopened via PATCH. If
  true deletion is ever wanted (e.g. an account created by mistake), it stays a
  deliberate future decision — schema makes it impossible by accident.

## D-022 — QUERY HTTP method rejected for v1

- **Date:** 2026-09-14
- **Status:** accepted
- **Context:** Owner asked whether the QUERY method (safe, idempotent; query
  sent in the request body) is useful here. Surveyed support: chi v5.3.2 routes
  it natively (`mQUERY`); Go `net/http` has no `MethodQuery` constant yet
  (cosmetic); browsers/curl accept it; Railway's edge behavior is unknown.
- **Decision:** Not used in v1. List/report endpoints stay `GET` + query
  params; mutations stay `POST`/`PATCH`/`DELETE`.
- **Why not:** the API contract has no client-driven complex queries — budget
  scoping (D-006) is computed server-side from stored scope definitions, so the
  client never builds AND/OR trees; URL-encoding pain is a non-issue at
  single-user scale; the decisive risk is untestable-until-deploy
  (intermediaries/PaaS edges may drop request bodies on non-GET/POST methods).
- **Consequences:** Rejecting costs nothing — chi already supports QUERY, so
  adoption later is a small, contained change.
- **Revisit trigger:** a frontend (Phase 9) filter-builder that needs
  client-driven nested queries → new decision entry then, not automatic
  adoption.

## D-023 — Unset account color is stored as NULL and round-trips as `null`

- **Date:** 2026-09-19
- **Status:** accepted
- **Context:** `accounts.color` is nullable (`data-model.md`), the create DTO
  makes `color` optional, and pgx v5 refuses to scan SQL NULL into a plain Go
  `string` (`cannot scan NULL into *string`). A repo that scans `color` into
  `string` would error on every row that has no color — surfacing as
  INTERNAL_ERROR on read. Found while reviewing `account_repo.go` (Phase 3).
- **Decision:** Unset color stays SQL NULL end-to-end: create → read. The
  domain representation of `color` is nullable (`*string`), and every repo
  scan accepts NULL for it. The API serializes an unset color as `"color":
  null`.
- **Consequences:** JSON responses may now contain explicit `null` — the
  frontend contract treats `color` as `string | null`. Making the column
  NOT NULL instead was rejected: the docs define it as nullable, so the Go
  side adapts, not the schema.

## D-024 — PATCH partial update cannot clear a nullable field in v1

- **Date:** 2026-09-19
- **Status:** accepted
- **Context:** PATCH updates use `COALESCE($n, column)` with pointer fields.
  A `*string` field conflates "field absent" with `"field": null` (JSON null
  also decodes to nil), so SQL cannot distinguish "leave unchanged" from
  "clear". Harmless for NOT NULL columns (`name`, `type`); for nullable
  `color` it makes clearing impossible. Found while reviewing
  `account_repo.go` (Phase 3).
- **Decision:** In v1, absent and explicit-null are the same: the field is
  unchanged. Clearing `color` once set is not possible via the API.
- **Consequences:** Accepted gap at single-user scale; `updated_at` still
  bumps on an unchanged PATCH (the UPDATE always writes `now()`).
- **Revisit trigger:** a frontend (Phase 9) "remove color" control → new
  decision entry on how to express clearing, then.

## D-025 — Closed accounts: hidden from lists, still fetchable by id; DELETE idempotent

- **Date:** 2026-09-19
- **Status:** accepted
- **Context:** `GET /accounts` filters `NOT is_closed`, while
  `GET /accounts/:id` and `DELETE /accounts/:id` do not — and a repeated
  DELETE returns the account again. `api-design.md` defines DELETE as soft
  close (D-021) but does not state list/detail behavior for closed accounts.
  Found while reviewing `account_repo.go` (Phase 3).
- **Decision:** Lists return only open accounts. Detail lookups still return
  closed accounts — old transaction views and PATCH-reopen (D-021) need
  them. DELETE is idempotent: closing an already-closed account succeeds
  with the same response.
- **Consequences:** Handlers must not assume a fetched account is open. Phase
  6 reports/dashboard must decide how closed accounts count in balances.
- **Revisit trigger:** UI needing "show closed accounts" → add
  `?include_closed=true` to the list endpoint (additive, non-breaking).

## D-026 — The owner writes the code; agents assist (learning-first)

- **Date:** 2026-09-19
- **Status:** accepted
- **Context:** The project is learning-first: the owner builds Go/PostgreSQL
  skills by doing, one phase at a time. Every new thread starts with an agent
  that may default to editing code directly — the AGENTS.md workflow says
  "implement one domain at a time" without saying who writes the code.
  Automating implementation would quietly take the learning out of the loop
  and leave the owner unable to maintain the codebase alone.
- **Decision:** The owner is the coder. Agents default to a hands-off-code
  posture: they plan, design, review, verify exit criteria, maintain docs,
  and produce concrete change specifications (file-level diffs, SQL,
  signatures, step-by-step instructions) — but do not edit implementation
  files unless the owner explicitly asks them to. An explicit delegation
  ("write it for me") is scoped to that request.
- **Scope:** "implementation files" = backend Go source, SQL migrations,
  tests, and build config (`Dockerfile`, `Makefile`, `docker-compose.yml`,
  `go.mod`); the same applies to the frontend when it starts (Phase 9).
  Docs (`docs/`, `AGENTS.md`, `README.md`) and decision entries stay
  agent-writable — the docs-first workflow depends on it.
- **Why not:** An always-coding agent is faster per change and matches common
  AI workflows, but it contradicts the project's stated mission
  (learning-first, no shortcuts).
- **Consequences:** Slower per-change velocity, by design; the owner is
  hands-on with every feature. Agent-side verification (curl / `make test`)
  matters more, since the agent proposes and the owner applies. New threads
  read this entry via `decision-log.md` and start from the hands-off
  default.
- **Revisit trigger:** The owner finds the tradeoff no longer serves them
  (e.g. time pressure) → new decision entry making agent-coding the default.