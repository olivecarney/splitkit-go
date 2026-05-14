# SplitKit

SplitKit is a lightweight expense-splitting web app for small groups. It is planned as a clean Go MVP that lets users create groups, add members, record shared expenses, calculate balances, and mark settlements as paid.

The app should stay simple, fast, and server-rendered.

## MVP Scope

The first version focuses on:

- Creating groups
- Adding group members
- Adding shared expenses
- Splitting expenses equally
- Calculating member balances
- Suggesting who owes whom
- Marking settlements as paid
- Running with a development-only user

Out of scope for the MVP:

- Real authentication
- Email invites
- Payment processing
- Bank connections
- Multi-currency support
- Unequal or percentage splits
- Receipt uploads

## Tech Stack

- Go
- PostgreSQL
- sqlc
- templ
- TailwindCSS
- htmx
- Alpine.js for small UI interactions where useful

## Planned Structure

```txt
cmd/
  web/
    main.go

internal/
  app/
  db/
  groups/
  expenses/
  balances/
  settlements/
  views/

migrations/
static/
```

## Core Flow

```txt
User creates a group
User adds group members
User adds an expense
App creates equal split records
App calculates balances
App suggests settlements
User marks a settlement as paid
```

Money should be stored as integer cents, not floats. The MVP should use GBP only.

## Development Status

This repository has an initial runnable scaffold for the MVP.

The current app uses an in-memory store so you can run and tweak the Go code before wiring PostgreSQL/sqlc into the request path. Migrations and an initial sqlc config are included for the database stage.

Run the app:

```sh
make run
```

Then open:

```txt
http://localhost:8080
```

Useful commands:

```sh
make fmt
make test
```

The server also accepts a custom port:

```sh
PORT=3000 make run
```

## Current Implementation Notes

- `cmd/web/main.go` starts the web server.
- `internal/app` owns routing, rendering, and page loading.
- `internal/models` contains the domain structs.
- `internal/store` provides the temporary in-memory data store.
- `internal/groups`, `internal/expenses`, `internal/balances`, and `internal/settlements` contain the business logic.
- `internal/views` contains server-rendered HTML templates.
- `static/css/app.css` contains the first-pass UI styles.
- `migrations` and `internal/db/queries` are ready for the PostgreSQL/sqlc stage.

Next likely build steps:

- Add PostgreSQL connection config.
- Replace `internal/store.MemoryStore` with a sqlc-backed store.
- Expand sqlc queries for members, expenses, splits, balances, and settlements.
- Add focused tests for money parsing, equal split rounding, and settlement suggestions.
- Move templates to templ components once the core flow is settled.

## Success Criteria

The MVP is complete when a user can create a group, add members, add expenses, calculate balances, view settlement suggestions, and mark settlements as paid.
