package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/olivercarney/splitkit-go/internal/expenses"
	"github.com/olivercarney/splitkit-go/internal/models"
)

type MemoryStore struct {
	mu          sync.RWMutex
	nextID      int
	groups      map[string]models.Group
	members     map[string][]models.Member
	expenses    map[string][]models.Expense
	settlements map[string][]models.Settlement
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nextID:      1,
		groups:      make(map[string]models.Group),
		members:     make(map[string][]models.Member),
		expenses:    make(map[string][]models.Expense),
		settlements: make(map[string][]models.Settlement),
	}
}

func (s *MemoryStore) CreateGroup(ctx context.Context, name string, createdBy models.User) (models.Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	group := models.Group{
		ID:        s.id("group"),
		Name:      name,
		CreatedBy: createdBy.ID,
		CreatedAt: time.Now(),
	}
	s.groups[group.ID] = group
	s.members[group.ID] = append(s.members[group.ID], models.Member{
		ID:          createdBy.ID,
		GroupID:     group.ID,
		DisplayName: "You",
		CreatedAt:   time.Now(),
	})
	return group, nil
}

func (s *MemoryStore) ListGroups(ctx context.Context) ([]models.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := make([]models.Group, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].CreatedAt.After(groups[j].CreatedAt)
	})
	return groups, nil
}

func (s *MemoryStore) GetGroup(ctx context.Context, id string) (models.Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	group, ok := s.groups[id]
	if !ok {
		return models.Group{}, errors.New("group not found")
	}
	return group, nil
}

func (s *MemoryStore) DeleteGroup(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[id]; !ok {
		return errors.New("group not found")
	}
	delete(s.groups, id)
	delete(s.members, id)
	delete(s.expenses, id)
	delete(s.settlements, id)
	return nil
}

func (s *MemoryStore) AddMember(ctx context.Context, groupID string, displayName string) (models.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[groupID]; !ok {
		return models.Member{}, errors.New("group not found")
	}
	member := models.Member{
		ID:          s.id("member"),
		GroupID:     groupID,
		DisplayName: displayName,
		CreatedAt:   time.Now(),
	}
	s.members[groupID] = append(s.members[groupID], member)
	return member, nil
}

func (s *MemoryStore) RemoveMember(ctx context.Context, groupID string, memberID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[groupID]; !ok {
		return errors.New("group not found")
	}
	if len(s.members[groupID]) <= 1 {
		return errors.New("a group needs at least one member")
	}

	members := s.members[groupID]
	index := -1
	for i, member := range members {
		if member.ID == memberID {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("member not found")
	}

	s.members[groupID] = append(members[:index], members[index+1:]...)
	s.removeMemberExpenses(groupID, memberID)
	s.removeMemberSettlements(groupID, memberID)
	return nil
}

func (s *MemoryStore) ListMembers(ctx context.Context, groupID string) ([]models.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := append([]models.Member(nil), s.members[groupID]...)
	sort.Slice(members, func(i, j int) bool {
		return members[i].CreatedAt.Before(members[j].CreatedAt)
	})
	return members, nil
}

func (s *MemoryStore) CreateExpense(ctx context.Context, input models.CreateExpenseInput) (models.Expense, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[input.GroupID]; !ok {
		return models.Expense{}, errors.New("group not found")
	}
	amount, err := expenses.ParseMoney(input.Amount)
	if err != nil {
		return models.Expense{}, err
	}
	membersByID := make(map[string]models.Member)
	for _, member := range s.members[input.GroupID] {
		membersByID[member.ID] = member
	}
	payer, ok := membersByID[input.PaidByID]
	if !ok {
		return models.Expense{}, errors.New("payer must be a group member")
	}

	splitCount := int64(len(input.SplitWith))
	baseShare := amount / splitCount
	remainder := amount % splitCount
	splits := make([]models.ExpenseSplit, 0, len(input.SplitWith))
	for index, userID := range input.SplitWith {
		member, ok := membersByID[userID]
		if !ok {
			return models.Expense{}, errors.New("all split participants must be group members")
		}
		share := baseShare
		if int64(index) < remainder {
			share++
		}
		splits = append(splits, models.ExpenseSplit{
			UserID:      userID,
			DisplayName: member.DisplayName,
			AmountCents: share,
		})
	}

	expense := models.Expense{
		ID:          s.id("expense"),
		GroupID:     input.GroupID,
		PaidByID:    payer.ID,
		PaidByName:  payer.DisplayName,
		Description: input.Description,
		AmountCents: amount,
		Currency:    "GBP",
		Splits:      splits,
		CreatedAt:   time.Now(),
	}
	for i := range expense.Splits {
		expense.Splits[i].ExpenseID = expense.ID
	}
	s.expenses[input.GroupID] = append(s.expenses[input.GroupID], expense)
	return expense, nil
}

func (s *MemoryStore) ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	expenses := append([]models.Expense(nil), s.expenses[groupID]...)
	sort.Slice(expenses, func(i, j int) bool {
		return expenses[i].CreatedAt.After(expenses[j].CreatedAt)
	})
	return expenses, nil
}

func (s *MemoryStore) MarkSettlementPaid(ctx context.Context, input models.Settlement) (models.Settlement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	membersByID := make(map[string]models.Member)
	for _, member := range s.members[input.GroupID] {
		membersByID[member.ID] = member
	}
	from, fromOK := membersByID[input.FromUserID]
	to, toOK := membersByID[input.ToUserID]
	if !fromOK || !toOK {
		return models.Settlement{}, errors.New("settlement users must be group members")
	}

	now := time.Now()
	settlement := models.Settlement{
		ID:          s.id("settlement"),
		GroupID:     input.GroupID,
		FromUserID:  input.FromUserID,
		FromName:    from.DisplayName,
		ToUserID:    input.ToUserID,
		ToName:      to.DisplayName,
		AmountCents: input.AmountCents,
		SettledAt:   &now,
		CreatedAt:   now,
	}
	s.settlements[input.GroupID] = append(s.settlements[input.GroupID], settlement)
	return settlement, nil
}

func (s *MemoryStore) ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	settlements := append([]models.Settlement(nil), s.settlements[groupID]...)
	sort.Slice(settlements, func(i, j int) bool {
		return settlements[i].CreatedAt.After(settlements[j].CreatedAt)
	})
	return settlements, nil
}

func (s *MemoryStore) id(prefix string) string {
	id := fmt.Sprintf("%s-%d", prefix, s.nextID)
	s.nextID++
	return id
}

func (s *MemoryStore) removeMemberExpenses(groupID string, memberID string) {
	kept := make([]models.Expense, 0, len(s.expenses[groupID]))
	for _, expense := range s.expenses[groupID] {
		if expense.PaidByID == memberID {
			continue
		}

		splits := expense.Splits[:0]
		for _, split := range expense.Splits {
			if split.UserID != memberID {
				splits = append(splits, split)
			}
		}
		if len(splits) == 0 {
			continue
		}
		expense.Splits = splits
		kept = append(kept, expense)
	}
	s.expenses[groupID] = kept
}

func (s *MemoryStore) removeMemberSettlements(groupID string, memberID string) {
	kept := make([]models.Settlement, 0, len(s.settlements[groupID]))
	for _, settlement := range s.settlements[groupID] {
		if settlement.FromUserID == memberID || settlement.ToUserID == memberID {
			continue
		}
		kept = append(kept, settlement)
	}
	s.settlements[groupID] = kept
}
