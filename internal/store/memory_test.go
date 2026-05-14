package store

import (
	"context"
	"testing"

	"github.com/olivercarney/splitkit-go/internal/models"
)

func TestDeleteGroupRemovesRelatedData(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	group, err := store.CreateGroup(ctx, "Trip", models.User{ID: "local-user"})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	member, err := store.AddMember(ctx, group.ID, "Bob")
	if err != nil {
		t.Fatalf("AddMember returned error: %v", err)
	}
	if _, err := store.CreateExpense(ctx, models.CreateExpenseInput{
		GroupID:     group.ID,
		PaidByID:    "local-user",
		Description: "Dinner",
		Amount:      "20.00",
		SplitWith:   []string{"local-user", member.ID},
	}); err != nil {
		t.Fatalf("CreateExpense returned error: %v", err)
	}
	if _, err := store.MarkSettlementPaid(ctx, models.Settlement{
		GroupID:     group.ID,
		FromUserID:  member.ID,
		ToUserID:    "local-user",
		AmountCents: 1000,
	}); err != nil {
		t.Fatalf("MarkSettlementPaid returned error: %v", err)
	}

	if err := store.DeleteGroup(ctx, group.ID); err != nil {
		t.Fatalf("DeleteGroup returned error: %v", err)
	}

	if _, err := store.GetGroup(ctx, group.ID); err == nil {
		t.Fatal("GetGroup expected error after delete")
	}
	if members, _ := store.ListMembers(ctx, group.ID); len(members) != 0 {
		t.Fatalf("members = %d, want 0", len(members))
	}
	if expenses, _ := store.ListExpenses(ctx, group.ID); len(expenses) != 0 {
		t.Fatalf("expenses = %d, want 0", len(expenses))
	}
	if settlements, _ := store.ListSettlements(ctx, group.ID); len(settlements) != 0 {
		t.Fatalf("settlements = %d, want 0", len(settlements))
	}
}

func TestRemoveMemberRemovesRelatedExpensesAndSettlements(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	group, err := store.CreateGroup(ctx, "Trip", models.User{ID: "local-user"})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	bob, err := store.AddMember(ctx, group.ID, "Bob")
	if err != nil {
		t.Fatalf("AddMember Bob returned error: %v", err)
	}
	charlie, err := store.AddMember(ctx, group.ID, "Charlie")
	if err != nil {
		t.Fatalf("AddMember Charlie returned error: %v", err)
	}
	if _, err := store.CreateExpense(ctx, models.CreateExpenseInput{
		GroupID:     group.ID,
		PaidByID:    bob.ID,
		Description: "Taxi",
		Amount:      "15.00",
		SplitWith:   []string{"local-user", bob.ID, charlie.ID},
	}); err != nil {
		t.Fatalf("CreateExpense returned error: %v", err)
	}
	if _, err := store.MarkSettlementPaid(ctx, models.Settlement{
		GroupID:     group.ID,
		FromUserID:  charlie.ID,
		ToUserID:    bob.ID,
		AmountCents: 500,
	}); err != nil {
		t.Fatalf("MarkSettlementPaid returned error: %v", err)
	}

	if err := store.RemoveMember(ctx, group.ID, bob.ID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	members, _ := store.ListMembers(ctx, group.ID)
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	expenses, _ := store.ListExpenses(ctx, group.ID)
	if len(expenses) != 0 {
		t.Fatalf("expenses = %d, want 0 because removed member paid the expense", len(expenses))
	}
	settlements, _ := store.ListSettlements(ctx, group.ID)
	if len(settlements) != 0 {
		t.Fatalf("settlements = %d, want 0 because removed member was involved", len(settlements))
	}
}
