package settlements

import (
	"context"
	"testing"
	"time"

	"github.com/olivecarney/splitkit-go/internal/models"
)

type fakeStore struct {
	input models.Settlement
}

func (s *fakeStore) MarkSettlementPaid(ctx context.Context, input models.Settlement) (models.Settlement, error) {
	s.input = input
	now := time.Now()
	input.ID = "settlement-1"
	input.FromName = "Bob"
	input.ToName = "Alice"
	input.SettledAt = &now
	input.CreatedAt = now
	return input, nil
}

func (s *fakeStore) ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error) {
	return nil, nil
}

func TestMarkPaidRecordsSettlement(t *testing.T) {
	store := &fakeStore{}
	service := Service{Store: store}

	got, err := service.MarkPaid(context.Background(), models.Settlement{
		GroupID:     "group-1",
		FromUserID:  "bob",
		ToUserID:    "alice",
		AmountCents: 1234,
	})
	if err != nil {
		t.Fatalf("MarkPaid returned error: %v", err)
	}

	if store.input.AmountCents != 1234 || store.input.FromUserID != "bob" || store.input.ToUserID != "alice" {
		t.Fatalf("store input = %#v", store.input)
	}
	if got.ID == "" {
		t.Fatal("settlement ID was not set")
	}
	if got.SettledAt == nil {
		t.Fatal("settlement should be marked paid with SettledAt")
	}
}

func TestMarkPaidRejectsInvalidInput(t *testing.T) {
	service := Service{Store: &fakeStore{}}

	tests := []models.Settlement{
		{ToUserID: "alice", AmountCents: 100},
		{FromUserID: "bob", AmountCents: 100},
		{FromUserID: "bob", ToUserID: "alice"},
	}
	for _, input := range tests {
		if _, err := service.MarkPaid(context.Background(), input); err == nil {
			t.Fatalf("MarkPaid(%#v) expected error", input)
		}
	}
}
