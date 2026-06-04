package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olivercarney/splitkit-go/internal/db/sqlc"
	"github.com/olivercarney/splitkit-go/internal/expenses"
	"github.com/olivercarney/splitkit-go/internal/models"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (s *PostgresStore) CreateGroup(ctx context.Context, name string, createdBy models.User) (models.Group, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Group{}, err
	}
	defer rollback(ctx, tx)

	q := s.queries.WithTx(tx)
	userID, err := modelID(createdBy.ID)
	if err != nil {
		return models.Group{}, err
	}
	if _, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:    userID,
		Email: userEmail(createdBy),
	}); err != nil {
		return models.Group{}, err
	}

	group, err := q.CreateGroup(ctx, sqlc.CreateGroupParams{
		ID:        newID(),
		Name:      name,
		CreatedBy: userID,
	})
	if err != nil {
		return models.Group{}, err
	}
	if _, err := q.AddMember(ctx, sqlc.AddMemberParams{
		GroupID:     group.ID,
		UserID:      userID,
		DisplayName: "You",
	}); err != nil {
		return models.Group{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Group{}, err
	}
	return groupFromRow(group), nil
}

func (s *PostgresStore) ListGroups(ctx context.Context) ([]models.Group, error) {
	rows, err := s.queries.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	groups := make([]models.Group, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, groupFromRow(row))
	}
	return groups, nil
}

func (s *PostgresStore) GetGroup(ctx context.Context, id string) (models.Group, error) {
	groupID, err := modelID(id)
	if err != nil {
		return models.Group{}, err
	}
	row, err := s.queries.GetGroup(ctx, groupID)
	if err != nil {
		return models.Group{}, notFound(err, "group not found")
	}
	return groupFromRow(row), nil
}

func (s *PostgresStore) DeleteGroup(ctx context.Context, id string) error {
	groupID, err := modelID(id)
	if err != nil {
		return err
	}
	if _, err := s.queries.GetGroup(ctx, groupID); err != nil {
		return notFound(err, "group not found")
	}
	return s.queries.DeleteGroup(ctx, groupID)
}

func (s *PostgresStore) AddMember(ctx context.Context, groupID string, displayName string) (models.Member, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Member{}, err
	}
	defer rollback(ctx, tx)
	q := s.queries.WithTx(tx)

	groupUUID, err := modelID(groupID)
	if err != nil {
		return models.Member{}, err
	}
	if _, err := q.GetGroup(ctx, groupUUID); err != nil {
		return models.Member{}, notFound(err, "group not found")
	}

	memberID := newID()
	if _, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:    memberID,
		Email: fmt.Sprintf("%s@members.splitkit.local", idString(memberID)),
	}); err != nil {
		return models.Member{}, err
	}
	member, err := q.AddMember(ctx, sqlc.AddMemberParams{
		GroupID:     groupUUID,
		UserID:      memberID,
		DisplayName: displayName,
	})
	if err != nil {
		return models.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Member{}, err
	}
	return memberFromRow(member), nil
}

