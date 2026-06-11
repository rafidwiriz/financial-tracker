# Financial Tracker — Requirements & Backend Plan

> Personal financial tracker with multi-currency support, tags, condition-based budgeting, and compound interest tracking.

---

## Table of Contents

1. [Requirements Overview](#1-requirements-overview)
2. [Data Model](#2-data-model)
3. [API Design](#3-api-design)
4. [Tech Stack](#4-tech-stack)
5. [Project Structure](#5-project-structure)
6. [Key Design Decisions](#6-key-design-decisions)
7. [Implementation Milestones](#7-implementation-milestones)
8. [Future Considerations](#8-future-considerations)

---

## 1. Requirements Overview

### 1.1 Project Goals

This is a **learning-first, long-term project**:

- **Learning-first** — The tech stack and architecture should be things worth learning (Go, PostgreSQL, clean layered design). Each milestone builds on the last so you can learn incrementally.
- **Long-term viable** — Avoid dead ends: no magic frameworks, no codegen you can't control, no architectural shortcuts that'll need a rewrite later. The plan favors simplicity and explicitness over cleverness.
- **Iterative** — Get a working (small) thing fast, then add features. Every milestone should leave the app in a usable state.

### 1.2 Functional Requirements

| # | Feature | Description |
|---|---|---|
| FR1 | **Income/Expense Tracking** | Record transactions with amount, date, description, account, and category |
| FR2 | **Tags/Labels** | Attach custom, user-defined tags to transactions (many-to-many). Tags are free-text names. |
| FR3 | **Category Tree** | Hierarchical categories (parent/child) for organizing income and expense types |
| FR4 | **Multi-Currency** | Each account has its own currency. Transactions can be in any currency with a manually entered exchange rate. |
| FR5 | **Condition-Based Budgeting** | Budgets scoped to a combination of categories, tags, and/or accounts, with configurable periods |
| FR6 | **Compound Interest** | Manually post interest entries to accounts. Reports showing interest earned over time. |
| FR7 | **Single User** | MVP is single-user. Architecture must support adding multi-user later with minimal changes. |

### 1.2 Non-Functional Requirements

- All monetary values stored as integers (minor currency units — cents).
- RESTful API with JSON request/response.
- PostgreSQL for data integrity (transactions, foreign keys).
- Run locally with Docker Compose.

---

## 2. Data Model

### 2.1 Entity Relationship Diagram

```
┌──────────┐       ┌──────────────┐       ┌──────────────────┐
│   users  │1──N──│   accounts   │1──N──│   transactions    │
└──────────┘       └──────────────┘       └──────────────────┘
                          ▲                         │  │
                          │              (transfer pairs —
                          │               two records
                          │               linked by
                          │               transfer_pair_id)
                          │                         │  │
                          └─────────────────────────┘  │
                                                       │
     ┌──────────────┐                    (many-to-many via junction)
     │  categories  │1──N──│             ┌──────────────────┐
     │  (tree, FK   │       │            │transaction_tags  │
     │   parent_id) │       │            └──────────────────┘
     └──────────────┘       │                       │
                            │              ┌──────────────────┐
                    ┌──────────────┐       │      tags        │
                    │   budgets    │       └──────────────────┘
                    └──────────────┘
                            │
                    (budget_scopes)
                    ┌────────────────┐
                    │budget_categories│
                    │  budget_tags   │
                    │ budget_accounts│
                    └────────────────┘

┌──────────────────┐
│ interest_entries  │──N──1 accounts
└──────────────────┘
```

### 2.2 Table Definitions

#### `users`
*Pre-populated for single-user; enables multi-user later.*

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK, default `gen_random_uuid()` | |
| `email` | `VARCHAR(255)` | NOT NULL, UNIQUE | |
| `password_hash` | `VARCHAR(255)` | NOT NULL | bcrypt |
| `display_name` | `VARCHAR(100)` | NOT NULL | |
| `default_currency` | `VARCHAR(3)` | NOT NULL, default `'USD'` | ISO 4217 |
| `created_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL, default `now()` | |

#### `accounts`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | |
| `name` | `VARCHAR(100)` | NOT NULL | e.g. "My Wallet", "EUR Bank Account" |
| `type` | `VARCHAR(20)` | NOT NULL | `checking`, `savings`, `credit_card`, `cash`, `investment` |
| `balance` | `BIGINT` | NOT NULL, default `0` | Cached from transactions, in account's minor unit |
| `currency` | `VARCHAR(3)` | NOT NULL | ISO 4217 — each account has one currency |
| `is_closed` | `BOOLEAN` | NOT NULL, default `false` | Soft close |
| `color` | `VARCHAR(7)` | nullable | Hex e.g. `#4CAF50` |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | |

**Index:** `(user_id)`

#### `categories`
*Tree (adjacency list) via parent_id.*

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | |
| `parent_id` | `UUID` | FK → `categories(id)`, nullable | Root category when NULL |
| `name` | `VARCHAR(100)` | NOT NULL | |
| `type` | `VARCHAR(10)` | NOT NULL | `income` or `expense` |
| `icon` | `VARCHAR(50)` | nullable | Icon name for UI |
| `sort_order` | `INTEGER` | NOT NULL, default `0` | For ordering siblings |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | |

**Index:** `(user_id, parent_id)` — efficient tree traversal.

> **Why adjacency list?** Simpler than nested sets / materialized paths. For personal finance the tree is shallow (2–3 levels max), so recursive CTEs work fine.

#### `tags`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | |
| `name` | `VARCHAR(100)` | NOT NULL | User-defined, e.g. "tax-deductible", "work-trip" |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |

**Unique constraint:** `(user_id, name)` — no duplicate tag names per user.

#### `transaction_tags`
*Junction table — many-to-many between transactions and tags.*

| Column | Type | Constraints |
|---|---|---|
| `transaction_id` | `UUID` | FK → `transactions(id)` ON DELETE CASCADE |
| `tag_id` | `UUID` | FK → `tags(id)` ON DELETE RESTRICT |

**PK:** `(transaction_id, tag_id)`

#### `transactions`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | |
| `account_id` | `UUID` | FK → `accounts(id)`, NOT NULL | |
| `category_id` | `UUID` | FK → `categories(id)`, nullable | Null for transfers |
| `transfer_pair_id` | `UUID` | nullable | If set, this transaction is part of a pair (see §2.5) |
| `description` | `TEXT` | nullable | |
| `date` | `DATE` | NOT NULL | Transaction date (not created_at) |
| **Multi-currency fields** | | | |
| `original_amount` | `BIGINT` | NOT NULL | Amount in **original currency's minor unit**. Positive = income, negative = expense. |
| `original_currency` | `VARCHAR(3)` | NOT NULL | Original ISO 4217 currency code |
| `account_amount` | `BIGINT` | NOT NULL | Impact on the account balance, in account's currency & minor unit. Same sign as `original_amount`. |
| | | | |
| `is_reconciled` | `BOOLEAN` | NOT NULL, default `false` | Verified against bank statement |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | |

**Exchange rate:** Not stored — it's derived when needed: `|account_amount| ÷ |original_amount|`. The user provides both amounts explicitly, so there's no ambiguity.

**Indexes:**
- `(user_id, date DESC)` — primary query pattern
- `(user_id, transfer_pair_id)` — find both sides of a transfer
- `(user_id, account_id, date DESC)` — account-specific queries
- `(user_id, category_id)` — category filtering
- `(user_id, original_currency)` — multi-currency filtering

> **Amount sign:** Income → positive. Expense → negative. `original_amount` and `account_amount` always share the same sign.

Example:

| Scenario | `original_amount` | `original_currency` | `account_amount` | Account currency |
|---|---|---|---|---|
| Spend $50 from USD account | `-5000` | USD | `-5000` | USD |
| Earn €100 into USD account | `10000` | EUR | `10500` | USD (rate: 1.05) |
| Buy £30 from EUR account | `-3000` | GBP | `-3540` | EUR (rate: 1.18) |
| Transfer $500 FROM USD to EUR (outgoing) | `-50000` | USD | `-50000` | USD |
| Transfer $500 FROM USD to EUR (incoming) | `46000` | EUR | `46000` | EUR |

#### `budgets`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | |
| `name` | `VARCHAR(100)` | NOT NULL | e.g. "Dining Out Budget" |
| `amount` | `BIGINT` | NOT NULL | Budget limit in account's minor units (the budget has one target currency) |
| `currency` | `VARCHAR(3)` | NOT NULL | Which currency this budget is in |
| `period` | `VARCHAR(10)` | NOT NULL | `weekly`, `monthly`, `yearly`, `custom` |
| `period_start` | `DATE` | NOT NULL | Start of budget period |
| `period_end` | `DATE` | nullable | For `custom` period; NULL means indefinite for periodic budgets |
| `rollover` | `BOOLEAN` | NOT NULL, default `false` | Unused amount rolls to next period |
| `scope_type` | `VARCHAR(10)` | NOT NULL, default `'all'` | `all`, `categories`, `tags`, `accounts`, `mixed` |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |
| `updated_at` | `TIMESTAMPTZ` | NOT NULL | |

**Scope type definitions:**
- `all` — matches all transactions in the period (no scope restrictions)
- `categories` — matches only transactions whose category is in `budget_categories`
- `tags` — matches only transactions that have at least one of the tags in `budget_tags`
- `accounts` — matches only transactions in the specified accounts
- `mixed` — matches transactions that satisfy a combination (e.g., category X AND tag Y)

#### Budget scope tables

**`budget_categories`**
| Column | Type | Constraints |
|---|---|---|
| `budget_id` | `UUID` | FK → `budgets(id)` ON DELETE CASCADE |
| `category_id` | `UUID` | FK → `categories(id)` ON DELETE CASCADE |

**`budget_tags`**
| Column | Type | Constraints |
|---|---|---|
| `budget_id` | `UUID` | FK → `budgets(id)` ON DELETE CASCADE |
| `tag_id` | `UUID` | FK → `tags(id)` ON DELETE CASCADE |

**`budget_accounts`**
| Column | Type | Constraints |
|---|---|---|
| `budget_id` | `UUID` | FK → `budgets(id)` ON DELETE CASCADE |
| `account_id` | `UUID` | FK → `accounts(id)` ON DELETE CASCADE |

#### `interest_entries`

| Column | Type | Constraints | Notes |
|---|---|---|---|
| `id` | `UUID` | PK | |
| `account_id` | `UUID` | FK → `accounts(id)`, NOT NULL | |
| `user_id` | `UUID` | FK → `users(id)`, NOT NULL | Denormalized for query safety |
| `amount` | `BIGINT` | NOT NULL | Interest amount in account's minor unit. Positive = earned, negative = charged. |
| `date` | `DATE` | NOT NULL | When interest was posted |
| `note` | `TEXT` | nullable | Optional description |
| `created_at` | `TIMESTAMPTZ` | NOT NULL | |

**Index:** `(account_id, date DESC)`

> **Why separate table instead of using transactions?** Interest entries are conceptually different — they represent a specific financial mechanism rather than a spend/earning event. The separate table makes reporting (total interest, projected compound growth) trivial without filtering transaction categories.

### 2.3 Multi-Currency: Amount Storage Rule

The user provides **both amounts explicitly** — there is no computed exchange rate stored in the database.

```
Table columns:
  original_amount   — amount in original currency (minor units, signed)
  original_currency — ISO 4217 code of the original currency
  account_amount    — impact on the account balance (minor units, signed)
```

**Rules:**
- `original_amount` and `account_amount` always have the same sign (both positive for income, both negative for expense).
- If the original currency matches the account's currency, then `original_amount = account_amount`.
- If the currencies differ, the user enters both values. For example, spending €50 from a USD account:
  - `original_amount` = `-5000` (EUR cents)
  - `account_amount` = `-5300` (USD cents, reflecting the actual debit)
- The effective exchange rate can be derived: `|account_amount| ÷ |original_amount|` (≈ 1.06 in this case).
- The account balance is always updated by `account_amount` (in the account's currency).

### 2.4 Balance Consistency

All operations that affect account balance must be wrapped in a database transaction:

```
BEGIN;
  INSERT INTO transactions (...);
  UPDATE accounts SET balance = balance + account_amount WHERE id = ?;
COMMIT;
```

Same pattern for updates (adjust old balance back, then apply new) and deletes (reverse the balance change).

### 2.5 Transfers Between Accounts

A transfer moves money from one account to another. Rather than a separate table, a transfer is modeled as **two transaction records linked by a shared `transfer_pair_id`**:

- **Outgoing leg** — Transaction on the source account with a negative `account_amount`. `category_id` is null. `transfer_pair_id` is set.
- **Incoming leg** — Transaction on the destination account with a positive `account_amount`. `category_id` is null. `transfer_pair_id` is set to the same value.

The user provides the outgoing leg's details and the incoming leg's `account_amount` explicitly. The server creates both legs atomically:

```
BEGIN;
  -- Outgoing leg (user specifies original_amount + original_currency + account_amount)
  INSERT INTO transactions (id, account_id, original_amount, original_currency, account_amount, transfer_pair_id, ...)
  VALUES (gen_random_uuid(), $from_account, $outgoing_original, $outgoing_currency, $from_amount, $pair_id, ...);
  UPDATE accounts SET balance = balance + $from_amount WHERE id = $from_account;

  -- Incoming leg (user specifies account_amount for the destination)
  INSERT INTO transactions (id, account_id, original_amount, original_currency, account_amount, transfer_pair_id, ...)
  VALUES (gen_random_uuid(), $to_account, $incoming_original, $incoming_currency, $to_amount, $pair_id, ...);
  UPDATE accounts SET balance = balance + $to_amount WHERE id = $to_account;
COMMIT;
```

**Key rules:**
- Both legs share the same `transfer_pair_id` (a UUID generated at creation time).
- `category_id` is always NULL for transfers — they aren't income or expense.
- Transfers are **excluded** from budget calculations and income/expense reports.
- Updating or deleting a transfer always operates on both legs atomically.
- If both accounts use the same currency, `original_amount = account_amount` on both legs.

---

## 3. API Design

### 3.1 Base URL

```
/api/v1
```

### 3.2 Standard Response Formats

```jsonc
// Single resource
{ "data": { ... } }

// Collection (paginated)
{
  "data": [ ... ],
  "pagination": { "page": 1, "limit": 50, "total": 342, "total_pages": 7 }
}

// Error
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      { "field": "amount", "message": "Amount is required" }
    ]
  }
}
```

### 3.3 Endpoints

#### Auth (for future multi-user; stubbed in single-user MVP)

| Method | Path | Description |
|---|---|---|
| `POST` | `/auth/register` | Create user |
| `POST` | `/auth/login` | Login, get tokens |
| `POST` | `/auth/refresh` | Refresh access token |
| `GET` | `/auth/me` | Current user profile |

In single-user mode: auto-seed a user on first run, use a simple API key or skip auth entirely.

---

#### Accounts

| Method | Path | Description |
|---|---|---|
| `GET` | `/accounts` | List all accounts |
| `POST` | `/accounts` | Create account |
| `GET` | `/accounts/:id` | Get account details |
| `PATCH` | `/accounts/:id` | Update account |
| `DELETE` | `/accounts/:id` | Close account (soft) |

**`POST /accounts`**
```jsonc
// Request
{
  "name": "Euro Travel Card",
  "type": "checking",
  "currency": "EUR",
  "color": "#2196F3"
}

// Response 201
{
  "data": {
    "id": "uuid",
    "name": "Euro Travel Card",
    "type": "checking",
    "balance": 0,
    "currency": "EUR",
    "color": "#2196F3",
    "is_closed": false,
    "created_at": "2026-06-11T00:00:00Z"
  }
}
```

---

#### Categories

| Method | Path | Description |
|---|---|---|
| `GET` | `/categories` | List all categories (flat or tree) |
| `POST` | `/categories` | Create category |
| `GET` | `/categories/:id` | Get category with children |
| `PATCH` | `/categories/:id` | Update category |
| `DELETE` | `/categories/:id` | Delete category |

**`GET /categories?tree=true`** — returns nested structure:
```jsonc
{
  "data": [
    {
      "id": "uuid",
      "name": "Transport",
      "type": "expense",
      "children": [
        { "id": "uuid", "name": "Fuel", "type": "expense", "children": [] },
        { "id": "uuid", "name": "Parking", "type": "expense", "children": [] }
      ]
    }
  ]
}
```

**`POST /categories`**
```jsonc
// Request
{
  "name": "Public Transit",
  "type": "expense",
  "parent_id": "uuid-of-transport"     // nullable, omit for root
}

// Response 201
{
  "data": {
    "id": "uuid",
    "name": "Public Transit",
    "type": "expense",
    "parent_id": "uuid-of-transport",
    "icon": null,
    "sort_order": 0
  }
}
```

> Deleting a category with children should either fail (with a message to delete children first) or optionally cascade. Recommend: soft-delete categories to preserve historical transaction links.

---

#### Tags

| Method | Path | Description |
|---|---|---|
| `GET` | `/tags` | List all tags |
| `POST` | `/tags` | Create tag |
| `PATCH` | `/tags/:id` | Rename tag |
| `DELETE` | `/tags/:id` | Delete tag |

**`POST /tags`**
```jsonc
// Request
{ "name": "tax-deductible" }

// Response 201
{ "data": { "id": "uuid", "name": "tax-deductible" } }
```

---

#### Transactions

| Method | Path | Description |
|---|---|---|
| `GET` | `/transactions` | List transactions (paginated) |
| `POST` | `/transactions` | Create transaction |
| `POST` | `/transactions/transfer` | Create a transfer between two accounts |
| `GET` | `/transactions/:id` | Get transaction |
| `PATCH` | `/transactions/:id` | Update transaction |
| `DELETE` | `/transactions/:id` | Delete transaction |
| `DELETE` | `/transactions/transfer/:pair_id` | Delete both legs of a transfer |

**Query parameters for `GET /transactions`:**

| Param | Type | Description |
|---|---|---|
| `page` | int | Default: 1 |
| `limit` | int | Default: 50, Max: 100 |
| `account_id` | UUID | Filter by account |
| `category_id` | UUID | Filter by category |
| `tag_id` | UUID | Filter by tag (any transaction with this tag) |
| `transfer_id` | UUID | Filter by transfer pair (get both legs) |
| `date_from` | date | Inclusive start |
| `date_to` | date | Inclusive end |
| `original_currency` | string | Filter by original currency (e.g. "EUR") |
| `search` | string | Search description |
| `min_amount` | int | Filter by abs(account_amount) >= |
| `max_amount` | int | Filter by abs(account_amount) <= |
| `type` | `income` / `expense` / `transfer` | Filter by direction. `transfer` returns records with a non-null `transfer_pair_id` |
| `sort_by` | string | `date` (default), `amount`, `created_at` |
| `sort_order` | `asc` / `desc` | Default: `desc` |

**`POST /transactions`**
```jsonc
// Request
{
  "account_id": "uuid",
  "category_id": "uuid",
  "original_amount": -500,        // -5.00 EUR
  "original_currency": "EUR",
  "account_amount": -530,          // -$5.30 USD (actual debit on the USD account)
  "description": "Coffee in Berlin",
  "date": "2026-06-10",
  "tag_ids": ["uuid1", "uuid2"]
}

// Response 201
{
  "data": {
    "id": "uuid",
    "account_id": "uuid",
    "category_id": "uuid",
    "original_amount": -500,
    "original_currency": "EUR",
    "account_amount": -530,
    "description": "Coffee in Berlin",
    "date": "2026-06-10",
    "tags": [
      { "id": "uuid1", "name": "travel" },
      { "id": "uuid2", "name": "food" }
    ],
    "is_reconciled": false,
    "account_balance_after": 99470
  }
}
```

> If the original currency matches the account's currency, `original_amount` and `account_amount` are identical. The user can omit `account_amount` in that case — the server treats it as equal to `original_amount`.

**`POST /transactions/transfer`**

Creates two linked transaction records — one outgoing on the source account, one incoming on the destination account. The user provides the outgoing original amount and both account amounts explicitly.

```jsonc
// Request
{
  "from_account_id": "uuid",
  "to_account_id": "uuid",
  "original_amount": 50000,              // 500.00 in source currency (positive magnitude)
  "original_currency": "USD",            // Source currency
  "from_account_amount": -50000,         // -500.00 USD debited from source
  "to_account_amount": 46000,            // 460.00 EUR credited to destination
  "to_original_amount": 46000,           // 460.00 EUR (destination's original currency)
  "to_original_currency": "EUR",         // Destination currency
  "description": "Transfer to savings",
  "date": "2026-06-11"
}

// Response 201
{
  "data": {
    "transfer_pair_id": "uuid",
    "from_transaction": {
      "id": "uuid",
      "account_id": "uuid-of-from-account",
      "original_amount": 50000,
      "original_currency": "USD",
      "account_amount": -50000,
      "description": "Transfer to savings",
      "date": "2026-06-11",
      "account_balance_after": 150000
    },
    "to_transaction": {
      "id": "uuid",
      "account_id": "uuid-of-to-account",
      "original_amount": 46000,
      "original_currency": "EUR",
      "account_amount": 46000,
      "description": "Transfer to savings",
      "date": "2026-06-11",
      "account_balance_after": 104600
    }
  }
}
```

Both legs are created atomically (single database transaction). The `type` query param with value `transfer` can be used to find all transfer transactions.

---

#### Budgets

| Method | Path | Description |
|---|---|---|
| `GET` | `/budgets` | List budgets with progress |
| `POST` | `/budgets` | Create budget |
| `GET` | `/budgets/:id` | Get budget details |
| `PATCH` | `/budgets/:id` | Update budget |
| `DELETE` | `/budgets/:id` | Delete budget |

**`POST /budgets`**
```jsonc
// Request
{
  "name": "Eating Out",
  "amount": 20000,           // 200.00
  "currency": "USD",
  "period": "monthly",
  "period_start": "2026-06-01",
  "scope_type": "mixed",
  "scope": {
    "category_ids": ["uuid-of-food"],
    "tag_ids": ["uuid-of-restaurant"],
    "account_ids": []         // empty = any account
  }
}
```

**`GET /budgets`** — returns progress calculated server-side:
```jsonc
{
  "data": [
    {
      "id": "uuid",
      "name": "Eating Out",
      "amount": 20000,
      "currency": "USD",
      "spent": 13500,
      "remaining": 6500,
      "percentage": 67.5,
      "period": "monthly",
      "period_start": "2026-06-01",
      "period_end": "2026-06-30"
    }
  ]
}
```

---

#### Interest Entries

| Method | Path | Description |
|---|---|---|
| `GET` | `/accounts/:id/interest` | List interest entries for an account |
| `POST` | `/accounts/:id/interest` | Post a manual interest entry |
| `PATCH` | `/interest/:id` | Update an interest entry |
| `DELETE` | `/interest/:id` | Delete an interest entry |

**`POST /accounts/:id/interest`**
```jsonc
// Request
{
  "amount": 1250,       // $12.50 interest earned
  "date": "2026-06-01",
  "note": "Monthly savings interest"
}

// Response 201
{
  "data": {
    "id": "uuid",
    "account_id": "uuid",
    "amount": 1250,
    "date": "2026-06-01",
    "note": "Monthly savings interest"
  }
}
```

> Interest entries also update the account balance (same atomic pattern as transactions).

---

#### Reports

> **Transfer handling in reports:** All income/expense reports and dashboard summaries exclude transfer transactions (`transfer_pair_id IS NULL`). Transfers do not affect income, expense, or budget calculations — they only affect account balances.


| Method | Path | Description |
|---|---|---|
| `GET` | `/reports/summary` | Period-over-period summary |
| `GET` | `/reports/spending-by-category` | Breakdown by category |
| `GET` | `/reports/monthly-trend` | Monthly income/expense over time |
| `GET` | `/reports/interest` | Interest earned/charged by account and period |
| `GET` | `/dashboard` | Aggregated main view data |

**`GET /reports/interest?account_id=uuid&date_from=2026-01-01&date_to=2026-12-31`**
```jsonc
{
  "data": {
    "account": { "id": "uuid", "name": "Savings Account", "currency": "USD" },
    "total_interest_earned": 5000,
    "total_interest_charged": 0,
    "entries": [
      { "amount": 1250, "date": "2026-01-01", "note": "Jan interest" },
      { "amount": 1260, "date": "2026-02-01", "note": "Feb interest" }
    ],
    "balance_impact": 5000       // how much balance changed due to interest
  }
}
```

**`GET /dashboard`**
```jsonc
{
  "data": {
    "total_balance": { "USD": 452000, "EUR": 150000 },
    "monthly_income": { "USD": 350000, "EUR": 0 },
    "monthly_expenses": { "USD": -212000, "EUR": -3000 },
    "total_interest_ytd": 5000,
    "recent_transactions": [ ... ],
    "budget_progress": [ ... ]
  }
}
```

---

## 4. Tech Stack

| Layer | Technology | Rationale |
|---|---|---|
| **Language** | Go 1.22+ | Great for REST APIs, single binary, strong stdlib, excellent PostgreSQL support |
| **HTTP Router** | `chi` | Lightweight, stdlib-compatible, middleware pattern |
| **Database Driver** | `pgx` v5 | Native PostgreSQL driver, best performance, supports `NUMERIC`, `UUID`, arrays |
| **Migrations** | `golang-migrate/migrate` | Standard, file-based, supports embedded migrations |
| **Validation** | `go-playground/validator` | Struct-tag-based validation, well-established |
| **Auth** | `golang-jwt/jwt/v5` + `golang.org/x/crypto/bcrypt` | Standard JWT + bcrypt for passwords |
| **Config** | `caarlos0/env` | Simple struct-based env var parsing |
| **UUID** | `google/uuid` | Standard UUID generation |
| **Logging** | `log/slog` (stdlib) | Structured logging, zero dependencies |
| **Testing** | `testing` + `testify` + `testcontainers-go` | Integration tests with real PG on demand |
| **Container** | Docker + docker-compose | PostgreSQL + app, reproducible dev |

### Key Libraries

```
chi           github.com/go-chi/chi/v5
pgx           github.com/jackc/pgx/v5
migrate       github.com/golang-migrate/migrate/v4
validator     github.com/go-playground/validator/v10
jwt           github.com/golang-jwt/jwt/v5
bcrypt        golang.org/x/crypto/bcrypt
env           github.com/caarlos0/env/v11
uuid          github.com/google/uuid
testify       github.com/stretchr/testify
testcontainers github.com/testcontainers/testcontainers-go
```

---

## 5. Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                # Entry point
├── internal/
│   ├── config/
│   │   └── config.go              # Env-based config struct
│   ├── database/
│   │   ├── postgres.go             # pgx pool setup
│   │   └── migrations/            # SQL migration files
│   │       ├── 000001_create_users.up.sql
│   │       ├── 000001_create_users.down.sql
│   │       ├── 000002_create_accounts.up.sql
│   │       ├── 000002_create_accounts.down.sql
│   │       ├── ...
│   │       └── 000007_create_interest_entries.up.sql
│   ├── server/
│   │   ├── server.go              # HTTP server setup, middleware, routes
│   │   ├── middleware/
│   │   │   ├── auth.go            # JWT verification
│   │   │   ├── logger.go          # Request logging
│   │   │   ├── recoverer.go       # Panic recovery
│   │   │   └── cors.go            # CORS headers
│   │   └── response.go            # Standard JSON response helpers
│   ├── domain/                    # Domain types / entities
│   │   ├── account.go
│   │   ├── category.go
│   │   ├── tag.go
│   │   ├── transaction.go
│   │   ├── budget.go
│   │   └── interest.go
│   ├── repository/                # Database access layer (pgx)
│   │   ├── account_repo.go
│   │   ├── category_repo.go
│   │   ├── tag_repo.go
│   │   ├── transaction_repo.go
│   │   ├── budget_repo.go
│   │   └── interest_repo.go
│   ├── service/                   # Business logic layer
│   │   ├── account_svc.go
│   │   ├── category_svc.go
│   │   ├── tag_svc.go
│   │   ├── transaction_svc.go
│   │   ├── budget_svc.go
│   │   └── interest_svc.go
│   ├── handler/                   # HTTP handlers (controllers)
│   │   ├── account_handler.go
│   │   ├── category_handler.go
│   │   ├── tag_handler.go
│   │   ├── transaction_handler.go
│   │   ├── budget_handler.go
│   │   ├── interest_handler.go
│   │   ├── report_handler.go
│   │   └── auth_handler.go
│   ├── router/
│   │   └── routes.go              # Route registration
│   └── dto/                       # Request/Response DTOs
│       ├── account_dto.go
│       ├── transaction_dto.go
│       └── ...
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── Makefile                       # Common commands
```

### Layered Architecture

```
Handler (HTTP) → DTO validation → Service (business logic) → Repository (database)
```

- **Handler** — Parses request, validates input (via DTO struct tags), calls service, writes response.
- **Service** — Implements business rules (budget progress calculation, balance updates, exchange rate logic).
- **Repository** — Raw SQL queries via `pgx`. No business logic.
- **Domain** — Pure data structs, shared across layers.

---

## 6. Key Design Decisions

### 6.1 Amount Storage (Multi-Currency)

- All amounts stored as `BIGINT` (minor currency units — cents for most fiat currencies).
- Two amounts per transaction: `original_amount` (original currency) and `account_amount` (account currency). Both are user-provided and share the same sign.
- Exchange rate is **not stored** — it's derived on read as `|account_amount| ÷ |original_amount|` when needed for display.
- Balance updates always use `account_amount`, ensuring account balance is always in the account's own currency.
- If the original currency matches the account's currency, the user can omit `account_amount` (server treats it as equal to `original_amount`).

### 6.2 Budget Progress Calculation

Budget progress is calculated server-side as a query:

```sql
SELECT COALESCE(SUM(t.account_amount), 0) AS spent
FROM transactions t
LEFT JOIN transaction_tags tt ON tt.transaction_id = t.id
WHERE t.user_id = $1
  AND t.date >= $2
  AND t.date <= $3
  AND t.transfer_pair_id IS NULL  -- transfers are not expenses
  AND t.account_amount < 0      -- expenses only
  AND (                          -- scope filtering
    (b.scope_type = 'all')
    OR (b.scope_type = 'categories' AND t.category_id = ANY($4))
    OR (b.scope_type = 'tags' AND tt.tag_id = ANY($5))
    OR ...
  )
```

For the `mixed` scope type, combine filters with AND/OR logic based on the budget definition.

> **Transfer exclusion rule:** All budget queries exclude transactions where `transfer_pair_id IS NOT NULL`. Transfers are neither income nor expense and should not affect budget tracking.

### 6.3 Category Tree

The API will support two modes:
- **Flat** (`GET /categories?tree=false`) — returns all rows, client builds tree.
- **Tree** (`GET /categories?tree=true`) — server returns nested structure using recursive CTEs.

```sql
WITH RECURSIVE cat_tree AS (
  SELECT id, parent_id, name, type, sort_order, 0 AS depth
  FROM categories WHERE parent_id IS NULL AND user_id = $1
  UNION ALL
  SELECT c.id, c.parent_id, c.name, c.type, c.sort_order, ct.depth + 1
  FROM categories c
  JOIN cat_tree ct ON ct.id = c.parent_id
)
SELECT * FROM cat_tree ORDER BY depth, sort_order;
```

### 6.4 Single-User Mode

For the MVP:
- No auth endpoints. A single, default user is created on first migration/seed.
- All endpoints operate under this user's scope using a fixed user ID.
- The database schema still uses `user_id` FK everywhere — the user_id is injected by a simple middleware that reads from a config or environment variable.
- When multi-user is added, swap that middleware for JWT auth, and everything else stays the same.

**Seed on first run:**
```
INSERT INTO users (id, email, password_hash, display_name)
VALUES ('00000000-0000-0000-0000-000000000001', 'default@local', '', 'Default User');
```

### 6.5 Soft Deletes

- Accounts → `is_closed` boolean.
- Categories → add `deleted_at TIMESTAMPTZ` nullable (to preserve historical transaction category links).
- Tags → hard delete allowed (no historical impact if we cascade properly... actually, tags in `transaction_tags` should prevent deletion or cascade).

Rule of thumb: if a resource is referenced by historical transactions, prefer soft delete.

---

## 7. Implementation Milestones

Each milestone leaves the app in a working state — you can run it, hit the API, and see results. Milestones build on each other, so you learn one concept at a time rather than trying to implement everything at once.

**Ordering rationale:** Scaffold → Accounts (simplest CRUD) → Categories (tree) → Tags (many-to-many) → Transactions (the core, most complex) → Budgets → Interest → Reports → Testing. Each step adds a new concept without overwhelming.

| # | Milestone | Key Deliverables | Est. |
|---|---|---|---|
| 1 | **Scaffold** | Go module, chi server, pgx pool, config, migrations runner, health check endpoint, Docker Compose, Makefile | 1 session |
| 2 | **Accounts** | Accounts table + migration, CRUD handler/service/repo, soft close | 1 session |
| 3 | **Categories** | Categories table + migration, tree CRUD, recursive CTE query | 1 session |
| 4 | **Tags** | Tags table + migration, tag CRUD | 1 session |
| 5 | **Transactions & Transfers** | Transactions + transaction_tags tables + migrations, CRUD with balance updates, multi-currency support, transfer pairs (atomic dual-leg creation), pagination + filtering | 2 sessions |
| 6 | **Budgets** | Budgets + scope tables + migrations, CRUD, progress calculation | 2 sessions |
| 7 | **Interest** | Interest entries table + migration, CRUD with balance impact, interest report endpoint | 1 session |
| 8 | **Dashboard & Reports** | Summary, spending-by-category, monthly-trend, dashboard aggregation | 1 session |
| 9 | **Testing** | Integration test suite with testcontainers, edge case coverage for balance consistency | 2 sessions |

---

## 8. Future Considerations

| Feature | Notes |
|---|---|
| **Multi-user auth** | Add JWT middleware, login/register endpoints, replace fixed user ID with JWT claims |
| **Recurring transactions** | Background goroutine/scheduler that creates transactions from templates |
| **CSV/OFX import** | Parse bank exports, auto-match categories |
| **Exchange rate auto-fetch** | Optional: fetch current rates from an API, user can override manually |
| **Tag analytics** | "Show me all tax-deductible transactions" — already supported via tag_id filter |
| **Budget rollover** | Unused budget carries to next period — schema already has `rollover` field |
| **Audit log** | Track changes to transactions for accountability |
| **Backup / export** | Export all data as JSON or CSV |
