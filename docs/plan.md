# Financial Tracker Master Plan

Goal: a personal financial tracker web app with a **REST API backend** (Go +
PostgreSQL) and a **separate frontend** later. Backend-first: plan, build, and
validate the API before any UI work. This plan is phase-based; each phase leaves the
app in a working state — you can run it and hit the API.

Progress conventions: `✅` = done · `🔄` = in progress · `⬜` = not started.

## Phase 0 — Requirements & scope

**Status: ✅ done (2026-09-14).**

**Tasks**

1. Interview the owner on scope one feature at a time.
2. Record functional/non-functional requirements in `docs/requirements.md`.
3. Capture every ambiguous or non-obvious choice as a decision in
   `docs/decision-log.md`.

**Exit criteria**

- ✅ FR1–FR8 documented (`docs/requirements.md`).
- ✅ Stack chosen: Go + PostgreSQL (D-001), chi + pgx (D-002).
- ✅ Multi-currency model settled: dual-amount, no stored rate (D-007).
- ✅ Transfer model settled: linked transaction pairs (D-008).
- ✅ Single-user MVP scoped; multi-user later (D-003).

## Phase 1 — Architecture & conventions

**Status: ✅ done (2026-09-14).** Target architecture in `docs/architecture.md`;
naming/code/money rules in `docs/conventions.md`; data model in
`docs/data-model.md`; API contract in `docs/api-design.md`. Anything found in later
phases that contradicts these docs is reconciled in `docs/decision-log.md`.

## Phase 2 — Scaffold

**Status: ⬜**

Goal: a running server with a health check, connected to PostgreSQL, with
migrations working.

**Tasks**

1. `go mod init`, `cmd/server/main.go`, chi router skeleton.
2. `pgx` pool + config via env (`caarlos0/env`).
3. `golang-migrate` runner wired into a `make migrate` target.
4. Docker Compose: `postgres:16-alpine` + app service, Makefile targets
   (`up`, `down`, `logs`, `ps`, `shell`, `migrate`).
5. Seed the default single user (D-003).
6. Global error handler + standard response envelope (`docs/conventions.md`).
7. `GET /healthz` returns 200 when DB reachable.

**Exit criteria**

- `make up && curl localhost:8080/healthz` → `200`, from a clean clone.
- `make migrate` applies all migrations; re-runnable (idempotent).

## Phase 3 — Core domains: accounts, categories, tags

**Status: ⬜** — one domain at a time; each ends with working CRUD.

**Tasks (per domain)**

1. Migration (accounts/000001 → categories/000002 → tags/000003).
2. Domain structs → repository (`pgx`) → service → handler, per
   `docs/conventions.md` layering.
3. Soft-delete behavior where defined (D-011) — `is_closed` for accounts,
   `deleted_at` for categories.
4. Category tree: recursive CTE for `GET /categories?tree=true` (D-004).
5. Tag name uniqueness per user (D-005).

**Exit criteria**

- CRUD endpoints tested by hand with `curl` (accounts, categories incl. tree, tags).
- `DELETE` on an account with transactions is refused (or soft-closed with a clear
  error) — decided before implementing; record as D-012 if it changes behavior.

## Phase 4 — Transactions & transfers (the core)

**Status: ⬜**

The most complex phase: multi-currency dual amounts (D-007, D-010), atomic balance
updates, pagination/filtering, and transfer pairs (D-008).

**Tasks**

1. `transactions` + `transaction_tags` migrations (indexes per `data-model.md`).
2. `POST /transactions`: validate account ownership + category ownership, compute
   nothing — store `original_amount`/`original_currency`/`account_amount` as sent
   (D-007), update account balance atomically (see `data-model.md` §Balance
   consistency).
3. `PATCH`/`DELETE` — reverse balance delta atomically.
4. `GET /transactions` — pagination, filters (account/category/tag/date range/
   search/min-max/original_currency), sorts. Transfers excluded from type
   `income`/`expense`.
5. `POST /transactions/transfer` + `DELETE /transactions/transfer/:pair_id` —
   atomic dual-leg create/delete.
6. Tag attach/detach on create/update.

