package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// StackDeploymentService handles CloudFormation stack operations for bot Knowledge Bases
type StackDeploymentService struct {
	cfnClient *cloudformation.Client
	envPrefix string
	region    string
}

// NewStackDeploymentService creates a new stack deployment service
func NewStackDeploymentService(cfg aws.Config, envPrefix, region string) *StackDeploymentService {
	return &StackDeploymentService{
		cfnClient: cloudformation.NewFromConfig(cfg),
		envPrefix: envPrefix,
		region:    region,
	}
}

// StackDeploymentRequest contains parameters for deploying a bot Knowledge Base stack
type StackDeploymentRequest struct {
	BotID                    string
	OwnerUserID              string
	Instruction              string
	ExistingKnowledgeBaseID  *string
	KnowledgeBaseCreation    *models.KnowledgeBaseCreationConfig
}

// StackDeploymentResult contains the result of a stack deployment
type StackDeploymentResult struct {
	StackName            string                 `json:"stackName"`
	StackID              string                 `json:"stackId"`
	StackStatus          string                 `json:"stackStatus"`
	KnowledgeBaseID      *string                `json:"knowledgeBaseId,omitempty"`
	DataSourceID         *string                `json:"dataSourceId,omitempty"`
	DocumentBucketName   *string                `json:"documentBucketName,omitempty"`
	GuardrailArn         *string                `json:"guardrailArn,omitempty"`
	GuardrailVersion     *string                `json:"guardrailVersion,omitempty"`
	Outputs              map[string]string      `json:"outputs,omitempty"`
}

// DeployBotKnowledgeBaseStack deploys a CloudFormation stack for a bot's Knowledge Base
func (s *StackDeploymentService) DeployBotKnowledgeBaseStack(ctx context.Context, req StackDeploymentRequest) (*StackDeploymentResult, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", req.BotID)
	
	log.Printf("🚀 Deploying Knowledge Base stack: %s", stackName)
	
	// Prepare stack parameters
	parameters, err := s.buildStackParameters(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build stack parameters: %w", err)
	}
	
	// Create CloudFormation stack
	createInput := &cloudformation.CreateStackInput{
		StackName:  aws.String(stackName),
		Parameters: parameters,
		Tags: []types.Tag{
			{
				Key:   aws.String("Environment"),
				Value: aws.String(s.envPrefix),
			},
			{
				Key:   aws.String("Project"),
				Value: aws.String("Foundry"),
			},
			{
				Key:   aws.String("BotID"),
				Value: aws.String(req.BotID),
			},
			{
				Key:   aws.String("OwnerUserID"),
				Value: aws.String(req.OwnerUserID),
			},
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("Foundry-Backend"),
			},
		},
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
			types.CapabilityCapabilityNamedIam,
		},
		EnableTerminationProtection: aws.Bool(true), // Protect against accidental deletion
	}
	
	// For this to work, we need to either:
	// 1. Pre-build and upload the CDK template to S3, then reference it
	// 2. Use CDK programmatically to synthesize the template
	// 3. Create the template as a JSON/YAML string
	
	// Option 3: Inline template (for now, we'll create a basic template)
	// In production, you'd want to use the CDK construct we created earlier
	templateBody, err := s.buildCloudFormationTemplate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build CloudFormation template: %w", err)
	}
	
	createInput.TemplateBody = aws.String(templateBody)
	
	// Create the stack
	createOutput, err := s.cfnClient.CreateStack(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudFormation stack: %w", err)
	}
	
	log.Printf("✅ Stack creation initiated: %s", *createOutput.StackId)
	
	result := &StackDeploymentResult{
		StackName:   stackName,
		StackID:     *createOutput.StackId,
		StackStatus: string(types.StackStatusCreateInProgress),
	}
	
	return result, nil
}

