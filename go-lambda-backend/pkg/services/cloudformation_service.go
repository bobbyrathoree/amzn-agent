package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// CloudFormationService handles Knowledge Base stack deployments using CloudFormation SDK
type CloudFormationService struct {
	cfnClient *cloudformation.Client
	region    string
}

// NewCloudFormationService creates a new CloudFormation service
func NewCloudFormationService(cfg aws.Config, region string) *CloudFormationService {
	return &CloudFormationService{
		cfnClient: cloudformation.NewFromConfig(cfg),
		region:    region,
	}
}

// KnowledgeBaseStackRequest contains parameters for Knowledge Base stack creation
type KnowledgeBaseStackRequest struct {
	BotID                    string
	OwnerUserID              string
	Instruction              string
	ExistingKnowledgeBaseID  *string
	KnowledgeBaseCreation    *models.KnowledgeBaseCreationConfig
}

// KnowledgeBaseStackResult contains the result of a CloudFormation deployment
type KnowledgeBaseStackResult struct {
	StackName            string                 `json:"stackName"`
	StackId              string                 `json:"stackId"`
	Status               string                 `json:"status"`
	KnowledgeBaseId      string                 `json:"knowledgeBaseId,omitempty"`
	DataSourceId         string                 `json:"dataSourceId,omitempty"`
	S3BucketName         string                 `json:"s3BucketName,omitempty"`
	Outputs              map[string]string      `json:"outputs,omitempty"`
	Error                string                 `json:"error,omitempty"`
}

// CreateKnowledgeBaseStack creates a new Knowledge Base stack using CloudFormation
func (s *CloudFormationService) CreateKnowledgeBaseStack(ctx context.Context, req KnowledgeBaseStackRequest) (*KnowledgeBaseStackResult, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", req.BotID)
	
	// Generate CloudFormation template
	template, err := s.generateKnowledgeBaseTemplate(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate template: %w", err)
	}

	// Create stack parameters
	parameters := []types.Parameter{
		{
			ParameterKey:   aws.String("BotId"),
			ParameterValue: aws.String(req.BotID),
		},
		{
			ParameterKey:   aws.String("OwnerUserId"),
			ParameterValue: aws.String(req.OwnerUserID),
		},
	}

	if req.Instruction != "" {
		parameters = append(parameters, types.Parameter{
			ParameterKey:   aws.String("Instruction"),
			ParameterValue: aws.String(req.Instruction),
		})
	}

	// Create the stack
	createInput := &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(template),
		Parameters:   parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityIam,
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("BotId"),
				Value: aws.String(req.BotID),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("KnowledgeBase"),
			},
		},
	}

	output, err := s.cfnClient.CreateStack(ctx, createInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create stack: %w", err)
	}

	log.Printf("✅ Started CloudFormation stack creation: %s (ID: %s)", stackName, *output.StackId)

	return &KnowledgeBaseStackResult{
		StackName: stackName,
		StackId:   *output.StackId,
		Status:    "CREATE_IN_PROGRESS",
	}, nil
}

// GetStackStatus retrieves the current status of a CloudFormation stack
func (s *CloudFormationService) GetStackStatus(ctx context.Context, stackName string) (*KnowledgeBaseStackResult, error) {
	input := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}

	output, err := s.cfnClient.DescribeStacks(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe stack: %w", err)
	}

	if len(output.Stacks) == 0 {
		return nil, fmt.Errorf("stack not found: %s", stackName)
	}

	stack := output.Stacks[0]
	result := &KnowledgeBaseStackResult{
		StackName: stackName,
		StackId:   *stack.StackId,
		Status:    string(stack.StackStatus),
		Outputs:   make(map[string]string),
	}

	// Extract outputs
	for _, output := range stack.Outputs {
		if output.OutputKey != nil && output.OutputValue != nil {
			result.Outputs[*output.OutputKey] = *output.OutputValue
		}
	}

	// Extract specific Knowledge Base information from outputs
	if kbId, exists := result.Outputs["KnowledgeBaseId"]; exists {
		result.KnowledgeBaseId = kbId
	}
	if dsId, exists := result.Outputs["DataSourceId"]; exists {
		result.DataSourceId = dsId
	}
	if s3Bucket, exists := result.Outputs["S3BucketName"]; exists {
		result.S3BucketName = s3Bucket
	}

	return result, nil
}

// DeleteStack deletes a CloudFormation stack
func (s *CloudFormationService) DeleteStack(ctx context.Context, stackName string) error {
	input := &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	}

	_, err := s.cfnClient.DeleteStack(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete stack: %w", err)
	}

	log.Printf("✅ Started CloudFormation stack deletion: %s", stackName)
	return nil
}

