# SplitKit

SplitKit is a Go/PostgreSQL expense-splitting backend with a small server-rendered UI. It lets a development user create groups, add members, record shared expenses, calculate balances, get settlement suggestions, and mark settlements as paid.

## Problem

Small groups often need to settle a shared trip, household bill, or dinner without tracking every payment manually. SplitKit records who paid, who participated in each equal split, and what has already been settled so the app can answer two questions:

- What is each member's current net balance?
- What is the smallest practical set of transfers that clears those balances?

The MVP deliberately avoids payment processing, real authentication, bank connections, receipt uploads, and multi-currency support.

## Architecture

```txt
cmd/
  web/
    main.go              # boot, PostgreSQL pool, migrations, HTTP server

internal/
  app/                   # routing, form handling, template rendering
  db/
    migrate.go           # simple ordered SQL migration runner
    queries/             # sqlc query source
    sqlc/                # generated sqlc package
  groups/                # group/member service validation
  expenses/              # money parsing and equal-split logic
  balances/              # balance calculation and settlement optimisation
  settlements/           # paid-settlement service validation
  store/                 # PostgreSQL adapter implementing app store interfaces
  views/                 # server-rendered HTML templates

migrations/
static/                  # CSS
```

The app uses `pgx` for PostgreSQL access and `sqlc` for typed query methods. `internal/store.PostgresStore` is the adapter between generated database rows and domain structs in `internal/models`.

## Schema

The schema is in [migrations](migrations):

- `users`: development/local users plus generated group-member users.
- `groups`: expense groups, owned by a user.
- `group_members`: membership display names scoped to a group.
- `expenses`: one row per paid expense, with `amount_cents BIGINT`.
- `expense_splits`: integer-cent split rows linked to an expense.
- `settlements`: paid transfers between members, also stored as integer cents.

Group deletion cascades through expenses, splits, members, and settlements. Member removal deletes expenses paid by that member, removes their split rows from remaining expenses, deletes expenses with no remaining splits, and removes settlements involving that member.

## Local Development

Start PostgreSQL:

```sh
make db-up
```

Run the app:

```sh
make run
```

Then open:

```txt
http://localhost:8080
```

The default connection string is:

```txt
postgres://splitkit:splitkit@localhost:5432/splitkit?sslmode=disable
```

Override it with `DATABASE_URL`. The server applies migrations from `migrations/` on startup.

Useful commands:

```sh
make test
make fmt
make sqlc
make db-down
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

## Settlement Optimisation

Balances are calculated by adding paid expense amounts to payers, subtracting split shares from participants, then applying paid settlements. Settlement suggestions split members into debtors and creditors, then greedily match the next debtor to the next creditor until all non-zero balances are cleared. This minimises the number of suggested transfers for the current balance list without changing the underlying expense history.

## Financial Correctness Notes

- Money is parsed from decimal strings into integer cents. Floats are never used.
- Inputs must be positive and can have at most two decimal places.
- Stored money columns are `BIGINT` to match Go `int64`.
- Equal split rounding distributes any one-cent remainder to the earliest selected participants, so split rows always sum exactly to the original expense.
- GBP is the only supported currency in the MVP.
- Paid settlements are stored as immutable transfer records with `settled_at`; they are applied to future balance calculations.

## Trade-Offs

- A development-only user keeps the MVP focused on expense correctness; real auth can be added later without changing expense math.
- Equal splits are the only supported split type. Unequal splits would require explicit per-participant amounts and additional validation.
- Startup migrations are intentionally simple and work for local development. Production deployments may want a dedicated migration tool and rollback policy.
- Settlement suggestions are deterministic and simple, but they do not consider user preferences, payment rails, or partial payments.
- The UI is server-rendered HTML for speed and simplicity rather than a separate frontend application.

## Screenshots/GIF

Add screenshots or a short GIF under `docs/screenshots/` once the UI flow is captured. Suggested captures:

- Group dashboard with balances and settlement suggestions.
- Expense creation form showing equal split participants.
- Paid settlement history after marking a suggestion as paid.

## CI

GitHub Actions runs:

```sh
go test ./...
```

See [.github/workflows/test.yml](.github/workflows/test.yml).
