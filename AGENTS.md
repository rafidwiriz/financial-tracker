# AGENTS.md

Working rules for this project. Read this before making changes.

## Mission

Build a personal **financial tracker** web app: a REST API backend (Go +
PostgreSQL) with a **separate frontend** later. The backend is built first, one
phase at a time, and every phase leaves the app in a working state. This is a
**learning-first, long-term** project: no magic frameworks, no codegen we can't
control, no shortcuts that require a rewrite later.

## Current state

Planning is complete (`docs/`): requirements (FR1–FR8), architecture,
data model, API contract, conventions, and decision log (D-001…D-020). Phases
0–2 ✅ done — **Phase 3 (Accounts) is next** — see `docs/plan.md`. The scaffold
(`backend/`) runs locally via Docker Compose: Go 1.26.2, chi, pgx, embedded SQL
migrations, seeded default user, `/healthz` verified. For a multi-user future
the schema carries `user_id` from day one (D-003).

## Non-negotiables

1. **Money is integer minor units.** Every money column/value is `BIGINT`/`int64`
   in the currency's minor unit, signed (income +, expense −). `float64` for money
   is a hard failure (D-010).
2. **No computed conversions.** Transactions store `original_amount` +
   `original_currency` + `account_amount` exactly as the user entered them. No
   stored exchange rate, no server-side rounding (D-007).
3. **Balance and ledger never diverge.** Every balance mutation happens in the
   same DB transaction as the ledger row it belongs to (transaction or interest
   entry). Update/delete reverses the delta atomically.
4. **`user_id` scoping from the first commit.** Every table carries a `user_id`
   FK; every query is scoped by it — even in single-user mode. This is what makes
   multi-user safe later (D-003, R1).
5. **Transfers are pairs.** A transfer creates two linked transaction rows
   (`transfer_pair_id`) atomically. Transfer legs are never counted as income or
   expense in budgets/reports (D-008).
6. **Docs before solutions.** Ambiguity → decision-log entry **before**
   implementing. Code must not contradict `architecture.md`/`conventions.md`/
   `api-design.md` without a superseding decision entry.
7. **Pin everything.** Base images, Go toolchain, and dependencies are pinned to
   exact versions — no `latest`, no floating `@latest`.

## Repo layout

```
financial-tracker/
├── backend/          # REST API (Go + PostgreSQL) — created in Phase 2
├── frontend/         # separate app — Phase 8 (future, out of scope)
├── docs/             # planning & design docs (source of truth)
│   ├── plan.md       # master plan — status, tasks, exit criteria, risks
│   ├── requirements.md
│   ├── architecture.md
│   ├── data-model.md
│   ├── api-design.md
│   ├── conventions.md
│   └── decision-log.md
└── AGENTS.md / README.md
```

`backend/` follows the layout in `docs/architecture.md`
(`cmd/server` + `internal/{config,database,server,domain,repository,service,handler,dto}`).

## Per-phase workflow

1. **Read** the current phase in `docs/plan.md`; check its exit criteria.
2. **Resolve ambiguity** in `docs/decision-log.md` **before** implementing
   (new decision = new D-### entry).
3. **Implement** one domain at a time; keep queries user-scoped and balance
   mutations atomic (Non-negotiables).
4. **Verify** the phase's exit criteria (curl / `make test`) before marking the
   phase ✅ in `docs/plan.md`.
5. **Update docs** if behavior revealed anything the docs didn't capture.

## Definition of Done — per phase

- [ ] Exit criteria in `docs/plan.md` for the phase are met, verified by running
      the app (not just reading code).
- [ ] All queries are `user_id`-scoped.
- [ ] All money handling follows Non-negotiables 1–2.
- [ ] Balance mutations are atomic with the ledger row (Non-negotiable 3).
- [ ] New endpoints match `docs/api-design.md` (envelope, errors, money units).
- [ ] Migrations are plain SQL up/down pairs, pinned, embedded.
- [ ] No secrets or environment-specific values in committed code (only
      `.env.example`).
- [ ] Ambiguities encountered are recorded in `docs/decision-log.md`.

## Working rules

- **Scoping:** stay inside `financial-tracker/`. Don't touch anything outside the
  project root.
- **Docs are source of truth.** If code and docs disagree, the docs win — update
  the docs, or add a decision entry if the code was right.
- **Sequencing:** finish phase N before starting N+1. Phases 3–6 are one domain at
  a time; complete one domain fully before the next.
- **Networking:** dev DB and builds may need network (Docker pulls, `go mod
  download`). Say so explicitly when required.
- **No git history rewriting or restructuring** of the user's projects unless
  explicitly requested. Do not commit or create branches unless asked.
- **Single-user mode** (D-003): no auth endpoints yet — do not expose the server
  beyond localhost without adding auth first.

<!-- lean-ctx -->
## lean-ctx

lean-ctx is active — the MCP tools replace native equivalents.
Full rules: LEAN-CTX.md (open on demand — do not auto-load).
<!-- /lean-ctx -->
