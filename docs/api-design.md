# API Design

Base URL (dev): `http://localhost:8080/api/v1`. JSON only, no HTML responses.

## Response envelope

Every response follows one of three shapes:

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
    "details": [ { "field": "amount", "message": "is required" } ]
  }
}
```

> `details[].message` is a rule predicate ("is required", "must be at least 3
> characters"); combine it with a field label client-side — e.g.
> `"Amount " + message`.

## Money fields (contract with the frontend)

- All money values are **integers in minor units** (cents for most fiat) — never
  floats in any payload (D-010).
- Transaction money fields: `original_amount` (signed), `original_currency`,
  `account_amount` (signed). Same sign required for the two amounts (R3).
- `account_amount` may be omitted when `original_currency` equals the account's
  currency; the server stores it equal to `original_amount`.

## Error codes

| HTTP | Code | Meaning |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Malformed or out-of-range input; `details` lists fields |
| 404 | `NOT_FOUND` | Resource missing (or not owned by the user — never reveal existence) |
| 409 | `CONFLICT` | Uniqueness violation (e.g. duplicate tag name) |
| 422 | `UNPROCESSABLE_ENTITY` | Business rule violation (e.g. amount signs mismatch, deleting a non-empty category) |
| 500 | `INTERNAL_ERROR` | Unexpected; stack trace logged server-side only |

## Endpoints

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | 200 when DB reachable; `{"status":"ok"}` |

### Accounts

| Method | Path | Description |
|---|---|---|
| `GET` | `/accounts` | List accounts |
| `POST` | `/accounts` | Create account |
| `GET` | `/accounts/:id` | Account details |
| `PATCH` | `/accounts/:id` | Update (name, type, color, `is_closed` — D-028) |
| `DELETE` | `/accounts/:id` | Soft close (`is_closed=true`) |

```jsonc
// POST /accounts — request
{
  "name": "Euro Travel Card",
  "type": "checking",
  "currency": "EUR",
  "color": "#2196F3"
}
// 201 — response
{
  "data": {
    "id": "uuid", "name": "Euro Travel Card", "type": "checking",
    "balance": 0, "currency": "EUR", "color": "#2196F3",
    "is_closed": false, "created_at": "2026-06-11T00:00:00Z"
  }
}
```

> `PATCH` may send `is_closed` to reopen (`false`) or close (`true`); an absent
> field leaves the account unchanged (D-024, D-028). `DELETE` only closes.

### Categories

| Method | Path | Description |
|---|---|---|
| `GET` | `/categories` | List; `?tree=true` returns nested tree (recursive CTE, D-004) |
| `POST` | `/categories` | Create (optionally under `parent_id`) |
| `GET` | `/categories/:id` | Category with immediate children (D-034) |
| `PATCH` | `/categories/:id` | Update, including re-parenting (D-033) |
| `DELETE` | `/categories/:id` | Soft delete; 422 only if live children remain (D-032); idempotent, 200 + body (D-035) |

```jsonc
// GET /categories?tree=true — response
{
  "data": [
    {
      "id": "uuid", "name": "Transport", "type": "expense",
      "parent_id": null, "icon": "car", "sort_order": 10,
      "children": [
        { "id": "uuid", "name": "Fuel", "type": "expense",
          "parent_id": "uuid", "icon": null, "sort_order": 0,
          "children": [] }
      ]
    }
  ]
}

// POST /categories — request
{ "name": "Public Transit", "type": "expense", "parent_id": "uuid-of-transport",
  "icon": "bus", "sort_order": 20 }   // icon and sort_order optional
