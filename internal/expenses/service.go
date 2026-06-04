package expenses

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/olivercarney/splitkit-go/internal/models"
)

type Store interface {
	CreateExpense(ctx context.Context, input models.CreateExpenseInput) (models.Expense, error)
	ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error)
}

type Service struct {
	Store Store
}

func (s Service) Create(ctx context.Context, input models.CreateExpenseInput) (models.Expense, error) {
	input.Amount = strings.TrimSpace(input.Amount)
	if input.Description == "" {
		return models.Expense{}, errors.New("description is required")
	}
	if input.PaidByID == "" {
		return models.Expense{}, errors.New("payer is required")
	}
	if len(input.SplitWith) == 0 {
		return models.Expense{}, errors.New("choose at least one person to split with")
	}
	if _, err := ParseMoney(input.Amount); err != nil {
		return models.Expense{}, err
	}

	expense, err := s.Store.CreateExpense(ctx, input)
	if err != nil {
		return models.Expense{}, err
	}
	return expense, nil
}

func (s Service) List(ctx context.Context, groupID string) ([]models.Expense, error) {
	return s.Store.ListExpenses(ctx, groupID)
}

func ParseMoney(value string) (int64, error) {
	clean := strings.TrimSpace(strings.ReplaceAll(value, "£", ""))
	if clean == "" {
		return 0, errors.New("amount is required")
	}

	parts := strings.Split(clean, ".")
	if len(parts) > 2 {
		return 0, errors.New("amount must be a valid money value")
	}

	pounds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || pounds < 0 {
		return 0, errors.New("amount must be a valid money value")
	}

	var pence int64
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, errors.New("amount can only have two decimal places")
		}
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		pence, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, errors.New("amount must be a valid money value")
		}
	}

	cents := pounds*100 + pence
	if cents <= 0 {
		return 0, errors.New("amount must be greater than zero")
	}
	return cents, nil
}

func EqualSplits(amountCents int64, participants []string) ([]models.ExpenseSplit, error) {
	if amountCents <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if len(participants) == 0 {
		return nil, errors.New("choose at least one person to split with")
	}

	seen := make(map[string]struct{}, len(participants))
	splitCount := int64(len(participants))
	baseShare := amountCents / splitCount
	remainder := amountCents % splitCount
	splits := make([]models.ExpenseSplit, 0, len(participants))
	for index, userID := range participants {
		if userID == "" {
			return nil, errors.New("split participant is required")
		}
		if _, ok := seen[userID]; ok {
			return nil, fmt.Errorf("duplicate split participant: %s", userID)
		}
		seen[userID] = struct{}{}

		share := baseShare
		if int64(index) < remainder {
			share++
		}
		splits = append(splits, models.ExpenseSplit{
			UserID:      userID,
			AmountCents: share,
		})
	}
	return splits, nil
}
