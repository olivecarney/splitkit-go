# SplitKit MVP Plan

A lightweight expense-splitting web app built with Go, PostgreSQL, sqlc, templ, TailwindCSS, and htmx.

SplitKit helps small groups track shared expenses, calculate balances, and generate simple settlement suggestions.

The goal is to build a clean, minimal fintech MVP without unnecessary complexity. The app should be simple, fast, server-rendered, and easy to extend later with real authentication, transactional email, and analytics.

---

## Repo Name

Recommended repo name:

```txt
splitkit
```

Why this works:

- Short and memorable
- Fits the idea of splitting expenses
- Sounds lightweight and developer-friendly
- Does not lock the project into being only a Splitwise clone
- Works nicely as a Go project name

Alternative names:

```txt
settleup-go
tabmate
fairshare
splitstack
duesplit
balancer
```

---

## Tech Stack

### Core MVP Stack

```txt
Go
PostgreSQL
sqlc
templ
TailwindCSS
htmx
Alpine.js where useful
```

### Future Integrations

These should not be part of the first build, but the codebase should make them easy to add later:

```txt
Clerk      -> authentication
PostHog    -> analytics
Resend     -> transactional email
Mailgun    -> transactional email alternative
MailerLite -> marketing/email alternative
```

---

## MVP Goal

The MVP should allow a user to:

1. Create a group
2. Add members to the group
3. Add shared expenses
4. Split expenses equally
5. View balances
6. See who owes whom
7. Mark settlements as paid

The focus should be:

```txt
Correctness
Simplicity
Clean structure
Fast development
Easy future integrations
```

---

## MVP Scope

### In Scope

```txt
Groups
Group members
Expenses
Equal splits
Balances
Settlement suggestions
Mark settlements as paid
Basic server-rendered UI
Development-only user
```

### Out of Scope

These should be skipped until the core app works:

```txt
Real authentication
Email invites
Analytics
Payment processing
Bank connections
Open Banking
Multi-currency support
Unequal splits
Percentage splits
Receipt uploads
Mobile app
Push notifications
Advanced permissions
Complex dashboards
```

---

## Core User Flow

The core flow should be:

```txt
User creates a group
↓
User adds group members
↓
User adds an expense
↓
App creates equal split records
↓
App calculates balances
↓
App suggests settlements
↓
User marks settlement as paid
```

Example:

```txt
Alice creates a group called "Croatia Trip".

Alice adds:
- Alice
- Bob
- Charlie

Alice pays £30 for dinner, split between Alice, Bob, and Charlie.

Each person owes £10.

Because Alice paid the full £30:
- Bob owes Alice £10
- Charlie owes Alice £10
```

---

## Suggested Project Structure

```txt
cmd/
  web/
    main.go

internal/
  app/
    server.go
    routes.go

  auth/
    user.go
    middleware.go

  db/
    queries/
    sqlc/

  groups/
    handlers.go
    service.go

  expenses/
    handlers.go
    service.go

  balances/
    service.go

  settlements/
    service.go

  analytics/
    analytics.go

  mailer/
    mailer.go

  views/
    layouts/
    groups/
    expenses/
    components/

migrations/
  001_create_users.sql
  002_create_groups.sql
  003_create_group_members.sql
  004_create_expenses.sql
  005_create_expense_splits.sql
  006_create_settlements.sql

static/
  css/
  js/
```

---

## Database Tables

The MVP should start with these tables:

```txt
users
groups
group_members
expenses
expense_splits
settlements
```

---

## Database Schema Draft

### users

Stores local app users.

Even before adding real auth, the app should still have a users table so Clerk can be added later cleanly.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    external_auth_id TEXT UNIQUE,
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

For development, seed one fake user.

---

### groups

Stores expense groups.