// GetStackStatus retrieves the current status of a CloudFormation stack
func (s *StackDeploymentService) GetStackStatus(ctx context.Context, stackName string) (*StackDeploymentResult, error) {
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	
	output, err := s.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}
	
	if len(output.Stacks) == 0 {
		return nil, fmt.Errorf("stack %s not found", stackName)
	}
	
	stack := output.Stacks[0]
	
	result := &StackDeploymentResult{
		StackName:   stackName,
		StackID:     *stack.StackId,
		StackStatus: string(stack.StackStatus),
		Outputs:     make(map[string]string),
	}
	
	// Parse stack outputs
	for _, output := range stack.Outputs {
		if output.OutputKey != nil && output.OutputValue != nil {
			result.Outputs[*output.OutputKey] = *output.OutputValue
			
			// Map specific outputs to result fields
			switch *output.OutputKey {
			case "KnowledgeBaseId":
				result.KnowledgeBaseID = output.OutputValue
			case "DataSourceId":
				result.DataSourceID = output.OutputValue
			case "DocumentBucketName":
				result.DocumentBucketName = output.OutputValue
			case "GuardrailArn":
				result.GuardrailArn = output.OutputValue
			case "GuardrailVersion":
				result.GuardrailVersion = output.OutputValue
			}
		}
	}
	
	return result, nil
}

// WaitForStackCompletion waits for a stack to reach a stable state
func (s *StackDeploymentService) WaitForStackCompletion(ctx context.Context, stackName string, timeout time.Duration) (*StackDeploymentResult, error) {
	log.Printf("⏳ Waiting for stack completion: %s", stackName)
	
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		result, err := s.GetStackStatus(ctx, stackName)
		if err != nil {
			return nil, err
		}
		
		status := types.StackStatus(result.StackStatus)
		
		// Check for completion states
		switch status {
		case types.StackStatusCreateComplete:
			log.Printf("✅ Stack creation completed: %s", stackName)
			return result, nil
		case types.StackStatusUpdateComplete:
			log.Printf("✅ Stack update completed: %s", stackName)
			return result, nil
		case types.StackStatusCreateFailed:
			return result, fmt.Errorf("stack creation failed: %s", stackName)
		case types.StackStatusUpdateFailed:
			return result, fmt.Errorf("stack update failed: %s", stackName)
		case types.StackStatusDeleteComplete:
			return result, fmt.Errorf("stack was deleted: %s", stackName)
		case types.StackStatusDeleteFailed:
			return result, fmt.Errorf("stack deletion failed: %s", stackName)
		case types.StackStatusRollbackComplete:
			return result, fmt.Errorf("stack creation rolled back: %s", stackName)
		case types.StackStatusRollbackFailed:
			return result, fmt.Errorf("stack rollback failed: %s", stackName)
		}
		
		// Wait before next check
		time.Sleep(10 * time.Second)
	}
	
	return nil, fmt.Errorf("timeout waiting for stack completion: %s", stackName)
}

// DeleteBotKnowledgeBaseStack deletes a CloudFormation stack
func (s *StackDeploymentService) DeleteBotKnowledgeBaseStack(ctx context.Context, stackName string) error {
	log.Printf("🗑️ Deleting Knowledge Base stack: %s", stackName)
	
	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}
	
	_, err := s.cfnClient.DeleteStack(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete stack %s: %w", stackName, err)
	}
	
	log.Printf("✅ Stack deletion initiated: %s", stackName)
	return nil
}

