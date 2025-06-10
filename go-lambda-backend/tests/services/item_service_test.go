package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
)

// MockItemRepository is a mock implementation of the ItemRepository interface
type MockItemRepository struct {
	GetItemFunc    func(ctx context.Context, id string) (*models.Item, error)
	ListItemsFunc  func(ctx context.Context, category string, limit int, nextToken string) ([]models.Item, string, error)
	CreateItemFunc func(ctx context.Context, req models.CreateItemRequest) (*models.Item, error)
	UpdateItemFunc func(ctx context.Context, id string, req models.UpdateItemRequest) (*models.Item, error)
	DeleteItemFunc func(ctx context.Context, id string) error
}

func (m *MockItemRepository) GetItem(ctx context.Context, id string) (*models.Item, error) {
	return m.GetItemFunc(ctx, id)
}

func (m *MockItemRepository) ListItems(ctx context.Context, category string, limit int, nextToken string) ([]models.Item, string, error) {
	return m.ListItemsFunc(ctx, category, limit, nextToken)
}

func (m *MockItemRepository) CreateItem(ctx context.Context, req models.CreateItemRequest) (*models.Item, error) {
	return m.CreateItemFunc(ctx, req)
}

func (m *MockItemRepository) UpdateItem(ctx context.Context, id string, req models.UpdateItemRequest) (*models.Item, error) {
	return m.UpdateItemFunc(ctx, id, req)
}

func (m *MockItemRepository) DeleteItem(ctx context.Context, id string) error {
	return m.DeleteItemFunc(ctx, id)
}

func TestGetItem(t *testing.T) {
	mockRepo := &MockItemRepository{
		GetItemFunc: func(ctx context.Context, id string) (*models.Item, error) {
			if id == "existing-id" {
				return &models.Item{
					ID:          "existing-id",
					Name:        "Test Item",
					Description: "This is a test item",
					Status:      "active",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			}
			return nil, errors.New("item not found")
		},
	}

	service := services.NewItemService(mockRepo)

	// Test successful retrieval
	item, err := service.GetItem(context.Background(), "existing-id")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if item.ID != "existing-id" {
		t.Errorf("Expected item ID 'existing-id', got '%s'", item.ID)
	}

	// Test not found case
	_, err = service.GetItem(context.Background(), "non-existing-id")
	if err == nil {
		t.Error("Expected error for non-existing item, got nil")
	}

	// Test empty ID
	_, err = service.GetItem(context.Background(), "")
	if err == nil {
		t.Error("Expected error for empty ID, got nil")
	}
}

func TestCreateItem(t *testing.T) {
	mockRepo := &MockItemRepository{
		CreateItemFunc: func(ctx context.Context, req models.CreateItemRequest) (*models.Item, error) {
			return &models.Item{
				ID:          "new-id",
				Name:        req.Name,
				Description: req.Description,
				Category:    req.Category,
				Status:      "active",
				Metadata: models.Metadata{
					Tags:       req.Tags,
					Attributes: req.Attributes,
					UserID:     "test-user",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	service := services.NewItemService(mockRepo)

	// Test successful creation
	createReq := models.CreateItemRequest{
		Name:        "New Test Item",
		Description: "A new test item",
		Category:    "test",
		Tags:        []string{"test", "example"},
	}

	item, err := service.CreateItem(context.Background(), createReq)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if item.Name != createReq.Name {
		t.Errorf("Expected item name '%s', got '%s'", createReq.Name, item.Name)
	}
	if item.Category != createReq.Category {
		t.Errorf("Expected category '%s', got '%s'", createReq.Category, item.Category)
	}
}

func TestListItems(t *testing.T) {
	testItems := []models.Item{
		{
			ID:       "id1",
			Name:     "Item 1",
			Category: "category1",
		},
		{
			ID:       "id2",
			Name:     "Item 2",
			Category: "category1",
		},
	}

	mockRepo := &MockItemRepository{
		ListItemsFunc: func(ctx context.Context, category string, limit int, nextToken string) ([]models.Item, string, error) {
			if category == "category1" {
				return testItems, "", nil
			}
			return []models.Item{}, "", nil
		},
	}

	service := services.NewItemService(mockRepo)

	// Test with results
	response, err := service.ListItems(context.Background(), "category1", 10, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(response.Items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(response.Items))
	}

	// Test with no results
	response, err = service.ListItems(context.Background(), "empty-category", 10, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(response.Items) != 0 {
		t.Errorf("Expected 0 items for empty category, got %d", len(response.Items))
	}

	// Test with default category
	response, err = service.ListItems(context.Background(), "", 10, "")
	if err != nil {
		t.Errorf("Expected no error for default category, got %v", err)
	}

	// Test with limits
	_, err = service.ListItems(context.Background(), "category1", 0, "")
	if err != nil {
		t.Errorf("Expected no error for zero limit (should use default), got %v", err)
	}

	_, err = service.ListItems(context.Background(), "category1", 150, "")
	if err != nil {
		t.Errorf("Expected no error for limit > max (should cap at max), got %v", err)
	}
}