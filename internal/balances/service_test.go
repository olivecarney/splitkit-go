package balances

import (
	"context"
	"reflect"
	"testing"

	"github.com/olivecarney/splitkit-go/internal/models"
)

type fakeStore struct {
	members     []models.Member
	expenses    []models.Expense
	settlements []models.Settlement
}

func (s fakeStore) ListMembers(ctx context.Context, groupID string) ([]models.Member, error) {
	return s.members, nil
}

func (s fakeStore) ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error) {
	return s.expenses, nil
}

func (s fakeStore) ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error) {
	return s.settlements, nil
}

func TestForGroupAppliesPaidSettlements(t *testing.T) {
	service := Service{
		Store: fakeStore{
			members: []models.Member{
				{ID: "alice", DisplayName: "Alice"},
				{ID: "bob", DisplayName: "Bob"},
			},
			expenses: []models.Expense{
				{
					GroupID:     "group-1",
					PaidByID:    "alice",
					AmountCents: 2000,
					Splits: []models.ExpenseSplit{
						{UserID: "alice", AmountCents: 1000},
						{UserID: "bob", AmountCents: 1000},
					},
				},
			},
			settlements: []models.Settlement{
				{GroupID: "group-1", FromUserID: "bob", ToUserID: "alice", AmountCents: 1000},
			},
		},
	}

	balances, suggestions, err := service.ForGroup(context.Background(), "group-1")
	if err != nil {
		t.Fatalf("ForGroup returned error: %v", err)
	}

	wantBalances := []models.Balance{
		{UserID: "alice", DisplayName: "Alice", AmountCents: 0},
		{UserID: "bob", DisplayName: "Bob", AmountCents: 0},
	}
	if !reflect.DeepEqual(balances, wantBalances) {
		t.Fatalf("balances = %#v, want %#v", balances, wantBalances)
	}
	if len(suggestions) != 0 {
		t.Fatalf("suggestions = %#v, want none", suggestions)
	}
}

func TestSuggestSettlements(t *testing.T) {
	got := suggestSettlements("group-1", []models.Balance{
		{UserID: "alice", DisplayName: "Alice", AmountCents: 2000},
		{UserID: "bob", DisplayName: "Bob", AmountCents: -1000},
		{UserID: "charlie", DisplayName: "Charlie", AmountCents: -1000},
	})

	want := []models.Settlement{
		{GroupID: "group-1", FromUserID: "bob", FromName: "Bob", ToUserID: "alice", ToName: "Alice", AmountCents: 1000},
		{GroupID: "group-1", FromUserID: "charlie", FromName: "Charlie", ToUserID: "alice", ToName: "Alice", AmountCents: 1000},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("suggestSettlements() = %#v, want %#v", got, want)
	}
}