// generateKnowledgeBaseTemplate generates a CloudFormation template for Knowledge Base resources
func (s *CloudFormationService) generateKnowledgeBaseTemplate(req KnowledgeBaseStackRequest) (string, error) {
	template := map[string]interface{}{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Description":              fmt.Sprintf("Knowledge Base stack for Bot %s", req.BotID),
		"Parameters": map[string]interface{}{
			"BotId": map[string]interface{}{
				"Type":        "String",
				"Description": "Bot ID for the Knowledge Base",
			},
			"OwnerUserId": map[string]interface{}{
				"Type":        "String",
				"Description": "User ID of the bot owner",
			},
			"Instruction": map[string]interface{}{
				"Type":        "String",
				"Description": "Bot instruction/prompt",
				"Default":     "",
			},
		},
		"Resources": map[string]interface{}{
			// S3 Bucket for Knowledge Base documents
			"KnowledgeBaseS3Bucket": map[string]interface{}{
				"Type": "AWS::S3::Bucket",
				"Properties": map[string]interface{}{
					"BucketName": map[string]interface{}{
						"Fn::Sub": "${BotId}-knowledge-base-${AWS::AccountId}",
					},
					"BucketEncryption": map[string]interface{}{
						"ServerSideEncryptionConfiguration": []map[string]interface{}{
							{
								"ServerSideEncryptionByDefault": map[string]interface{}{
									"SSEAlgorithm": "AES256",
								},
							},
						},
					},
					"PublicAccessBlockConfiguration": map[string]interface{}{
						"BlockPublicAcls":       true,
						"BlockPublicPolicy":     true,
						"IgnorePublicAcls":      true,
						"RestrictPublicBuckets": true,
					},
				},
			},
			
			// OpenSearch Serverless Collection for vector storage
			"OpenSearchCollection": map[string]interface{}{
				"Type": "AWS::OpenSearchServerless::Collection",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "kb-${BotId}",
					},
					"Type": "VECTORSEARCH",
					"Description": map[string]interface{}{
						"Fn::Sub": "Vector search collection for bot ${BotId} knowledge base",
					},
				},
				"DependsOn": []string{"OpenSearchNetworkPolicy", "OpenSearchEncryptionPolicy", "OpenSearchAccessPolicy"},
			},
			
			// OpenSearch Serverless Network Policy
			"OpenSearchNetworkPolicy": map[string]interface{}{
				"Type": "AWS::OpenSearchServerless::SecurityPolicy",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "kb-${BotId}-network-policy",
					},
					"Type": "network",
					"Policy": map[string]interface{}{
						"Fn::Sub": `[{"Rules":[{"ResourceType":"collection","Resource":["collection/kb-${BotId}"]}, {"ResourceType":"dashboard","Resource":["collection/kb-${BotId}"]}],"AllowFromPublic":true}]`,
					},
				},
			},
			
			// OpenSearch Serverless Encryption Policy
			"OpenSearchEncryptionPolicy": map[string]interface{}{
				"Type": "AWS::OpenSearchServerless::SecurityPolicy",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "kb-${BotId}-encryption-policy",
					},
					"Type": "encryption",
					"Policy": map[string]interface{}{
						"Fn::Sub": `{"Rules":[{"ResourceType":"collection","Resource":["collection/kb-${BotId}"]}],"AWSOwnedKey":true}`,
					},
				},
			},
			
			// OpenSearch Serverless Access Policy
			"OpenSearchAccessPolicy": map[string]interface{}{
				"Type": "AWS::OpenSearchServerless::AccessPolicy",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "kb-${BotId}-access-policy",
					},
					"Type": "data",
					"Policy": map[string]interface{}{
						"Fn::Sub": `[{"Rules":[{"Resource":["collection/kb-${BotId}"],"Permission":["aoss:CreateCollectionItems","aoss:DeleteCollectionItems","aoss:UpdateCollectionItems","aoss:DescribeCollectionItems"],"ResourceType":"collection"},{"Resource":["index/kb-${BotId}/*"],"Permission":["aoss:CreateIndex","aoss:DeleteIndex","aoss:UpdateIndex","aoss:DescribeIndex","aoss:ReadDocument","aoss:WriteDocument"],"ResourceType":"index"}],"Principal":["${BedrockKnowledgeBaseRole.Arn}"]}]`,
					},
				},
			},
			
			// IAM Role for Bedrock Knowledge Base
			"BedrockKnowledgeBaseRole": map[string]interface{}{
				"Type": "AWS::IAM::Role",
				"Properties": map[string]interface{}{
					"RoleName": map[string]interface{}{
						"Fn::Sub": "BedrockKnowledgeBaseRole-${BotId}",
					},
					"AssumeRolePolicyDocument": map[string]interface{}{
						"Version": "2012-10-17",
						"Statement": []map[string]interface{}{
							{
								"Effect": "Allow",
								"Principal": map[string]interface{}{
									"Service": "bedrock.amazonaws.com",
								},
								"Action": "sts:AssumeRole",
							},
						},
					},
					"Policies": []map[string]interface{}{
						{
							"PolicyName": "BedrockKnowledgeBasePolicy",
							"PolicyDocument": map[string]interface{}{
								"Version": "2012-10-17",
								"Statement": []map[string]interface{}{
									{
										"Effect": "Allow",
										"Action": []string{
											"s3:GetObject",
											"s3:ListBucket",
											"s3:GetBucketLocation",
											"s3:GetBucketVersioning",
										},
										"Resource": []interface{}{
											map[string]interface{}{"Fn::GetAtt": []string{"KnowledgeBaseS3Bucket", "Arn"}},
											map[string]interface{}{"Fn::Sub": "${KnowledgeBaseS3Bucket}/*"},
										},
									},
									{
										"Effect": "Allow",
										"Action": []string{
											"aoss:APIAccessAll",
										},
										"Resource": []interface{}{
											map[string]interface{}{"Fn::GetAtt": []string{"OpenSearchCollection", "Arn"}},
										},
									},
									{
										"Effect": "Allow",
										"Action": []string{
											"bedrock:InvokeModel",
										},
										"Resource": "arn:aws:bedrock:*::foundation-model/amazon.titan-embed-text-v2:0",
									},
								},
							},
						},
					},
				},
			},
			
			// Bedrock Knowledge Base
			"BedrockKnowledgeBase": map[string]interface{}{
				"Type": "AWS::Bedrock::KnowledgeBase",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "${BotId}-knowledge-base",
					},
					"Description": map[string]interface{}{
						"Fn::Sub": "Knowledge base for bot ${BotId}",
					},
					"RoleArn": map[string]interface{}{
						"Fn::GetAtt": []string{"BedrockKnowledgeBaseRole", "Arn"},
					},
					"KnowledgeBaseConfiguration": map[string]interface{}{
						"Type": "VECTOR",
						"VectorKnowledgeBaseConfiguration": map[string]interface{}{
							"EmbeddingModelArn": fmt.Sprintf("arn:aws:bedrock:%s::foundation-model/amazon.titan-embed-text-v2:0", s.region),
						},
					},
					"StorageConfiguration": map[string]interface{}{
						"Type": "OPENSEARCH_SERVERLESS",
						"OpensearchServerlessConfiguration": map[string]interface{}{
							"CollectionArn": map[string]interface{}{
								"Fn::GetAtt": []string{"OpenSearchCollection", "Arn"},
							},
							"VectorIndexName": "bedrock-knowledge-base-default-index",
							"FieldMapping": map[string]interface{}{
								"VectorField":   "bedrock-knowledge-base-default-vector",
								"TextField":     "AMAZON_BEDROCK_TEXT_CHUNK",
								"MetadataField": "AMAZON_BEDROCK_METADATA",
							},
						},
					},
				},
				"DependsOn": []string{"OpenSearchCollection", "BedrockKnowledgeBaseRole"},
			},
			
			// S3 Data Source for the Knowledge Base
			"BedrockDataSource": map[string]interface{}{
				"Type": "AWS::Bedrock::DataSource",
				"Properties": map[string]interface{}{
					"Name": map[string]interface{}{
						"Fn::Sub": "${BotId}-s3-data-source",
					},
					"Description": map[string]interface{}{
						"Fn::Sub": "S3 data source for bot ${BotId} knowledge base",
					},
					"KnowledgeBaseId": map[string]interface{}{
						"Ref": "BedrockKnowledgeBase",
					},
					"DataSourceConfiguration": map[string]interface{}{
						"Type": "S3",
						"S3Configuration": map[string]interface{}{
							"BucketArn": map[string]interface{}{
								"Fn::GetAtt": []string{"KnowledgeBaseS3Bucket", "Arn"},
							},
						},
					},
					"VectorIngestionConfiguration": map[string]interface{}{
						"ChunkingConfiguration": map[string]interface{}{
							"ChunkingStrategy": "FIXED_SIZE",
							"FixedSizeChunkingConfiguration": map[string]interface{}{
								"MaxTokens":        300,
								"OverlapPercentage": 20,
							},
						},
					},
				},
				"DependsOn": []string{"BedrockKnowledgeBase"},
			},
		},
		"Outputs": map[string]interface{}{
			"S3BucketName": map[string]interface{}{
				"Description": "S3 Bucket for Knowledge Base documents",
				"Value":       map[string]interface{}{"Ref": "KnowledgeBaseS3Bucket"},
			},
			"KnowledgeBaseRoleArn": map[string]interface{}{
				"Description": "IAM Role ARN for Bedrock Knowledge Base",
				"Value":       map[string]interface{}{"Fn::GetAtt": []string{"BedrockKnowledgeBaseRole", "Arn"}},
			},
			"KnowledgeBaseId": map[string]interface{}{
				"Description": "Bedrock Knowledge Base ID",
				"Value":       map[string]interface{}{"Ref": "BedrockKnowledgeBase"},
			},
			"DataSourceId": map[string]interface{}{
				"Description": "Bedrock Data Source ID",
				"Value":       map[string]interface{}{"Ref": "BedrockDataSource"},
			},
			"OpenSearchCollectionArn": map[string]interface{}{
				"Description": "OpenSearch Serverless Collection ARN",
				"Value":       map[string]interface{}{"Fn::GetAtt": []string{"OpenSearchCollection", "Arn"}},
			},
			"OpenSearchCollectionId": map[string]interface{}{
				"Description": "OpenSearch Serverless Collection ID",
				"Value":       map[string]interface{}{"Fn::GetAtt": []string{"OpenSearchCollection", "Id"}},
			},
		},
	}

	templateJSON, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return string(templateJSON), nil
}