func (s *PostgresStore) RemoveMember(ctx context.Context, groupID string, memberID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(ctx, tx)
	q := s.queries.WithTx(tx)

	groupUUID, err := modelID(groupID)
	if err != nil {
		return err
	}
	memberUUID, err := modelID(memberID)
	if err != nil {
		return err
	}
	if _, err := q.GetGroup(ctx, groupUUID); err != nil {
		return notFound(err, "group not found")
	}
	if count, err := q.CountMembers(ctx, groupUUID); err != nil {
		return err
	} else if count <= 1 {
		return errors.New("a group needs at least one member")
	}
	if _, err := q.GetMember(ctx, sqlc.GetMemberParams{GroupID: groupUUID, UserID: memberUUID}); err != nil {
		return notFound(err, "member not found")
	}
	if err := q.DeleteExpensesPaidByMember(ctx, sqlc.DeleteExpensesPaidByMemberParams{GroupID: groupUUID, PaidBy: memberUUID}); err != nil {
		return err
	}
	if err := q.DeleteExpenseSplitsForMember(ctx, sqlc.DeleteExpenseSplitsForMemberParams{GroupID: groupUUID, UserID: memberUUID}); err != nil {
		return err
	}
	if err := q.DeleteExpensesWithoutSplits(ctx, groupUUID); err != nil {
		return err
	}
	if err := q.DeleteSettlementsForMember(ctx, sqlc.DeleteSettlementsForMemberParams{GroupID: groupUUID, FromUserID: memberUUID}); err != nil {
		return err
	}
	if err := q.RemoveMember(ctx, sqlc.RemoveMemberParams{GroupID: groupUUID, UserID: memberUUID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) ListMembers(ctx context.Context, groupID string) ([]models.Member, error) {
	groupUUID, err := modelID(groupID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListMembers(ctx, groupUUID)
	if err != nil {
		return nil, err
	}
	members := make([]models.Member, 0, len(rows))
	for _, row := range rows {
		members = append(members, memberFromRow(row))
	}
	return members, nil
}

func (s *PostgresStore) CreateExpense(ctx context.Context, input models.CreateExpenseInput) (models.Expense, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Expense{}, err
	}
	defer rollback(ctx, tx)
	q := s.queries.WithTx(tx)

	groupID, err := modelID(input.GroupID)
	if err != nil {
		return models.Expense{}, err
	}
	payerID, err := modelID(input.PaidByID)
	if err != nil {
		return models.Expense{}, err
	}
	if _, err := q.GetGroup(ctx, groupID); err != nil {
		return models.Expense{}, notFound(err, "group not found")
	}
	payer, err := q.GetMember(ctx, sqlc.GetMemberParams{GroupID: groupID, UserID: payerID})
	if err != nil {
		return models.Expense{}, notFound(err, "payer must be a group member")
	}
	amount, err := expenses.ParseMoney(input.Amount)
	if err != nil {
		return models.Expense{}, err
	}
	splits, err := expenses.EqualSplits(amount, input.SplitWith)
	if err != nil {
		return models.Expense{}, err
	}

	expenseRow, err := q.CreateExpense(ctx, sqlc.CreateExpenseParams{
		ID:          newID(),
		GroupID:     groupID,
		PaidBy:      payerID,
		Description: input.Description,
		AmountCents: amount,
		Currency:    "GBP",
	})
	if err != nil {
		return models.Expense{}, err
	}

	expense := expenseFromRow(expenseRow, payer.DisplayName)
	for _, split := range splits {
		userID, err := modelID(split.UserID)
		if err != nil {
			return models.Expense{}, err
		}
		member, err := q.GetMember(ctx, sqlc.GetMemberParams{GroupID: groupID, UserID: userID})
		if err != nil {
			return models.Expense{}, notFound(err, "all split participants must be group members")
		}
		if _, err := q.CreateExpenseSplit(ctx, sqlc.CreateExpenseSplitParams{
			ExpenseID:   expenseRow.ID,
			UserID:      userID,
			AmountCents: split.AmountCents,
		}); err != nil {
			return models.Expense{}, err
		}
		expense.Splits = append(expense.Splits, models.ExpenseSplit{
			ExpenseID:   expense.ID,
			UserID:      split.UserID,
			DisplayName: member.DisplayName,
			AmountCents: split.AmountCents,
		})
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Expense{}, err
	}
	return expense, nil
}

func (s *PostgresStore) ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error) {
	groupUUID, err := modelID(groupID)
	if err != nil {
		return nil, err
	}
	expenseRows, err := s.queries.ListExpenses(ctx, groupUUID)
	if err != nil {
		return nil, err
	}
	splitRows, err := s.queries.ListExpenseSplits(ctx, groupUUID)
	if err != nil {
		return nil, err
	}
	splitsByExpense := make(map[string][]models.ExpenseSplit)
	for _, row := range splitRows {
		expenseID := idString(row.ExpenseID)
		splitsByExpense[expenseID] = append(splitsByExpense[expenseID], models.ExpenseSplit{
			ExpenseID:   expenseID,
			UserID:      idString(row.UserID),
			DisplayName: row.DisplayName,
			AmountCents: row.AmountCents,
		})
	}
	expensesList := make([]models.Expense, 0, len(expenseRows))
	for _, row := range expenseRows {
		expense := expenseFromListRow(row)
		expense.Splits = splitsByExpense[expense.ID]
		expensesList = append(expensesList, expense)
	}
	return expensesList, nil
}

func (s *PostgresStore) MarkSettlementPaid(ctx context.Context, input models.Settlement) (models.Settlement, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Settlement{}, err
	}
	defer rollback(ctx, tx)
	q := s.queries.WithTx(tx)

	groupID, err := modelID(input.GroupID)
	if err != nil {
		return models.Settlement{}, err
	}
	fromID, err := modelID(input.FromUserID)
	if err != nil {
		return models.Settlement{}, err
	}
	toID, err := modelID(input.ToUserID)
	if err != nil {
		return models.Settlement{}, err
	}
	from, err := q.GetMember(ctx, sqlc.GetMemberParams{GroupID: groupID, UserID: fromID})
	if err != nil {
		return models.Settlement{}, notFound(err, "settlement users must be group members")
	}
	to, err := q.GetMember(ctx, sqlc.GetMemberParams{GroupID: groupID, UserID: toID})
	if err != nil {
		return models.Settlement{}, notFound(err, "settlement users must be group members")
	}
	now := time.Now()
	row, err := q.CreateSettlement(ctx, sqlc.CreateSettlementParams{
		ID:          newID(),
		GroupID:     groupID,
		FromUserID:  fromID,
		ToUserID:    toID,
		AmountCents: input.AmountCents,
		SettledAt:   pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return models.Settlement{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Settlement{}, err
	}
	return settlementFromRow(row, from.DisplayName, to.DisplayName), nil
}

func (s *PostgresStore) ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error) {
	groupUUID, err := modelID(groupID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListSettlements(ctx, groupUUID)
	if err != nil {
		return nil, err
	}
	settlements := make([]models.Settlement, 0, len(rows))
	for _, row := range rows {
		settlements = append(settlements, settlementFromListRow(row))
	}
	return settlements, nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func newID() pgtype.UUID {
	return uuidToPg(uuid.New())
}

func modelID(id string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid id: %w", err)
	}
	return uuidToPg(parsed), nil
}

func uuidToPg(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func idString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}

func timeFromPg(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timePtrFromPg(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func notFound(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New(message)
	}
	return err
}

func userEmail(user models.User) string {
	if strings.TrimSpace(user.Email) != "" {
		return strings.TrimSpace(user.Email)
	}
	return fmt.Sprintf("%s@users.splitkit.local", user.ID)
}

func groupFromRow(row sqlc.Group) models.Group {
	return models.Group{
		ID:        idString(row.ID),
		Name:      row.Name,
		CreatedBy: idString(row.CreatedBy),
		CreatedAt: timeFromPg(row.CreatedAt),
	}
}

func memberFromRow(row sqlc.GroupMember) models.Member {
	return models.Member{
		ID:          idString(row.UserID),
		GroupID:     idString(row.GroupID),
		DisplayName: row.DisplayName,
		CreatedAt:   timeFromPg(row.CreatedAt),
	}
}

func expenseFromRow(row sqlc.Expense, payerName string) models.Expense {
	return models.Expense{
		ID:          idString(row.ID),
		GroupID:     idString(row.GroupID),
		PaidByID:    idString(row.PaidBy),
		PaidByName:  payerName,
		Description: row.Description,
		AmountCents: row.AmountCents,
		Currency:    row.Currency,
		CreatedAt:   timeFromPg(row.CreatedAt),
	}
}

func expenseFromListRow(row sqlc.ListExpensesRow) models.Expense {
	return models.Expense{
		ID:          idString(row.ID),
		GroupID:     idString(row.GroupID),
		PaidByID:    idString(row.PaidBy),
		PaidByName:  row.PaidByName,
		Description: row.Description,
		AmountCents: row.AmountCents,
		Currency:    row.Currency,
		CreatedAt:   timeFromPg(row.CreatedAt),
	}
}

func settlementFromRow(row sqlc.Settlement, fromName string, toName string) models.Settlement {
	return models.Settlement{
		ID:          idString(row.ID),
		GroupID:     idString(row.GroupID),
		FromUserID:  idString(row.FromUserID),
		FromName:    fromName,
		ToUserID:    idString(row.ToUserID),
		ToName:      toName,
		AmountCents: row.AmountCents,
		SettledAt:   timePtrFromPg(row.SettledAt),
		CreatedAt:   timeFromPg(row.CreatedAt),
	}
}

func settlementFromListRow(row sqlc.ListSettlementsRow) models.Settlement {
	return models.Settlement{
		ID:          idString(row.ID),
		GroupID:     idString(row.GroupID),
		FromUserID:  idString(row.FromUserID),
		FromName:    row.FromName,
		ToUserID:    idString(row.ToUserID),
		ToName:      row.ToName,
		AmountCents: row.AmountCents,
		SettledAt:   timePtrFromPg(row.SettledAt),
		CreatedAt:   timeFromPg(row.CreatedAt),
	}
}
