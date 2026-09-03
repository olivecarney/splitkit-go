package groups

import (
	"context"
	"errors"

	"github.com/olivecarney/splitkit-go/internal/models"
)

type Store interface {
	CreateGroup(ctx context.Context, name string, createdBy models.User) (models.Group, error)
	ListGroups(ctx context.Context) ([]models.Group, error)
	GetGroup(ctx context.Context, id string) (models.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	AddMember(ctx context.Context, groupID string, displayName string) (models.Member, error)
	RemoveMember(ctx context.Context, groupID string, memberID string) error
	ListMembers(ctx context.Context, groupID string) ([]models.Member, error)
}

type Service struct {
	Store Store
}

func (s Service) Create(ctx context.Context, name string, user models.User) (models.Group, error) {
	if name == "" {
		return models.Group{}, errors.New("group name is required")
	}
	group, err := s.Store.CreateGroup(ctx, name, user)
	if err != nil {
		return models.Group{}, err
	}
	return group, nil
}

func (s Service) List(ctx context.Context) ([]models.Group, error) {
	return s.Store.ListGroups(ctx)
}

func (s Service) Get(ctx context.Context, id string) (models.Group, error) {
	return s.Store.GetGroup(ctx, id)
}

func (s Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("group id is required")
	}
	return s.Store.DeleteGroup(ctx, id)
}

func (s Service) AddMember(ctx context.Context, groupID string, displayName string) (models.Member, error) {
	if displayName == "" {
		return models.Member{}, errors.New("member name is required")
	}
	member, err := s.Store.AddMember(ctx, groupID, displayName)
	if err != nil {
		return models.Member{}, err
	}
	return member, nil
}

func (s Service) RemoveMember(ctx context.Context, groupID string, memberID string) error {
	if memberID == "" {
		return errors.New("member id is required")
	}
	return s.Store.RemoveMember(ctx, groupID, memberID)
}

func (s Service) Members(ctx context.Context, groupID string) ([]models.Member, error) {
	return s.Store.ListMembers(ctx, groupID)
}
