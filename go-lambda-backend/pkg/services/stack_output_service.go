package services

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// StackOutputService handles fetching CloudFormation stack outputs for Knowledge Base IDs
// This mirrors the functionality of bedrock-chat's fetch_stack_output.py
type StackOutputService struct {
	cfnClient *cloudformation.Client
	region    string
}

// NewStackOutputService creates a new stack output service
func NewStackOutputService(cfg aws.Config, region string) *StackOutputService {
	return &StackOutputService{
		cfnClient: cloudformation.NewFromConfig(cfg),
		region:    region,
	}
}

// StackOutputResult contains the outputs from a bot's CloudFormation stack
type StackOutputResult struct {
	KnowledgeBaseID  string                `json:"knowledgeBaseId"`
	DataSources      []StackDataSource     `json:"dataSources"`
	GuardrailArn     string                `json:"guardrailArn,omitempty"`
	GuardrailVersion string                `json:"guardrailVersion,omitempty"`
	BotID            string                `json:"botId"`
	OwnerUserID      string                `json:"ownerUserId"`
}

// StackDataSource represents a data source from the stack outputs
type StackDataSource struct {
	KnowledgeBaseID  string `json:"knowledgeBaseId"`
	DataSourceID     string `json:"dataSourceId"`
	GuardrailArn     string `json:"guardrailArn,omitempty"`
	GuardrailVersion string `json:"guardrailVersion,omitempty"`
	BotID            string `json:"botId"`
	OwnerUserID      string `json:"ownerUserId"`
}

// FetchStackOutputs fetches the outputs from a bot's CloudFormation stack
// This is equivalent to bedrock-chat's fetch_stack_output.py handler
func (s *StackOutputService) FetchStackOutputs(ctx context.Context, botID, ownerUserID string) (*StackOutputResult, error) {
	// Stack naming rule matches bedrock-chat: BrChatKbStack{botId}
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	log.Printf("🔍 Fetching stack outputs for: %s", stackName)
	
	// Describe the CloudFormation stack
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
	
	// Parse stack outputs
	result := &StackOutputResult{
		BotID:       botID,
		OwnerUserID: ownerUserID,
		DataSources: []StackDataSource{},
	}
	
	var dataSourceIDs []string
	
	for _, output := range stack.Outputs {
		if output.OutputKey == nil || output.OutputValue == nil {
			continue
		}
		
		outputKey := *output.OutputKey
		outputValue := *output.OutputValue
		
		switch outputKey {
		case "KnowledgeBaseId":
			result.KnowledgeBaseID = outputValue
		case "GuardrailArn":
			result.GuardrailArn = outputValue
		case "GuardrailVersion":
			result.GuardrailVersion = outputValue
		default:
			// Check for DataSource outputs (they start with "DataSource")
			if len(outputKey) > 10 && outputKey[:10] == "DataSource" {
				dataSourceIDs = append(dataSourceIDs, outputValue)
			}
		}
	}
	
	// Create data source entries (following bedrock-chat pattern)
	for _, dataSourceID := range dataSourceIDs {
		dataSource := StackDataSource{
			KnowledgeBaseID:  result.KnowledgeBaseID,
			DataSourceID:     dataSourceID,
			GuardrailArn:     result.GuardrailArn,
			GuardrailVersion: result.GuardrailVersion,
			BotID:            botID,
			OwnerUserID:      ownerUserID,
		}
		result.DataSources = append(result.DataSources, dataSource)
	}
	
	log.Printf("✅ Retrieved stack outputs - KB ID: %s, DataSources: %d", 
		result.KnowledgeBaseID, len(result.DataSources))
	
	return result, nil
}

// CheckStackExists checks if a CloudFormation stack exists for the given bot
func (s *StackOutputService) CheckStackExists(ctx context.Context, botID string) (bool, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	
	_, err := s.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		// If stack doesn't exist, AWS returns a specific error
		return false, nil
	}
	
	return true, nil
}

// GetStackStatus returns the current status of a bot's CloudFormation stack
func (s *StackOutputService) GetStackStatus(ctx context.Context, botID string) (string, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	
	output, err := s.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe stack %s: %w", stackName, err)
	}
	
	if len(output.Stacks) == 0 {
		return "", fmt.Errorf("stack %s not found", stackName)
	}
	
	return string(output.Stacks[0].StackStatus), nil
}

// ListStackResources returns the resources in a bot's CloudFormation stack
func (s *StackOutputService) ListStackResources(ctx context.Context, botID string) ([]string, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	input := &cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	}
	
	output, err := s.cfnClient.ListStackResources(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list stack resources for %s: %w", stackName, err)
	}
	
	var resources []string
	for _, resource := range output.StackResourceSummaries {
		if resource.LogicalResourceId != nil {
			resources = append(resources, *resource.LogicalResourceId)
		}
	}
	
	return resources, nil
}

// GetStackEvents returns recent events for a bot's CloudFormation stack (useful for debugging)
func (s *StackOutputService) GetStackEvents(ctx context.Context, botID string, limit int32) ([]string, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	input := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	}
	
	output, err := s.cfnClient.DescribeStackEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get stack events for %s: %w", stackName, err)
	}
	
	var events []string
	count := int32(0)
	
	for _, event := range output.StackEvents {
		if count >= limit {
			break
		}
		
		if event.LogicalResourceId != nil && event.ResourceStatus != "" {
			eventStr := fmt.Sprintf("%s: %s", *event.LogicalResourceId, string(event.ResourceStatus))
			if event.ResourceStatusReason != nil {
				eventStr += fmt.Sprintf(" - %s", *event.ResourceStatusReason)
			}
			events = append(events, eventStr)
			count++
		}
	}
	
	return events, nil
}