# Target Architecture

**Principle: backend-first, cleanly layered, no magic.** A single Go API server
exposes a REST API consumed later by a separate frontend. All state lives in
PostgreSQL. Nothing is shared with other apps; patterns are reused, code is not.

## System topology

```mermaid
graph LR
    subgraph Future[Frontend - Phase 8]
        UI[Web app<br/>separate repo]
    end
    subgraph Backend[backend/ - this plan]
        API[REST API<br/>Go + chi]
        DB[(PostgreSQL 16<br/>Docker volume)]
        API -->|SQL via pgx| DB
    end
    UI -->|HTTP /api/v1 JSON| API
    Dev[curl / Makefile] --> API
```

- The API never serves UI assets — it is JSON-only.
- PostgreSQL runs in Docker Compose (dev); production story is out of scope until
  the app proves useful.
- Single-user MVP: one seeded user, `user_id` injected per request (D-003).

## Internal layering

Every feature follows the same three layers; no layer reaches past the next one.

```mermaid
graph TD
    H[handler<br/>parse + validate + respond] --> S[service<br/>business rules]
    S --> R[repository<br/>raw SQL via pgx]
    R --> DB[(PostgreSQL)]
    subgraph DTO[request/response DTOs + struct tags]
        H
    end
```

- **handler** — owns HTTP concerns: reading params/bodies, validation (struct tags),
  calling exactly one service method, writing the standard envelope.
- **service** — business rules only: balance updates, budget progress, transfer pair
  creation, multi-currency rules. No SQL.
- **repository** — SQL only. Queries are hand-written, scoped by `user_id` always
  (R1). No business logic.

```mermaid
graph LR
    subgraph cmd
        main[cmd/server/main.go]
    end
    subgraph internal[internal/]
        config[config] --> server[server]
        database[database] --> server
        server --> handler
        handler --> service
        service --> repository
        domain[domain] --> handler
        domain --> service
        domain --> repository
    end
    main --> config
    main --> database
    main --> server
```

## Request lifecycle example: create a transaction

```mermaid
sequenceDiagram
    participant C as curl/frontend
    participant H as transaction handler
    participant S as transaction service
    participant R as transaction repository
    participant D as PostgreSQL

    C->>H: POST /api/v1/transactions
    H->>H: validate DTO (UUIDs, signs, date)
    H->>S: CreateTransaction(userID, dto)
    S->>S: resolve account + category (ownership)
    S->>R: BeginTx()
    R->>D: INSERT transaction; UPDATE accounts SET balance = balance + account_amount
    D-->>R: rows affected
    R-->>S: Commit()
    S-->>H: result + new account balance
    H-->>C: 201 { "data": { ... } }
```

In-flight failure of either statement rolls the whole thing back — balance and
transaction can never diverge (§ Balance consistency in `docs/data-model.md`).

## Service shape

- One binary: `cmd/server`; everything under `internal/`.
- Config via environment only (`caarlos0/env`): `DATABASE_URL`, `HTTP_ADDR`, etc.
  Nothing environment-specific is baked into the binary.
- Migrations live in `internal/database/migrations/` (embedded with
  `golang-migrate`), applied via `make migrate`.
- Logging: `log/slog` JSON output in dev and prod alike.

## Database topology

```mermaid
graph TD
    subgraph compose[Compose project ft-backend]
        API2[app:8080] --> DB2[(postgres:16-alpine<br/>volume for data)]
    end
    Dev[localhost:8080] --> API2
    Test[testcontainers-go<br/>ephemeral PG] -.migrations.-> API2
```

- Compose project `ft-backend`; app port `8080` (dev binds `0.0.0.0`).
- Test suite spins its own ephemeral PostgreSQL via testcontainers — never the dev
  database (Phase 7).

## Why not

| Option | Rejected because |
|---|---|
| ORM (GORM, sqlx row mapper) | Hidden SQL, migration magic, hard to undo (D-002) |
| Microservices | One domain, one binary — split later only with a decision entry |
| GraphQL | REST contract is simpler to version and consume; reports stay read-only queries |
| SQLite | Needs ACID multi-writer safety and the full SQL surface of PostgreSQL |
| Monorepo with frontend | Frontend stack is undecided; separate repos keep Phase 8 frictionless |