// buildStackParameters converts the deployment request to CloudFormation parameters
func (s *StackDeploymentService) buildStackParameters(req StackDeploymentRequest) ([]types.Parameter, error) {
	var parameters []types.Parameter
	
	// Core parameters
	parameters = append(parameters, []types.Parameter{
		{
			ParameterKey:   aws.String("BotId"),
			ParameterValue: aws.String(req.BotID),
		},
		{
			ParameterKey:   aws.String("OwnerUserId"),
			ParameterValue: aws.String(req.OwnerUserID),
		},
		{
			ParameterKey:   aws.String("EnvPrefix"),
			ParameterValue: aws.String(s.envPrefix),
		},
	}...)
	
	if req.Instruction != "" {
		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("Instruction"),
			ParameterValue: aws.String(req.Instruction),
		})
	}
	
	// Conditional Knowledge Base parameters
	if req.ExistingKnowledgeBaseID != nil {
		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("ExistingKnowledgeBaseId"),
			ParameterValue: req.ExistingKnowledgeBaseID,
		})
	} else if req.KnowledgeBaseCreation != nil {
		// New KB creation parameters
		kbc := req.KnowledgeBaseCreation
		
		if kbc.EmbeddingsModel != "" {
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("EmbeddingsModel"),
				ParameterValue: aws.String(kbc.EmbeddingsModel),
			})
		}
		
		if kbc.ChunkingStrategy != "" {
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("ChunkingStrategy"),
				ParameterValue: aws.String(kbc.ChunkingStrategy),
			})
		}
		
		if kbc.MaxTokens > 0 {
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("MaxTokens"),
				ParameterValue: aws.String(fmt.Sprintf("%d", kbc.MaxTokens)),
			})
		}
		
		if kbc.OverlapPercentage > 0 {
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("OverlapPercentage"),
				ParameterValue: aws.String(fmt.Sprintf("%d", kbc.OverlapPercentage)),
			})
		}
		
		// Data sources
		if len(kbc.ExistingS3Urls) > 0 {
			s3UrlsJson, _ := json.Marshal(kbc.ExistingS3Urls)
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("ExistingS3Urls"),
				ParameterValue: aws.String(string(s3UrlsJson)),
			})
		}
		
		if len(kbc.SourceUrls) > 0 {
			sourceUrlsJson, _ := json.Marshal(kbc.SourceUrls)
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("SourceUrls"),
				ParameterValue: aws.String(string(sourceUrlsJson)),
			})
		}
		
		// Guardrail configuration
		if kbc.GuardrailConfig != nil && kbc.GuardrailConfig.IsEnabled {
			guardrailJson, _ := json.Marshal(kbc.GuardrailConfig)
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("GuardrailConfig"),
				ParameterValue: aws.String(string(guardrailJson)),
			})
		}
		
		if kbc.EnableRagReplicas {
			parameters = append(parameters, types.Parameter{
				ParameterKey:   aws.String("EnableRagReplicas"),
				ParameterValue: aws.String("true"),
			})
		}
	}
	
	return parameters, nil
}

// buildCloudFormationTemplate creates a CloudFormation template for the bot Knowledge Base
// NOTE: In production, this should use the CDK construct we created earlier
// For now, we'll create a minimal template that references our CDK construct
func (s *StackDeploymentService) buildCloudFormationTemplate(req StackDeploymentRequest) (string, error) {
	// This is a placeholder - in the real implementation, you would:
	// 1. Use CDK programmatically to synthesize the BotKnowledgeBaseStack
	// 2. Or pre-build templates and store them in S3
	// 3. Or use AWS CDK CLI commands
	
	template := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              fmt.Sprintf("Knowledge Base stack for bot %s", req.BotID),
		"Parameters": map[string]interface{}{
			"BotId": map[string]interface{}{
				"Type":        "String",
				"Description": "Bot ID",
			},
			"OwnerUserId": map[string]interface{}{
				"Type":        "String",
				"Description": "Bot owner user ID",
			},
			"EnvPrefix": map[string]interface{}{
				"Type":        "String",
				"Description": "Environment prefix",
			},
		},
		"Resources": map[string]interface{}{
			// This would contain the actual resources from our CDK construct
			// For now, just a placeholder
			"PlaceholderResource": map[string]interface{}{
				"Type": "AWS::CloudFormation::WaitConditionHandle",
			},
		},
		"Outputs": map[string]interface{}{
			"BotId": map[string]interface{}{
				"Description": "Bot ID",
				"Value":       map[string]interface{}{"Ref": "BotId"},
				"Export": map[string]interface{}{
					"Name": fmt.Sprintf("BrChatKbStack%s-BotId", req.BotID),
				},
			},
		},
	}
	
	templateBytes, err := json.Marshal(template)
	if err != nil {
		return "", fmt.Errorf("failed to marshal CloudFormation template: %w", err)
	}
	
	return string(templateBytes), nil
}