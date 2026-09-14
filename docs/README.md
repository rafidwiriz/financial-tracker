# Financial Tracker — Documentation

Planning and design docs for the financial tracker app. **Start with
[`plan.md`](./plan.md)** — it is the master plan; the other docs are reference
material it points to.

## Document map

| Document | What it's for | Read when |
|---|---|---|
| [`plan.md`](./plan.md) | Phase-based master plan with status, risks (R#), owner answers | Starting work; tracking progress |
| [`requirements.md`](./requirements.md) | Functional (FR#) and non-functional requirements, out-of-scope list | Scoping a feature |
| [`architecture.md`](./architecture.md) | System topology + internal layering (handler/service/repository) | Understanding how pieces fit |
| [`data-model.md`](./data-model.md) | ER diagram + every table column, amounts rules, balance consistency, transfers, budget progress | Writing SQL/migrations |
| [`api-design.md`](./api-design.md) | Full endpoint contract: envelope, errors, request/response examples | Building handlers or the frontend |
| [`conventions.md`](./conventions.md) | Naming, ports, pinning, layering, money rules, testing rules | Any code review |
| [`decision-log.md`](./decision-log.md) | ADR entries (D-001…D-011): why each design choice exists | Before implementing anything non-obvious |

## How the convention works

1. **Ambiguous?** → write a decision entry **before** implementing
   (`decision-log.md`).
2. **Planning work** → update `plan.md` phases and exit criteria.
3. **Code** → must not contradict `architecture.md` / `conventions.md` /
   `api-design.md` without a decision entry superseding them.

## Project structure

```
financial-tracker/
├── backend/       # REST API (Go + PostgreSQL) — Phase 2+ in plan.md
├── frontend/      # separate app — Phase 8 (future)
└── docs/          # this directory
```