**Exit criteria**

- Creating a transaction changes the account balance correctly (curl-verified).
- Cross-currency entry (EUR original → USD account) stores both amounts as entered.
- A transfer debits one account and credits the other atomically; deleting it
  reverses both.
- Transfer legs do not appear in budget/report queries (Phase 6 verifies).

## Phase 5 — Budgets & interest

**Status: ⬜**

**Tasks**

1. `budgets` + scope junction tables (D-006).
2. Budget CRUD with scope validation (referenced tags/categories/accounts exist and
   belong to the user).
3. Progress calculation query (see `data-model.md` §Budget progress) — expenses
   only, transfers excluded.
4. `interest_entries` migration + CRUD, balance impact atomic (D-009).
5. `GET /reports/interest` — totals per account/period.

**Exit criteria**

- A "mixed" budget (category X AND tag Y, any account) sums only matching expenses.
- Posting/deleting an interest entry moves the account balance correctly.

## Phase 6 — Reports & dashboard

**Status: ⬜**

**Tasks**

1. `GET /reports/summary`, `GET /reports/spending-by-category`,
   `GET /reports/monthly-trend`.
2. `GET /dashboard` — balances per currency, monthly income/expense, recent
   transactions, budget progress.
3. Verify all report queries exclude transfers (D-008) and respect soft deletes.

**Exit criteria**

- Reports cross-checked against hand-computed sums over seeded test data.

## Phase 7 — Testing & hardening

**Status: ⬜**

**Tasks**

1. Integration tests with `testcontainers-go` (real PostgreSQL, migrations applied
   in a test DB, tables truncated between tests).
2. Balance-consistency edge cases: create/update/delete transaction, transfer
   rollback, concurrent updates (serialize via row lock).
3. Validation hardening: UUIDs, amounts, date formats, pagination bounds, unknown
   query params ignored or rejected (per `conventions.md`).
4. `make test` green from a clean clone.

**Exit criteria**

- No hand-verification needed to trust the API; `make test` covers the paths above.

## Phase 8 — Frontend (future)

**Status: ⬜**

- Separate app, tech stack chosen later (out of scope for the backend plan).
- Will consume `docs/api-design.md` as the contract; any backend additions get an
  ADR first.

---

## Known risks (tracked)

| # | Risk | Mitigation |
|---|---|---|
| R1 | Single-user auth bypass leaks into multi-user later (endpoints forgetting `user_id` scoping) | Every query carries `user_id` from the first commit (D-003); integration tests assert cross-user isolation once auth exists |
| R2 | Budget "mixed" scope semantics get subtle (AND vs OR across category/tag/account) | Define explicitly in `data-model.md` §Budget progress **before** implementing; test the matrix in Phase 5 |
| R3 | Dual-amount entry invites user errors (mismatched signs/rates) | Schema + Zod-style validation on the API boundary: signs must match, amounts non-zero; document field rules in `api-design.md` |
| R4 | Minor-unit integer amounts confuse API consumers (cents vs units) | Convention + docs: every money field documented as minor units; frontend format helper later |
| R5 | Learning curve: raw SQL + manual migrations cause mistakes in prod data | Migrations are plain SQL with up/down pairs; dev + test DBs only until Phase 7 |
| R6 | Frontend later wants fields the API lacks | API versioning (`/api/v1`) planned; additive fields are non-breaking; ADR-first policy |

## Owner answers (2026-09-14)

1. **Tags:** custom free-text names, user-defined, many per transaction (FR2).
2. **Budgets:** condition-based — scope is a combination of tags, categories, and/or
   accounts (FR5).
3. **Categories:** tree (nested) model preferred (FR3).
4. **Multi-currency:** track original amount/currency AND account impact amount —
   both entered explicitly, no manual exchange-rate field (D-007).
5. **Transfers:** required (FR7).
6. **Compound interest:** posted manually; at minimum a report for it (FR6).
7. **Users:** single-user for now, multi-user planned later (FR8).
8. **Stack:** Go + PostgreSQL (D-001).
9. **Project nature:** learning project that may become long-term — iterative,
   no dead ends.