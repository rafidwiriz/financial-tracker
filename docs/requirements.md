# Requirements

## Project goals

This is a **learning-first, long-term project**:

- **Learning-first** — the tech stack and architecture must be worth learning
  (Go, PostgreSQL, clean layered design). Each phase builds on the last so it can be
  learned incrementally.
- **Long-term viable** — no dead ends: no magic frameworks, no codegen we can't
  control, no shortcuts that require a rewrite later. Simplicity and explicitness
  over cleverness.
- **Iterative** — get a working (small) thing fast, then add features. Every phase
  leaves the app in a usable state.

## Functional requirements

| # | Feature | Description |
|---|---|---|
| FR1 | **Income/Expense Tracking** | Record transactions with amount, date, description, account, and category |
| FR2 | **Tags/Labels** | Custom, user-defined tags attached to transactions (many-to-many). Tag names are free-text. |
| FR3 | **Category Tree** | Hierarchical categories (parent/child) for organizing income and expense types |
| FR4 | **Multi-Currency** | Each account has its own currency; each transaction records its **original amount/currency** and its **account impact amount** (both entered explicitly — see D-007) |
| FR5 | **Condition-Based Budgeting** | Budgets scoped to a combination of categories, tags, and/or accounts, with configurable periods |
| FR6 | **Compound Interest** | Manually posted interest entries per account, with reports over time (D-009) |
| FR7 | **Transfers** | Move money between accounts — two linked transaction records, atomic (D-008) |
| FR8 | **Single User (MVP)** | Single-user now; schema and auth must support multi-user later with minimal changes (D-003) |

## Non-functional requirements

- All monetary values stored as integers (minor currency units — cents).
- RESTful API with consistent JSON envelope.
- PostgreSQL for integrity (ACID transactions, foreign keys, migrations).
- Runs locally via Docker Compose.
- No hard dependency on external services (no bank integrations, no auto exchange rates).

## Out of scope (v1)

- Multi-user auth (planned — D-003)
- Recurring transactions
- CSV/OFX import
- Auto-fetched exchange rates
- Frontend (planned later — see `plan.md` Phase 8)