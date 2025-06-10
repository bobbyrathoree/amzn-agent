package services

import (
	"context"
	"errors"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

// ItemService defines the business logic for handling items
type ItemService struct {
	repo repositories.ItemRepository
}

// NewItemService creates a new ItemService
func NewItemService(repo repositories.ItemRepository) *ItemService {
	return &ItemService{
		repo: repo,
	}
}

// GetItem retrieves an item by ID
func (s *ItemService) GetItem(ctx context.Context, id string) (*models.Item, error) {
	if id == "" {
		return nil, errors.New("item ID cannot be empty")
	}

	return s.repo.GetItem(ctx, id)
}

// ListItems retrieves a paginated list of items
func (s *ItemService) ListItems(ctx context.Context, category string, limit int, nextToken string) (*models.ItemsResponse, error) {
	// Default category if not provided
	if category == "" {
		category = "default"
	}

	// Validate and adjust limit
	if limit <= 0 {
		limit = 20 // Default page size
	}
	if limit > 100 {
		limit = 100 // Max page size
	}

	items, token, err := s.repo.ListItems(ctx, category, limit, nextToken)
	if err != nil {
		return nil, err
	}

	return &models.ItemsResponse{
		Items:     items,
		Count:     len(items),
		Total:     len(items), // In a real app, you would get the total count from another query
		NextToken: token,
	}, nil
}

// CreateItem creates a new item
func (s *ItemService) CreateItem(ctx context.Context, req models.CreateItemRequest) (*models.Item, error) {
	// Any additional validation or business logic can go here
	// For example, checking if the category exists, or if the user has permission

	return s.repo.CreateItem(ctx, req)
}

// UpdateItem updates an existing item
func (s *ItemService) UpdateItem(ctx context.Context, id string, req models.UpdateItemRequest) (*models.Item, error) {
	if id == "" {
		return nil, errors.New("item ID cannot be empty")
	}

	// Check if item exists
	_, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	// Any additional validation or business logic can go here

	return s.repo.UpdateItem(ctx, id, req)
}

// DeleteItem removes an item
func (s *ItemService) DeleteItem(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("item ID cannot be empty")
	}

	// Check if item exists
	_, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return err
	}

	// Any additional checks before deletion can go here
	// For example, checking if the user has permission to delete

	return s.repo.DeleteItem(ctx, id)
}