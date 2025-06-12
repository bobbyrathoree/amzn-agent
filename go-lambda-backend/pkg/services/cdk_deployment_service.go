package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// CDKDeploymentService handles bot Knowledge Base deployments using CDK CLI
type CDKDeploymentService struct {
	cdkPath           string // Path to CDK CLI binary
	infrastructurePath string // Path to infrastructure directory
	envPrefix         string
	region            string
}

// NewCDKDeploymentService creates a new CDK deployment service
func NewCDKDeploymentService(cdkPath, infrastructurePath, envPrefix, region string) *CDKDeploymentService {
	// Default CDK path if not provided
	if cdkPath == "" {
		cdkPath = "cdk" // Assumes CDK is in PATH
	}

	return &CDKDeploymentService{
		cdkPath:           cdkPath,
		infrastructurePath: infrastructurePath,
		envPrefix:         envPrefix,
		region:            region,
	}
}

// CDKDeploymentRequest contains parameters for CDK bot stack deployment
type CDKDeploymentRequest struct {
	BotID                    string
	OwnerUserID              string
	Instruction              string
	ExistingKnowledgeBaseID  *string
	KnowledgeBaseCreation    *models.KnowledgeBaseCreationConfig
}

// CDKDeploymentResult contains the result of a CDK deployment
type CDKDeploymentResult struct {
	StackName            string                 `json:"stackName"`
	StackStatus          string                 `json:"stackStatus"`
	KnowledgeBaseID      *string                `json:"knowledgeBaseId,omitempty"`
	DataSourceID         *string                `json:"dataSourceId,omitempty"`
	DocumentBucketName   *string                `json:"documentBucketName,omitempty"`
	GuardrailArn         *string                `json:"guardrailArn,omitempty"`
	GuardrailVersion     *string                `json:"guardrailVersion,omitempty"`
	DeploymentOutput     string                 `json:"deploymentOutput"`
	Outputs              map[string]string      `json:"outputs,omitempty"`
}

// DeployBotKnowledgeBaseStack deploys a bot Knowledge Base stack using CDK CLI
func (s *CDKDeploymentService) DeployBotKnowledgeBaseStack(ctx context.Context, req CDKDeploymentRequest) (*CDKDeploymentResult, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", req.BotID)
	
	log.Printf("🚀 Deploying bot Knowledge Base stack via CDK CLI: %s", stackName)
	
	// Create temporary CDK app file for this specific bot
	appFilePath, err := s.createBotCDKApp(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create CDK app file: %w", err)
	}
	defer os.Remove(appFilePath) // Cleanup
	
	// Build CDK deploy command
	cmd := exec.CommandContext(ctx, s.cdkPath, "deploy", stackName, 
		"--require-approval", "never",
		"--app", fmt.Sprintf("npx ts-node %s", appFilePath),
		"--outputs-file", filepath.Join(s.infrastructurePath, fmt.Sprintf("outputs-%s.json", req.BotID)),
		"--region", s.region,
	)
	
	// Set working directory to infrastructure path
	cmd.Dir = s.infrastructurePath
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CDK_DEFAULT_REGION=%s", s.region),
		fmt.Sprintf("ENV_PREFIX=%s", s.envPrefix),
	)
	
	log.Printf("📄 Running CDK command: %s", strings.Join(cmd.Args, " "))
	
	// Execute CDK deploy
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ CDK deployment failed: %s", string(output))
		return nil, fmt.Errorf("CDK deployment failed: %w\nOutput: %s", err, string(output))
	}
	
	log.Printf("✅ CDK deployment completed for stack: %s", stackName)
	
	// Parse CDK outputs
	result := &CDKDeploymentResult{
		StackName:        stackName,
		StackStatus:      "CREATE_COMPLETE", // CDK only returns on success
		DeploymentOutput: string(output),
		Outputs:          make(map[string]string),
	}
	
	// Read outputs file if it exists
	outputsFile := filepath.Join(s.infrastructurePath, fmt.Sprintf("outputs-%s.json", req.BotID))
	if outputs, err := s.parseOutputsFile(outputsFile); err == nil {
		result.Outputs = outputs
		
		// Map specific outputs to result fields
		if kbId, ok := outputs["KnowledgeBaseId"]; ok {
			result.KnowledgeBaseID = &kbId
		}
		if dsId, ok := outputs["DataSourceId"]; ok {
			result.DataSourceID = &dsId
		}
		if bucketName, ok := outputs["DocumentBucketName"]; ok {
			result.DocumentBucketName = &bucketName
		}
		if guardrailArn, ok := outputs["GuardrailArn"]; ok {
			result.GuardrailArn = &guardrailArn
		}
		if guardrailVersion, ok := outputs["GuardrailVersion"]; ok {
			result.GuardrailVersion = &guardrailVersion
		}
		
		// Cleanup outputs file
		os.Remove(outputsFile)
	}
	
	return result, nil
}

