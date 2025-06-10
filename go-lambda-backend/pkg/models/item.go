package models

import (
	"time"
)

// Item represents a basic data model in the application
type Item struct {
	ID          string    `json:"id" dynamodbav:"id"`
	Name        string    `json:"name" dynamodbav:"name" validate:"required"`
	Description string    `json:"description" dynamodbav:"description"`
	Category    string    `json:"category" dynamodbav:"category" validate:"required"`
	Status      string    `json:"status" dynamodbav:"status" validate:"oneof=active inactive pending"`
	Metadata    Metadata  `json:"metadata" dynamodbav:"metadata"`
	CreatedAt   time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
}

// Metadata contains additional information about an Item
type Metadata struct {
	Tags        []string          `json:"tags" dynamodbav:"tags"`
	Attributes  map[string]string `json:"attributes" dynamodbav:"attributes"`
	UserID      string            `json:"userId" dynamodbav:"userId"`
	VersionInfo VersionInfo       `json:"versionInfo" dynamodbav:"versionInfo"`
}

// VersionInfo tracks version-related details
type VersionInfo struct {
	Version     int       `json:"version" dynamodbav:"version"`
	PublishedAt time.Time `json:"publishedAt" dynamodbav:"publishedAt"`
	ChangedBy   string    `json:"changedBy" dynamodbav:"changedBy"`
}

// CreateItemRequest represents the input to create a new item
type CreateItemRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	Category    string   `json:"category" validate:"required"`
	Tags        []string `json:"tags"`
	Attributes  map[string]string `json:"attributes"`
}

// UpdateItemRequest represents the input to update an existing item
type UpdateItemRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Category    *string   `json:"category"`
	Status      *string   `json:"status" validate:"omitempty,oneof=active inactive pending"`
	Tags        []string  `json:"tags"`
	Attributes  map[string]string `json:"attributes"`
}

// ItemResponse represents the API response structure
type ItemResponse struct {
	Item      *Item  `json:"item,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// ItemsResponse represents a paginated list of items
type ItemsResponse struct {
	Items     []Item `json:"items"`
	Count     int    `json:"count"`
	Total     int    `json:"total"`
	NextToken string `json:"nextToken,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}