```sql
CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

### group_members

Stores which users belong to which groups.

```sql
CREATE TABLE group_members (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);
```

---

### expenses

Stores expenses added to groups.

```sql
CREATE TABLE expenses (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    paid_by UUID NOT NULL REFERENCES users(id),
    description TEXT NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    currency TEXT NOT NULL DEFAULT 'GBP',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

### expense_splits

Stores how much of each expense belongs to each member.

```sql
CREATE TABLE expense_splits (
    expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    amount_cents INTEGER NOT NULL CHECK (amount_cents >= 0),
    PRIMARY KEY (expense_id, user_id)
);
```

---

### settlements

Stores settlement records.

```sql
CREATE TABLE settlements (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    from_user_id UUID NOT NULL REFERENCES users(id),
    to_user_id UUID NOT NULL REFERENCES users(id),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    settled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

---

## Auth Strategy

The MVP should use a simple development user.

Real authentication can be added later through Clerk.

Create an interface like this:

```go
type CurrentUser struct {
    ID    string
    Email string
}

type UserProvider interface {
    CurrentUser(r *http.Request) (*CurrentUser, error)
}
```

For the MVP:

```go
type DevUserProvider struct{}

func (p DevUserProvider) CurrentUser(r *http.Request) (*CurrentUser, error) {
    return &CurrentUser{
        ID:    "dev-user-1",
        Email: "dev@example.com",
    }, nil
}
```

Later, Clerk can replace this without changing the rest of the app:

```go
type ClerkUserProvider struct {
    // Clerk client/config here
}
```

The important thing is that handlers should depend on the `UserProvider` interface, not directly on Clerk.

---

## Analytics Strategy

PostHog should not be added immediately.

Instead, create a small interface:

```go
type Analytics interface {
    Track(ctx context.Context, userID string, event string, props map[string]any)
}
```

For the MVP:

```go
type NoopAnalytics struct{}

func (a NoopAnalytics) Track(ctx context.Context, userID string, event string, props map[string]any) {}
```

Later, this can be replaced with a PostHog implementation.

Useful future analytics events:

```txt
Group created
Member added
Expense added
Balance viewed
Settlement marked as paid
```

---

## Email Strategy

Emails are not required for the MVP.

Create a simple mailer interface so Resend, Mailgun, or MailerLite can be added later.

```go
type Mailer interface {
    Send(ctx context.Context, to string, subject string, body string) error
}
```

For the MVP:

```go
type NoopMailer struct{}

func (m NoopMailer) Send(ctx context.Context, to string, subject string, body string) error {
    return nil
}
```

Future email use cases:

```txt
Group invitations
Expense added notifications
Settlement reminders
Monthly summaries
```

---

## Pages

The MVP should include these pages:

```txt
/
  Landing page

/groups
  List groups

/groups/new
  Create group

/groups/{id}
  Group dashboard

/groups/{id}/members
  Add or remove members

/groups/{id}/expenses/new
  Add expense

/groups/{id}/balances
  View balances and settlement suggestions
```

The group dashboard should show:

```txt
Group name
Members
Expense list
Add expense button
Balance summary
Settlement suggestions
```

---

## htmx Usage

Use htmx only where it makes the app feel smoother.

Good MVP uses:

```txt
Create group without full page reload
Add member inline
Add expense inline
Refresh balance summary after adding an expense
Mark settlement as paid
```

Avoid making the whole app depend on htmx.

Normal server-rendered pages are fine.

---

## Alpine.js Usage

Use Alpine.js only for small UI interactions.

Good uses:

```txt
Dropdown menus
Modals
Toggle panels
Mobile menu
Simple form state
```

Avoid putting core business logic in Alpine.js.

Expense calculations should happen on the server.

---

## Build Order

### 1. Project Setup

Set up:

```txt
Go module
Basic HTTP server
Environment config
PostgreSQL connection
Static file serving
templ setup
TailwindCSS setup
```

---

### 2. Database and Migrations

Create initial migrations for:

```txt
Users
Groups
Group members
Expenses
Expense splits
Settlements
```

Seed a development user.

---

### 3. sqlc Queries

Add queries for:

```txt
Creating groups
Listing groups
Getting a group by ID
Adding members
Listing group members
Creating expenses
Creating expense splits
Listing expenses
Calculating balances
Creating settlements
Marking settlements as paid
```

---

### 4. Layout and UI

Create:

```txt
Base layout
Navigation
Button component
Form input component
Card component
Table component
Empty state component
```

Keep the design clean and minimal.

---

### 5. Groups

Build:

```txt
Group list page
Create group page
Group detail page
```

A user should be able to create a group and view it.

---

### 6. Members

Build:

```txt
Add member form
Member list
Remove member action if simple enough
```

For the MVP, manual member creation is acceptable.

---

### 7. Expenses

Build:

```txt
Add expense form
Select who paid
Select members included in the split
Store equal split records
Show expense list
```

---

### 8. Balances

Build the balance calculation.

For each group member:

```txt
balance = total_paid - total_share
```

Where:

```txt
total_paid = total amount paid by the member
total_share = total amount owed by the member from expense splits
```

Positive balance means the member is owed money.

Negative balance means the member owes money.

---

### 9. Settlement Suggestions

Build a simple settlement algorithm.

High-level approach:

```txt
1. Find members with positive balances
2. Find members with negative balances
3. Match debtors to creditors
4. Suggest payments until balances reach zero
```

Example output:

```txt
Bob owes Alice £10
Charlie owes Alice £10
```

---

### 10. Mark Settlements as Paid

Allow users to mark a suggested settlement as paid.

This can create or update a row in the `settlements` table.

---

### 11. Polish

Add:

```txt
Better empty states
Form validation
Error messages
Loading states
Responsive layout
Basic tests for balance logic
README screenshots later
```

---

## Financial Logic Rules

### Store Money as Integers

Store money as integers in cents.

Good:

```txt
1250 = £12.50
```

Avoid storing money as floating-point numbers.

Bad:

```txt
12.50 as float
```

---

### Use One Currency for the MVP

Use one currency at first.

Default:

```txt
GBP
```

Multi-currency support can come later.

---

### Equal Split Logic

When an expense is split equally, divide the amount by the number of included members.

If the amount does not divide evenly, handle the remainder carefully.

Example:

```txt
£10.00 split between 3 people

1000 cents / 3 = 333 cents each with 1 cent remaining
```

Possible result:

```txt
Alice: 334
Bob: 333
Charlie: 333
```

The total split must always equal the original expense amount.

---

## Balance Example

Example expense:

```txt
Alice pays £30 for dinner.
The expense is split between Alice, Bob, and Charlie.
```

Each person's share:

```txt
£30 / 3 = £10
```

Balance calculation:

```txt
Alice paid: £30
Alice share: £10
Alice balance: +£20

Bob paid: £0
Bob share: £10
Bob balance: -£10

Charlie paid: £0
Charlie share: £10
Charlie balance: -£10
```

Balance table:

| Member | Balance |
| --- | ---: |
| Alice | +£20 |
| Bob | -£10 |
| Charlie | -£10 |

Settlement suggestions:

| From | To | Amount |
| --- | --- | ---: |
| Bob | Alice | £10 |
| Charlie | Alice | £10 |

---

## Success Criteria

The MVP is complete when:

```txt
A user can create a group
A user can add members
A user can add an expense
The app creates equal splits
The app calculates balances correctly
The app suggests settlements
A user can mark a settlement as paid
The app works cleanly without Clerk, PostHog, or Resend
The codebase has clear places to add those services later
```

---

## Future Features

After the MVP, consider adding:

```txt
Clerk authentication
Email invitations
Resend transactional emails
PostHog analytics
Unequal splits
Percentage splits
Recurring expenses
Multi-currency support
CSV export
Receipt uploads
Activity feed
Public landing page
Dark mode
Mobile-friendly PWA
```

---

## Project Philosophy

SplitKit should stay simple.

The aim is not to clone every feature of Splitwise.

The aim is to build a focused, reliable, well-structured fintech app that demonstrates:

```txt
Clean Go backend design
Type-safe SQL with sqlc
Server-rendered UI with templ
Practical htmx interactions
Correct financial calculations
Good project structure
Easy future integration points
```

Build the boring version first.

Then make it nicer.