// DestroyBotKnowledgeBaseStack destroys a bot Knowledge Base stack using CDK CLI
func (s *CDKDeploymentService) DestroyBotKnowledgeBaseStack(ctx context.Context, botID string) error {
	stackName := fmt.Sprintf("BrChatKbStack%s", botID)
	
	log.Printf("🗑️ Destroying bot Knowledge Base stack via CDK CLI: %s", stackName)
	
	// Build CDK destroy command
	cmd := exec.CommandContext(ctx, s.cdkPath, "destroy", stackName,
		"--force", // Skip confirmation prompts
		"--region", s.region,
	)
	
	// Set working directory to infrastructure path
	cmd.Dir = s.infrastructurePath
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CDK_DEFAULT_REGION=%s", s.region),
		fmt.Sprintf("ENV_PREFIX=%s", s.envPrefix),
	)
	
	log.Printf("📄 Running CDK destroy command: %s", strings.Join(cmd.Args, " "))
	
	// Execute CDK destroy
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("❌ CDK destroy failed: %s", string(output))
		return fmt.Errorf("CDK destroy failed: %w\nOutput: %s", err, string(output))
	}
	
	log.Printf("✅ CDK destroy completed for stack: %s", stackName)
	return nil
}

// SynthesizeBotKnowledgeBaseStack synthesizes the CDK template without deploying
func (s *CDKDeploymentService) SynthesizeBotKnowledgeBaseStack(ctx context.Context, req CDKDeploymentRequest) (string, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", req.BotID)
	
	log.Printf("🔍 Synthesizing bot Knowledge Base stack: %s", stackName)
	
	// Create temporary CDK app file
	appFilePath, err := s.createBotCDKApp(req)
	if err != nil {
		return "", fmt.Errorf("failed to create CDK app file: %w", err)
	}
	defer os.Remove(appFilePath)
	
	// Build CDK synth command
	cmd := exec.CommandContext(ctx, s.cdkPath, "synth", stackName,
		"--app", fmt.Sprintf("npx ts-node %s", appFilePath),
		"--region", s.region,
	)
	
	cmd.Dir = s.infrastructurePath
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CDK_DEFAULT_REGION=%s", s.region),
		fmt.Sprintf("ENV_PREFIX=%s", s.envPrefix),
	)
	
	// Execute CDK synth
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("CDK synthesis failed: %w\nOutput: %s", err, string(output))
	}
	
	return string(output), nil
}

