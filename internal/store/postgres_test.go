package store

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olivecarney/splitkit-go/internal/db"
	"github.com/olivecarney/splitkit-go/internal/models"
)

func TestPostgresStoreExpenseAndSettlementFlow(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New returned error: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("Migrate returned error: %v", err)
	}

	store := NewPostgresStore(pool)
	devUserID := uuid.NewString()
	group, err := store.CreateGroup(ctx, "Integration Trip", models.User{
		ID:    devUserID,
		Email: devUserID + "@tests.splitkit.local",
	})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	defer func() {
		if err := store.DeleteGroup(ctx, group.ID); err != nil {
			t.Logf("cleanup DeleteGroup returned error: %v", err)
		}
	}()

	bob, err := store.AddMember(ctx, group.ID, "Bob")
	if err != nil {
		t.Fatalf("AddMember Bob returned error: %v", err)
	}
	charlie, err := store.AddMember(ctx, group.ID, "Charlie")
	if err != nil {
		t.Fatalf("AddMember Charlie returned error: %v", err)
	}
	expense, err := store.CreateExpense(ctx, models.CreateExpenseInput{
		GroupID:     group.ID,
		PaidByID:    devUserID,
		Description: "Dinner",
		Amount:      "10.00",
		SplitWith:   []string{devUserID, bob.ID, charlie.ID},
	})
	if err != nil {
		t.Fatalf("CreateExpense returned error: %v", err)
	}
	wantSplits := []int64{334, 333, 333}
	for i, want := range wantSplits {
		if expense.Splits[i].AmountCents != want {
			t.Fatalf("split %d = %d, want %d", i, expense.Splits[i].AmountCents, want)
		}
	}

	settlement, err := store.MarkSettlementPaid(ctx, models.Settlement{
		GroupID:     group.ID,
		FromUserID:  bob.ID,
		ToUserID:    devUserID,
		AmountCents: 333,
	})
	if err != nil {
		t.Fatalf("MarkSettlementPaid returned error: %v", err)
	}
	if settlement.SettledAt == nil {
		t.Fatal("settlement should have SettledAt")
	}
}
