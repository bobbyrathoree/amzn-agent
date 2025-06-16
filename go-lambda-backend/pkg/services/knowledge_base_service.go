package services

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
)

// KnowledgeBaseInfo represents a Knowledge Base summary
type KnowledgeBaseInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// KnowledgeBaseService handles Bedrock Knowledge Base operations
type KnowledgeBaseService struct {
	client *bedrockagent.Client
	region string
}

// NewKnowledgeBaseService creates a new KnowledgeBaseService
func NewKnowledgeBaseService(awsConfig aws.Config, region string) *KnowledgeBaseService {
	return &KnowledgeBaseService{
		client: bedrockagent.NewFromConfig(awsConfig),
		region: region,
	}
}

// ListKnowledgeBases retrieves all Knowledge Bases accessible to the current AWS account
func (s *KnowledgeBaseService) ListKnowledgeBases(ctx context.Context, maxResults int32) ([]KnowledgeBaseInfo, error) {
	log.Printf("📚 Listing Knowledge Bases in region: %s", s.region)

	// Set default max results if not specified
	if maxResults == 0 {
		maxResults = 50 // Bedrock default is 10, we'll use 50 for better UX
	}

	input := &bedrockagent.ListKnowledgeBasesInput{
		MaxResults: aws.Int32(maxResults),
	}

	var allKnowledgeBases []KnowledgeBaseInfo
	var nextToken *string

	// Handle pagination
	for {
		if nextToken != nil {
			input.NextToken = nextToken
		}

		resp, err := s.client.ListKnowledgeBases(ctx, input)
		if err != nil {
			log.Printf("❌ Failed to list Knowledge Bases: %v", err)
			return nil, fmt.Errorf("failed to list knowledge bases: %w", err)
		}

		// Process each Knowledge Base
		for _, kb := range resp.KnowledgeBaseSummaries {
			kbInfo := KnowledgeBaseInfo{
				ID:     aws.ToString(kb.KnowledgeBaseId),
				Name:   aws.ToString(kb.Name),
				Status: string(kb.Status),
			}

			// Add optional fields if present
			if kb.Description != nil {
				kbInfo.Description = aws.ToString(kb.Description)
			}
			// Note: CreatedAt and UpdatedAt fields are not available in KnowledgeBaseSummary
			// They would be available in GetKnowledgeBase API response if needed

			allKnowledgeBases = append(allKnowledgeBases, kbInfo)
		}

		// Check if there are more results
		nextToken = resp.NextToken
		if nextToken == nil {
			break
		}

		// Safety check to prevent infinite loops
		if len(allKnowledgeBases) >= int(maxResults) {
			break
		}
	}

	log.Printf("✅ Found %d Knowledge Bases", len(allKnowledgeBases))
	return allKnowledgeBases, nil
}

// GetKnowledgeBase retrieves detailed information about a specific Knowledge Base
func (s *KnowledgeBaseService) GetKnowledgeBase(ctx context.Context, knowledgeBaseID string) (*KnowledgeBaseInfo, error) {
	log.Printf("📚 Getting Knowledge Base details: %s", knowledgeBaseID)

	input := &bedrockagent.GetKnowledgeBaseInput{
		KnowledgeBaseId: aws.String(knowledgeBaseID),
	}

	resp, err := s.client.GetKnowledgeBase(ctx, input)
	if err != nil {
		log.Printf("❌ Failed to get Knowledge Base %s: %v", knowledgeBaseID, err)
		return nil, fmt.Errorf("failed to get knowledge base %s: %w", knowledgeBaseID, err)
	}

	kb := resp.KnowledgeBase
	kbInfo := &KnowledgeBaseInfo{
		ID:     aws.ToString(kb.KnowledgeBaseId),
		Name:   aws.ToString(kb.Name),
		Status: string(kb.Status),
	}

	// Add optional fields if present
	if kb.Description != nil {
		kbInfo.Description = aws.ToString(kb.Description)
	}
	if kb.CreatedAt != nil {
		kbInfo.CreatedAt = kb.CreatedAt.Format("2006-01-02T15:04:05Z")
	}
	if kb.UpdatedAt != nil {
		kbInfo.UpdatedAt = kb.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	log.Printf("✅ Retrieved Knowledge Base: %s (%s)", kbInfo.Name, kbInfo.ID)
	return kbInfo, nil
}

// ValidateKnowledgeBaseAccess checks if a Knowledge Base exists and is accessible
func (s *KnowledgeBaseService) ValidateKnowledgeBaseAccess(ctx context.Context, knowledgeBaseID string) error {
	_, err := s.GetKnowledgeBase(ctx, knowledgeBaseID)
	if err != nil {
		return fmt.Errorf("knowledge base %s is not accessible: %w", knowledgeBaseID, err)
	}
	return nil
}

// GetKnowledgeBasesByStatus filters Knowledge Bases by status
func (s *KnowledgeBaseService) GetKnowledgeBasesByStatus(ctx context.Context, status types.KnowledgeBaseStatus, maxResults int32) ([]KnowledgeBaseInfo, error) {
	allKBs, err := s.ListKnowledgeBases(ctx, maxResults)
	if err != nil {
		return nil, err
	}

	var filtered []KnowledgeBaseInfo
	statusStr := string(status)

	for _, kb := range allKBs {
		if kb.Status == statusStr {
			filtered = append(filtered, kb)
		}
	}

	log.Printf("✅ Found %d Knowledge Bases with status '%s'", len(filtered), statusStr)
	return filtered, nil
}

// GetActiveKnowledgeBases returns only Knowledge Bases that are in ACTIVE status
func (s *KnowledgeBaseService) GetActiveKnowledgeBases(ctx context.Context, maxResults int32) ([]KnowledgeBaseInfo, error) {
	return s.GetKnowledgeBasesByStatus(ctx, types.KnowledgeBaseStatusActive, maxResults)
}