// createBotCDKApp creates a temporary CDK app file for the specific bot deployment
func (s *CDKDeploymentService) createBotCDKApp(req CDKDeploymentRequest) (string, error) {
	stackName := fmt.Sprintf("BrChatKbStack%s", req.BotID)
	
	// Convert request to CDK construct parameters
	constructParams := s.buildCDKConstructParams(req)
	paramsJson, _ := json.MarshalIndent(constructParams, "  ", "  ")
	
	// Generate CDK app TypeScript code
	appCode := fmt.Sprintf(`#!/usr/bin/env npx ts-node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { BotKnowledgeBaseStack } from './lib/constructs/bot-knowledge-base-stack';

const app = new cdk.App();

// Bot-specific parameters
const params = %s;

new BotKnowledgeBaseStack(app, '%s', {
  env: {
    region: process.env.CDK_DEFAULT_REGION || '%s',
  },
  ...params
});
`, string(paramsJson), stackName, s.region)
	
	// Write to temporary file
	tempFile := filepath.Join(s.infrastructurePath, fmt.Sprintf("bot-app-%s.ts", req.BotID))
	if err := os.WriteFile(tempFile, []byte(appCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write CDK app file: %w", err)
	}
	
	return tempFile, nil
}

// buildCDKConstructParams converts the deployment request to CDK construct parameters
func (s *CDKDeploymentService) buildCDKConstructParams(req CDKDeploymentRequest) map[string]interface{} {
	params := map[string]interface{}{
		"botId":      req.BotID,
		"ownerUserId": req.OwnerUserID,
		"envPrefix":  s.envPrefix,
	}
	
	if req.Instruction != "" {
		params["instruction"] = req.Instruction
	}
	
	// Conditional Knowledge Base parameters
	if req.ExistingKnowledgeBaseID != nil {
		params["existingKnowledgeBaseId"] = *req.ExistingKnowledgeBaseID
	} else if req.KnowledgeBaseCreation != nil {
		kbc := req.KnowledgeBaseCreation
		
		if kbc.EmbeddingsModel != "" {
			params["embeddingsModel"] = s.mapEmbeddingsModel(kbc.EmbeddingsModel)
		}
		
		if kbc.ChunkingStrategy != "" {
			params["chunkingStrategy"] = kbc.ChunkingStrategy
		}
		
		if kbc.MaxTokens > 0 {
			params["maxTokens"] = kbc.MaxTokens
		}
		
		if kbc.OverlapPercentage > 0 {
			params["overlapPercentage"] = kbc.OverlapPercentage
		}
		
		if len(kbc.ExistingS3Urls) > 0 {
			params["existingS3Urls"] = kbc.ExistingS3Urls
		}
		
		if len(kbc.SourceUrls) > 0 {
			params["sourceUrls"] = kbc.SourceUrls
		}
		
		if kbc.GuardrailConfig != nil && kbc.GuardrailConfig.IsEnabled {
			params["guardrailConfig"] = map[string]interface{}{
				"isEnabled":           kbc.GuardrailConfig.IsEnabled,
				"hateThreshold":       kbc.GuardrailConfig.HateThreshold,
				"insultsThreshold":   kbc.GuardrailConfig.InsultsThreshold,
				"sexualThreshold":    kbc.GuardrailConfig.SexualThreshold,
				"violenceThreshold":  kbc.GuardrailConfig.ViolenceThreshold,
				"misconductThreshold": kbc.GuardrailConfig.MisconductThreshold,
				"groundingThreshold": kbc.GuardrailConfig.GroundingThreshold,
				"relevanceThreshold": kbc.GuardrailConfig.RelevanceThreshold,
			}
		}
		
		if kbc.EnableRagReplicas {
			params["enableRagReplicas"] = true
		}
	}
	
	return params
}

// mapEmbeddingsModel maps Go model names to CDK BedrockFoundationModel
func (s *CDKDeploymentService) mapEmbeddingsModel(model string) string {
	switch model {
	case "amazon.titan-embed-text-v1":
		return "BedrockFoundationModel.AMAZON_TITAN_EMBED_TEXT_V1"
	case "amazon.titan-embed-text-v2:0":
		return "BedrockFoundationModel.AMAZON_TITAN_EMBED_TEXT_V2"
	case "cohere.embed-english-v3":
		return "BedrockFoundationModel.COHERE_EMBED_ENGLISH_V3"
	case "cohere.embed-multilingual-v3":
		return "BedrockFoundationModel.COHERE_EMBED_MULTILINGUAL_V3"
	default:
		return "BedrockFoundationModel.AMAZON_TITAN_EMBED_TEXT_V1"
	}
}

// parseOutputsFile parses the CDK outputs JSON file
func (s *CDKDeploymentService) parseOutputsFile(filePath string) (map[string]string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var outputs map[string]map[string]string
	if err := json.Unmarshal(data, &outputs); err != nil {
		return nil, err
	}
	
	// CDK outputs file format: { "StackName": { "OutputKey": "OutputValue" } }
	for _, stackOutputs := range outputs {
		return stackOutputs, nil // Return first (and only) stack's outputs
	}
	
	return make(map[string]string), nil
}

// CheckCDKAvailability checks if CDK CLI is available and properly configured
func (s *CDKDeploymentService) CheckCDKAvailability() error {
	log.Printf("🔍 Checking CDK CLI availability...")
	
	// Check CDK version
	cmd := exec.Command(s.cdkPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("CDK CLI not found or not executable: %w", err)
	}
	
	log.Printf("✅ CDK CLI available: %s", strings.TrimSpace(string(output)))
	
	// Check if we're in a CDK project
	if _, err := os.Stat(filepath.Join(s.infrastructurePath, "cdk.json")); os.IsNotExist(err) {
		return fmt.Errorf("cdk.json not found in %s - not a CDK project", s.infrastructurePath)
	}
	
	// Check if our construct exists
	constructPath := filepath.Join(s.infrastructurePath, "lib", "constructs", "bot-knowledge-base-stack.ts")
	if _, err := os.Stat(constructPath); os.IsNotExist(err) {
		return fmt.Errorf("BotKnowledgeBaseStack construct not found at %s", constructPath)
	}
	
	log.Printf("✅ CDK project structure verified")
	return nil
}