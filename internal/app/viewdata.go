package app

import "github.com/olivercarney/splitkit-go/internal/models"

type pageData struct {
	Title       string
	FocusMember bool
	Groups      []models.Group
	Group       models.Group
	Members     []models.Member
	Expenses    []models.Expense
	Balances    []models.Balance
	Suggestions []models.Settlement
	Settlements []models.Settlement
}
