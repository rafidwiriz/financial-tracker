# Financial Tracker

A personal financial tracker web app: track income and expenses across multiple
accounts and currencies, tag and categorize every transaction, scope budgets by
condition (tag / category / account), and post compound interest manually with
reports.

**Learning-first, long-term project** — built iteratively in phases, no shortcuts.

## Status

| Phase | What | Status |
|---|---|---|
| 0 | Requirements & scope (`FR1–FR8`) | ✅ done |
| 1 | Architecture, data model, API design, conventions, decisions (`D-001…D-011`) | ✅ done |
| 2 | Backend scaffold (Go + PostgreSQL) | ⬜ next |
| 3 | Core domains: accounts, categories, tags | ⬜ |
| 4 | Transactions & transfers | ⬜ |
| 5 | Budgets & interest | ⬜ |
| 6 | Reports & dashboard | ⬜ |
| 7 | Testing & hardening | ⬜ |
| 8 | Deployment (Railway, portable) | ⬜ |
| 9 | Frontend (separate app) | ⬜ future |

## Tech stack

| Layer | Choice |
|---|---|
| Backend | Go + `chi` + `pgx` (raw SQL, no ORM) |
| Database | PostgreSQL 16 (Docker Compose, pinned image) |
| Migrations | `golang-migrate` — plain SQL up/down pairs |
| API | REST `/api/v1`, consistent JSON envelope |

## Features (planned)

- Income/expense tracking with **categories** (tree hierarchy) and **tags**
  (custom, many per transaction)
- **Multi-currency**: each account has its own currency; transactions keep their
  original amount/currency *and* their account impact — both entered explicitly,
  no computed exchange rate
- **Transfers** between accounts: atomic linked transaction pairs, excluded from
  budgets/reports
- **Condition-based budgets**: scoped by categories, tags, and/or accounts, with
  progress tracking
- **Compound interest**: manually posted entries per account, with interest
  reports

## Repository layout

```
financial-tracker/
├── backend/          # REST API (Go + PostgreSQL) — Phase 2+
├── frontend/         # separate app — Phase 8 (future)
├── docs/             # planning & design docs (source of truth)
│   ├── plan.md       # master plan — phases, exit criteria, risks
│   ├── requirements.md
│   ├── architecture.md
│   ├── data-model.md
│   ├── api-design.md
│   ├── conventions.md
│   └── decision-log.md
└── AGENTS.md         # working rules for agents/contributors
```

## Documentation

Start at [`docs/plan.md`](docs/plan.md) — the master plan tracks phase status,
tasks, exit criteria, and known risks. `docs/decision-log.md` records every design
decision (ADR-style) and is the authority on *why* things are the way they are.

## Getting started

No runnable code yet — Phase 2 (scaffold) produces the first runnable backend.
Once it exists, this section will document:

```sh
make up        # start PostgreSQL + API via Docker Compose
make migrate   # apply migrations
curl localhost:8080/healthz   # → 200
```

## License

[MIT](LICENSE) — see D-017 in [`docs/decision-log.md`](docs/decision-log.md).