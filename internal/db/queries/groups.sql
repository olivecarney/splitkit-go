-- name: CreateGroup :one
INSERT INTO groups (id, name, created_by)
VALUES ($1, $2, $3)
RETURNING id, name, created_by, created_at;

-- name: CreateUser :one
INSERT INTO users (id, email)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email
RETURNING id, email, created_at;

-- name: ListGroups :many
SELECT id, name, created_by, created_at
FROM groups
ORDER BY created_at DESC;

-- name: GetGroup :one
SELECT id, name, created_by, created_at
FROM groups
WHERE id = $1;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1;

-- name: AddMember :one
INSERT INTO group_members (group_id, user_id, display_name)
VALUES ($1, $2, $3)
RETURNING group_id, user_id, display_name, created_at;

-- name: CountMembers :one
SELECT count(*) FROM group_members
WHERE group_id = $1;

-- name: GetMember :one
SELECT group_id, user_id, display_name, created_at
FROM group_members
WHERE group_id = $1 AND user_id = $2;

-- name: ListMembers :many
SELECT group_id, user_id, display_name, created_at
FROM group_members
WHERE group_id = $1
ORDER BY created_at ASC;

-- name: RemoveMember :exec
DELETE FROM group_members
WHERE group_id = $1 AND user_id = $2;

-- name: DeleteExpensesPaidByMember :exec
DELETE FROM expenses
WHERE group_id = $1 AND paid_by = $2;

-- name: DeleteExpenseSplitsForMember :exec
DELETE FROM expense_splits
WHERE user_id = $2
  AND expense_id IN (
    SELECT id FROM expenses WHERE group_id = $1
  );

-- name: DeleteExpensesWithoutSplits :exec
DELETE FROM expenses e
WHERE e.group_id = $1
  AND NOT EXISTS (
    SELECT 1 FROM expense_splits s WHERE s.expense_id = e.id
  );

-- name: DeleteSettlementsForMember :exec
DELETE FROM settlements
WHERE group_id = $1 AND (from_user_id = $2 OR to_user_id = $2);

-- name: CreateExpense :one
INSERT INTO expenses (id, group_id, paid_by, description, amount_cents, currency)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, group_id, paid_by, description, amount_cents, currency, created_at;

-- name: CreateExpenseSplit :one
INSERT INTO expense_splits (expense_id, user_id, amount_cents)
VALUES ($1, $2, $3)
RETURNING expense_id, user_id, amount_cents;

-- name: ListExpenses :many
SELECT
  e.id,
  e.group_id,
  e.paid_by,
  payer.display_name AS paid_by_name,
  e.description,
  e.amount_cents,
  e.currency,
  e.created_at
FROM expenses e
JOIN group_members payer ON payer.group_id = e.group_id AND payer.user_id = e.paid_by
WHERE e.group_id = $1
ORDER BY e.created_at DESC;

-- name: ListExpenseSplits :many
SELECT
  s.expense_id,
  s.user_id,
  member.display_name,
  s.amount_cents
FROM expense_splits s
JOIN expenses e ON e.id = s.expense_id
JOIN group_members member ON member.group_id = e.group_id AND member.user_id = s.user_id
WHERE e.group_id = $1
ORDER BY s.expense_id, member.created_at ASC;

-- name: CreateSettlement :one
INSERT INTO settlements (id, group_id, from_user_id, to_user_id, amount_cents, settled_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, group_id, from_user_id, to_user_id, amount_cents, settled_at, created_at;

-- name: ListSettlements :many
SELECT
  s.id,
  s.group_id,
  s.from_user_id,
  from_member.display_name AS from_name,
  s.to_user_id,
  to_member.display_name AS to_name,
  s.amount_cents,
  s.settled_at,
  s.created_at
FROM settlements s
JOIN group_members from_member ON from_member.group_id = s.group_id AND from_member.user_id = s.from_user_id
JOIN group_members to_member ON to_member.group_id = s.group_id AND to_member.user_id = s.to_user_id
WHERE s.group_id = $1
ORDER BY s.created_at DESC;
