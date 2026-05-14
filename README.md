# SplitKit

SplitKit is a lightweight expense-splitting web app for small groups. It is planned as a clean Go MVP that lets users create groups, add members, record shared expenses, calculate balances, and mark settlements as paid.

The app should stay simple, fast, server-rendered, and easy to extend later with real authentication, analytics, and email.

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
- Analytics
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

Future integrations are expected to be added behind small interfaces:

- Clerk for authentication
- PostHog for analytics
- Resend, Mailgun, or MailerLite for email

## Planned Structure

```txt
cmd/
  web/
    main.go

internal/
  app/
  auth/
  db/
  groups/
  expenses/
  balances/
  settlements/
  analytics/
  mailer/
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

This repository is currently at planning/bootstrap stage. The implementation plan lives in [splitkit-mvp-plan.md](./splitkit-mvp-plan.md).

Expected initial setup:

```sh
go mod init github.com/olivercarney/splitkit-go
```

Once the application is scaffolded, this README should be updated with the actual setup commands for:

- Environment variables
- Database creation and migrations
- sqlc generation
- templ generation
- Tailwind build/watch
- Running the web server
- Running tests

## Success Criteria

The MVP is complete when a user can create a group, add members, add expenses, calculate balances, view settlement suggestions, and mark settlements as paid without needing Clerk, PostHog, or email services.
