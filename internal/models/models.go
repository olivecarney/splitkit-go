package models

import "time"

type User struct {
	ID    string
	Email string
}

type Group struct {
	ID        string
	Name      string
	CreatedBy string
	CreatedAt time.Time
}

type Member struct {
	ID          string
	GroupID     string
	DisplayName string
	CreatedAt   time.Time
}

type Expense struct {
	ID          string
	GroupID     string
	PaidByID    string
	PaidByName  string
	Description string
	AmountCents int64
	Currency    string
	Splits      []ExpenseSplit
	CreatedAt   time.Time
}

type ExpenseSplit struct {
	ExpenseID   string
	UserID      string
	DisplayName string
	AmountCents int64
}

type CreateExpenseInput struct {
	GroupID     string
	PaidByID    string
	Description string
	Amount      string
	SplitWith   []string
}

type Balance struct {
	UserID      string
	DisplayName string
	AmountCents int64
}

type Settlement struct {
	ID          string
	GroupID     string
	FromUserID  string
	FromName    string
	ToUserID    string
	ToName      string
	AmountCents int64
	SettledAt   *time.Time
	CreatedAt   time.Time
}
