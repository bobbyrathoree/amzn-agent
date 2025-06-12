package services

import (
	"context"
)

// CDKDeploymentServiceInterface defines the interface for CDK deployment operations
type CDKDeploymentServiceInterface interface {
	DeployBotKnowledgeBaseStack(ctx context.Context, req CDKDeploymentRequest) (*CDKDeploymentResult, error)
	DestroyBotKnowledgeBaseStack(ctx context.Context, botID string) error
}

// StackOutputServiceInterface defines the interface for stack output operations
type StackOutputServiceInterface interface {
	FetchStackOutputs(ctx context.Context, botID, ownerUserID string) (*StackOutputResult, error)
}

// Ensure concrete types implement interfaces
var _ CDKDeploymentServiceInterface = (*CDKDeploymentService)(nil)
var _ StackOutputServiceInterface = (*StackOutputService)(nil)