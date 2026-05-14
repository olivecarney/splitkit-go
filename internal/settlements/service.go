package settlements

import (
	"context"
	"errors"

	"github.com/olivercarney/splitkit-go/internal/models"
)

type Store interface {
	MarkSettlementPaid(ctx context.Context, input models.Settlement) (models.Settlement, error)
	ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error)
}

type Service struct {
	Store Store
}

func (s Service) MarkPaid(ctx context.Context, input models.Settlement) (models.Settlement, error) {
	if input.FromUserID == "" || input.ToUserID == "" {
		return models.Settlement{}, errors.New("settlement needs a payer and recipient")
	}
	if input.AmountCents <= 0 {
		return models.Settlement{}, errors.New("settlement amount must be greater than zero")
	}
	settlement, err := s.Store.MarkSettlementPaid(ctx, input)
	if err != nil {
		return models.Settlement{}, err
	}
	return settlement, nil
}

func (s Service) List(ctx context.Context, groupID string) ([]models.Settlement, error) {
	return s.Store.ListSettlements(ctx, groupID)
}