```

> **Category rules:**
> - `type` must match the parent's, and a category with live children cannot
>   change `type` (D-030).
> - `icon` (nullable) and `sort_order` (default `0`) are optional on create/PATCH
>   and returned by the API; lists, tree, and child arrays are ordered by
>   `sort_order, LOWER(name), id` (D-031).
> - `PATCH` may re-parent; self/descendant targets and type mismatches → 422
>   (D-033). Absent or `null` `parent_id` means "unchanged" (D-024), so a
>   category cannot be moved back to root in v1.
> - `GET /categories/:id` returns direct, non-deleted children only (D-034).
> - Soft delete preserves transaction history (D-011): deleted categories are
>   hidden from lists/tree but still fetchable by id, and DELETE is idempotent
>   (D-035). Deleting a category referenced by transactions is allowed — the
>   transactions keep the reference.
> - Sibling name collisions are case-insensitive 409s, and a deleted sibling's
>   name is free to reuse (D-020, D-036).

### Tags

| Method | Path | Description |
|---|---|---|
| `GET` | `/tags` | List tags, ordered by `LOWER(name)` |
| `POST` | `/tags` | Create (`{ "name": "tax-deductible" }`); 409 on duplicate — case-insensitive per D-019 |
| `PATCH` | `/tags/:id` | Rename; 409 if the new name collides case-insensitively (D-019) |
| `DELETE` | `/tags/:id` | Hard delete; 422 if referenced by transactions (guard lands in Phase 4, D-037) |

```jsonc
// GET /tags — response
{
  "data": [
    { "id": "uuid", "name": "tax-deductible", "created_at": "2026-11-17T10:00:00Z" }
  ]
}
```

> Names keep the casing first entered while uniqueness is case-insensitive
> (D-019). `DELETE` is a hard delete — `transaction_tags.tag_id` is
> `ON DELETE RESTRICT` — and the reference guard ships together with
> `transaction_tags` in Phase 4 (D-037).

### Transactions

| Method | Path | Description |
|---|---|---|
| `GET` | `/transactions` | List, paginated + filtered |
| `POST` | `/transactions` | Create |
| `POST` | `/transactions/transfer` | Create a transfer (two legs, atomic — D-008) |
| `GET` | `/transactions/:id` | Details incl. tags |
| `PATCH` | `/transactions/:id` | Update |
| `DELETE` | `/transactions/:id` | Delete (reverses balance) |
| `DELETE` | `/transactions/transfer/:pair_id` | Delete both legs |

**Query params for `GET /transactions`:**

| Param | Description |
|---|---|
| `page` / `limit` | Default 1 / 50, max limit 100 |
| `account_id` | Filter by account |
| `category_id` | Filter by category |
| `tag_id` | Transactions having this tag |
| `transfer_id` | Both legs of a transfer pair |
| `date_from` / `date_to` | Inclusive date range |
| `original_currency` | e.g. `EUR` |
| `search` | Substring on description |
| `min_amount` / `max_amount` | `abs(account_amount)` bounds |
| `type` | `income` \| `expense` \| `transfer` |
| `sort_by` / `sort_order` | `date` (default) \| `amount` \| `created_at`; `asc` \| `desc` |

```jsonc
// POST /transactions — request
{
  "account_id": "uuid",
  "category_id": "uuid",
  "original_amount": -500,        // -5.00 EUR
  "original_currency": "EUR",
  "account_amount": -530,          // -$5.30 USD actual debit (D-007)
  "description": "Coffee in Berlin",
  "date": "2026-06-10",
  "tag_ids": ["uuid1", "uuid2"]
}
// 201 — response
{
  "data": {
    "id": "uuid", "account_id": "uuid", "category_id": "uuid",
    "original_amount": -500, "original_currency": "EUR", "account_amount": -530,
    "description": "Coffee in Berlin", "date": "2026-06-10",
    "tags": [ { "id": "uuid1", "name": "travel" }, { "id": "uuid2", "name": "food" } ],
    "is_reconciled": false,
    "account_balance_after": 99470
  }
}
```

```jsonc
// POST /transactions/transfer — request (D-008)
{
  "from_account_id": "uuid",
  "to_account_id": "uuid",
  "original_amount": 50000,          // magnitude in source currency
  "original_currency": "USD",
  "from_account_amount": -50000,     // debit on source
  "to_account_amount": 46000,        // credit on destination
  "to_original_amount": 46000,       // destination's own original currency
  "to_original_currency": "EUR",
  "description": "Transfer to savings",
  "date": "2026-06-11"
}
// 201 — response: { "data": { "transfer_pair_id", "from_transaction", "to_transaction" } }
```

### Budgets

| Method | Path | Description |
|---|---|---|
| `GET` | `/budgets` | List with computed `spent` / `remaining` / `percentage` |
| `POST` | `/budgets` | Create with scope |
| `GET` | `/budgets/:id` | Details |
| `PATCH` | `/budgets/:id` | Update |
| `DELETE` | `/budgets/:id` | Delete |

```jsonc
// POST /budgets — request (D-006)
{
  "name": "Eating Out",
  "amount": 20000,
  "currency": "USD",
  "period": "monthly",
  "period_start": "2026-06-01",
  "scope_type": "mixed",               // all | categories | tags | accounts | mixed
  "scope": {
    "category_ids": ["uuid-of-food"],
    "tag_ids": ["uuid-of-restaurant"],
    "account_ids": []
  }
}
// GET /budgets — response item
{
  "data": [{
    "id": "uuid", "name": "Eating Out",
    "amount": 20000, "currency": "USD",
    "spent": 13500, "remaining": 6500, "percentage": 67.5,
    "period": "monthly", "period_start": "2026-06-01", "period_end": "2026-06-30"
  }]
}
```

Spent = `SUM(account_amount)` over expenses matching the scope, transfers excluded —
definition in `docs/data-model.md` §Budget progress.

### Interest entries

| Method | Path | Description |
|---|---|---|
| `GET` | `/accounts/:id/interest` | List entries for an account |
| `POST` | `/accounts/:id/interest` | Post manual entry (updates balance atomically) |
| `PATCH` | `/interest/:id` | Update (reverses + reapplies balance impact) |
| `DELETE` | `/interest/:id` | Delete (reverses balance impact) |

```jsonc
// POST /accounts/:id/interest — request
{ "amount": 1250, "date": "2026-06-01", "note": "Monthly savings interest" }
// 201 — response
{ "data": { "id": "uuid", "account_id": "uuid", "amount": 1250,
            "date": "2026-06-01", "note": "Monthly savings interest" } }
```

### Reports & dashboard

| Method | Path | Description |
|---|---|---|
| `GET` | `/reports/summary` | Period-over-period income/expense summary |
| `GET` | `/reports/spending-by-category` | Breakdown by category (tree-aware) |
| `GET` | `/reports/monthly-trend` | Monthly income/expense series |
| `GET` | `/reports/interest` | Interest totals + entries per account/period (D-009) |
| `GET` | `/dashboard` | Balances per currency, monthly income/expense, recent transactions, budget progress |

All income/expense aggregates **exclude transfers** (`transfer_pair_id IS NULL`) and
respect soft deletes.

```jsonc
// GET /reports/interest?account_id=uuid&date_from=2026-01-01&date_to=2026-12-31
{
  "data": {
    "account": { "id": "uuid", "name": "Savings Account", "currency": "USD" },
    "total_interest_earned": 5000,
    "total_interest_charged": 0,
    "entries": [
      { "amount": 1250, "date": "2026-01-01", "note": "Jan interest" }
    ],
    "balance_impact": 5000
  }
}

// GET /dashboard
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

## Security notes (single-user MVP)

- No auth endpoints yet (D-003): every request runs as the seeded user via a
  middleware. The API is **not** to be exposed beyond localhost without adding auth.
- All ownership checks (`user_id`) happen in SQL, not in handlers — this is the
  pattern that makes multi-user safe later (R1).