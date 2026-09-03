package balances

import (
	"context"
	"sort"

	"github.com/olivecarney/splitkit-go/internal/models"
)

type Store interface {
	ListMembers(ctx context.Context, groupID string) ([]models.Member, error)
	ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error)
	ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error)
}

type Service struct {
	Store Store
}

func (s Service) ForGroup(ctx context.Context, groupID string) ([]models.Balance, []models.Settlement, error) {
	members, err := s.Store.ListMembers(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	settlements, err := s.Store.ListSettlements(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	expenses, err := s.Store.ListExpenses(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}

	balancesByUser := make(map[string]models.Balance, len(members))
	for _, member := range members {
		balancesByUser[member.ID] = models.Balance{
			UserID:      member.ID,
			DisplayName: member.DisplayName,
		}
	}

	for _, expense := range expenses {
		payer := balancesByUser[expense.PaidByID]
		payer.AmountCents += expense.AmountCents
		balancesByUser[expense.PaidByID] = payer

		for _, split := range expense.Splits {
			balance := balancesByUser[split.UserID]
			balance.AmountCents -= split.AmountCents
			balancesByUser[split.UserID] = balance
		}
	}

	for _, settlement := range settlements {
		from := balancesByUser[settlement.FromUserID]
		from.AmountCents += settlement.AmountCents
		balancesByUser[settlement.FromUserID] = from

		to := balancesByUser[settlement.ToUserID]
		to.AmountCents -= settlement.AmountCents
		balancesByUser[settlement.ToUserID] = to
	}

	list := make([]models.Balance, 0, len(balancesByUser))
	for _, balance := range balancesByUser {
		list = append(list, balance)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].DisplayName < list[j].DisplayName
	})

	return list, suggestSettlements(groupID, list), nil
}

func suggestSettlements(groupID string, balances []models.Balance) []models.Settlement {
	var debtors []models.Balance
	var creditors []models.Balance

	for _, balance := range balances {
		switch {
		case balance.AmountCents < 0:
			balance.AmountCents = -balance.AmountCents
			debtors = append(debtors, balance)
		case balance.AmountCents > 0:
			creditors = append(creditors, balance)
		}
	}

	var suggestions []models.Settlement
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		amount := min64(debtors[i].AmountCents, creditors[j].AmountCents)
		suggestions = append(suggestions, models.Settlement{
			GroupID:     groupID,
			FromUserID:  debtors[i].UserID,
			FromName:    debtors[i].DisplayName,
			ToUserID:    creditors[j].UserID,
			ToName:      creditors[j].DisplayName,
			AmountCents: amount,
		})

		debtors[i].AmountCents -= amount
		creditors[j].AmountCents -= amount
		if debtors[i].AmountCents == 0 {
			i++
		}
		if creditors[j].AmountCents == 0 {
			j++
		}
	}

	return suggestions
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
