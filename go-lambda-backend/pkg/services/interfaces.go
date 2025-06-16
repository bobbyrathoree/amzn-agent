package services

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// CDKDeploymentServiceInterface defines the interface for CDK deployment operations
type CDKDeploymentServiceInterface interface {
	DeployBotKnowledgeBaseStack(ctx context.Context, req CDKDeploymentRequest) (*CDKDeploymentResult, error)
	DestroyBotKnowledgeBaseStack(ctx context.Context, botID string) error
}

// CloudFormationServiceInterface defines the interface for CloudFormation operations
type CloudFormationServiceInterface interface {
	CreateKnowledgeBaseStack(ctx context.Context, req KnowledgeBaseStackRequest) (*KnowledgeBaseStackResult, error)
	GetStackStatus(ctx context.Context, stackName string) (*KnowledgeBaseStackResult, error)
	DeleteStack(ctx context.Context, stackName string) error
}

// StackOutputServiceInterface defines the interface for stack output operations
type StackOutputServiceInterface interface {
	FetchStackOutputs(ctx context.Context, botID, ownerUserID string) (*StackOutputResult, error)
}

// GuardrailServiceInterface defines the interface for guardrail operations
// Following the bedrock-chat pattern of inline guardrails with model calls
type GuardrailServiceInterface interface {
	BuildGuardrailConfig(bot *models.Bot, stream bool) (*types.GuardrailConfiguration, error)
	ValidateGuardrailConfiguration(bot *models.Bot) error
}

// Ensure concrete types implement interfaces
var _ CDKDeploymentServiceInterface = (*CDKDeploymentService)(nil)
var _ CloudFormationServiceInterface = (*CloudFormationService)(nil)
var _ StackOutputServiceInterface = (*StackOutputService)(nil)
var _ GuardrailServiceInterface = (*GuardrailService)(nil)