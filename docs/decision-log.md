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