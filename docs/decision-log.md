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