import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as opensearch from 'aws-cdk-lib/aws-opensearchserverless';
import * as cr from 'aws-cdk-lib/custom-resources';
import { Construct } from 'constructs';

// Import Bedrock constructs from generative-ai-cdk-constructs
// Note: You'll need to install this package: npm install @cdklabs/generative-ai-cdk-constructs
import {
  bedrock,
  opensearch_vectorindex,
  opensearchserverless,
} from '@cdklabs/generative-ai-cdk-constructs';

export interface BotKnowledgeBaseStackProps extends cdk.StackProps {
  // Core bot properties
  botId: string;
  ownerUserId: string;
  envPrefix: string;
  instruction?: string;

  // Conditional KB creation - either provide existing KB ID OR creation parameters
  existingKnowledgeBaseId?: string;

  // New KB creation parameters (used only if existingKnowledgeBaseId is not provided)
  embeddingsModel?: bedrock.BedrockFoundationModel;
  chunkingStrategy?: 'FIXED_SIZE' | 'NONE' | 'HIERARCHICAL' | 'SEMANTIC';
  maxTokens?: number;
  overlapPercentage?: number;
  
  // Data sources for new KB
  existingS3Urls?: string[];  // S3 URLs to index
  sourceUrls?: string[];      // Web URLs to crawl
  
  // Guardrail configuration
  guardrailConfig?: {
    isEnabled: boolean;
    hateThreshold?: number;
    insultsThreshold?: number;
    sexualThreshold?: number;
    violenceThreshold?: number;
    misconductThreshold?: number;
    groundingThreshold?: number;
    relevanceThreshold?: number;
  };

  // Infrastructure options
  enableRagReplicas?: boolean;
}

export class BotKnowledgeBaseStack extends cdk.Stack {
  public readonly knowledgeBase: bedrock.VectorKnowledgeBase;
  public readonly documentBucket?: s3.Bucket;
  public readonly dataSource?: bedrock.S3DataSource;
  public readonly guardrail?: bedrock.Guardrail;
  public readonly knowledgeBaseId: string;
  
  constructor(scope: Construct, id: string, props: BotKnowledgeBaseStackProps) {
    super(scope, id, props);
    
    const stackName = `BrChatKbStack${props.botId}`;
    
    // CONDITIONAL LOGIC: Existing KB vs New KB Creation
    if (props.existingKnowledgeBaseId) {
      // ========================================
      // PATH 1: USE EXISTING KNOWLEDGE BASE
      // ========================================
      console.log(`🔗 Using existing Knowledge Base: ${props.existingKnowledgeBaseId}`);
      
      // Get existing KB details using AWS Custom Resource
      const getKnowledgeBase = new cr.AwsCustomResource(this, 'GetKnowledgeBase', {
        onCreate: {
          service: 'BedrockAgent',
          action: 'getKnowledgeBase',
          parameters: {
            knowledgeBaseId: props.existingKnowledgeBaseId,
          },
          physicalResourceId: cr.PhysicalResourceId.of(props.existingKnowledgeBaseId),
        },
        policy: cr.AwsCustomResourcePolicy.fromSdkCalls({
          resources: [`arn:aws:bedrock:${this.region}:${this.account}:knowledge-base/${props.existingKnowledgeBaseId}`],
        }),
      });

      // Reference existing KB
      this.knowledgeBase = bedrock.VectorKnowledgeBase.fromKnowledgeBaseAttributes(this, 'ExistingKnowledgeBase', {
        knowledgeBaseId: props.existingKnowledgeBaseId,
        executionRoleArn: getKnowledgeBase.getResponseField('knowledgeBase.roleArn'),
        vectorStoreType: bedrock.VectorStoreType.OPENSEARCH_SERVERLESS, // Assume OpenSearch Serverless for existing KB
      }) as bedrock.VectorKnowledgeBase;

      this.knowledgeBaseId = props.existingKnowledgeBaseId;
      
    } else {
      // ========================================
      // PATH 2: CREATE NEW KNOWLEDGE BASE INFRASTRUCTURE
      // ========================================
      console.log(`🏗️ Creating new Knowledge Base infrastructure for bot ${props.botId}`);
      
      // Create S3 bucket for documents
      this.documentBucket = new s3.Bucket(this, 'DocumentBucket', {
        bucketName: `${props.envPrefix.toLowerCase()}-bot-${props.botId}-docs`,
        removalPolicy: cdk.RemovalPolicy.RETAIN, // Keep documents even if stack is deleted
        versioned: true,
        encryption: s3.BucketEncryption.S3_MANAGED,
        publicReadAccess: false,
        blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      });

      // Create OpenSearch Serverless collection for vector storage
      const collectionName = `bot-${props.botId}`.toLowerCase().substring(0, 32);
      
      const vectorCollection = new opensearchserverless.VectorCollection(this, 'VectorCollection', {
        collectionName,
        standbyReplicas: props.enableRagReplicas 
          ? opensearchserverless.VectorCollectionStandbyReplicas.ENABLED
          : opensearchserverless.VectorCollectionStandbyReplicas.DISABLED,
      });

      // Create vector index
      const embeddingsModel = props.embeddingsModel || bedrock.BedrockFoundationModel.TITAN_EMBED_TEXT_V2_1024;
      
      const vectorIndex = new opensearch_vectorindex.VectorIndex(this, 'VectorIndex', {
        collection: vectorCollection,
        indexName: 'bedrock-knowledge-base-default-index',
        vectorField: 'bedrock-knowledge-base-default-vector',
        vectorDimensions: embeddingsModel.vectorDimensions!,
        precision: 'float32',
        distanceType: 'cosine',
        mappings: [
          {
            mappingField: 'AMAZON_BEDROCK_METADATA',
            dataType: 'text',
            filterable: true,
          },
          {
            mappingField: 'AMAZON_BEDROCK_TEXT_CHUNK',
            dataType: 'text',
            filterable: false,
          },
        ],
      });

      // Create Knowledge Base with new infrastructure
      this.knowledgeBase = new bedrock.VectorKnowledgeBase(this, 'KnowledgeBase', {
        name: `${props.envPrefix}-bot-${props.botId}-kb`,
        description: props.instruction || `Knowledge base for bot ${props.botId}`,
        embeddingsModel,
        vectorStore: vectorCollection,
        vectorIndex: vectorIndex,
        instruction: props.instruction,
      });

      this.knowledgeBaseId = this.knowledgeBase.knowledgeBaseId;

      // Create S3 data source if we have a document bucket
      if (this.documentBucket) {
        this.dataSource = new bedrock.S3DataSource(this, 'S3DataSource', {
          knowledgeBase: this.knowledgeBase,
          dataSourceName: `${props.envPrefix}-bot-${props.botId}-s3-source`,
          bucket: this.documentBucket,
          inclusionPrefixes: ['documents/'], // Only index files in documents/ folder
          chunkingStrategy: this.getChunkingStrategy(props),
        });
      }

      // Add web crawler data sources if URLs provided
      if (props.sourceUrls && props.sourceUrls.length > 0) {
        new bedrock.WebCrawlerDataSource(this, 'WebCrawlerDataSource', {
          knowledgeBase: this.knowledgeBase,
          dataSourceName: `${props.envPrefix}-bot-${props.botId}-web-source`,
          sourceUrls: props.sourceUrls,
          chunkingStrategy: this.getChunkingStrategy(props),
        });
      }
    }

    // Create guardrail if enabled (applies to both existing and new KB)
    if (props.guardrailConfig?.isEnabled) {
      this.guardrail = new bedrock.Guardrail(this, 'Guardrail', {
        name: `${props.envPrefix}-bot-${props.botId}-guardrail`,
        description: `Content guardrail for bot ${props.botId}`,
        blockedInputMessaging: 'This content violates our content policy.',
        blockedOutputsMessaging: 'This response has been blocked due to content policy.',
        contentFilters: this.createContentFilters(props.guardrailConfig),
      });
    }
    
    // Stack outputs - critical for fetching KB details later
    new cdk.CfnOutput(this, 'KnowledgeBaseId', {
      value: this.knowledgeBaseId,
      description: 'Bedrock Knowledge Base ID',
      exportName: `${stackName}-KnowledgeBaseId`,
    });
    
    if (this.dataSource) {
      new cdk.CfnOutput(this, 'DataSourceId', {
        value: this.dataSource.dataSourceId,
        description: 'Bedrock Data Source ID', 
        exportName: `${stackName}-DataSourceId`,
      });
    }
    
    if (this.documentBucket) {
      new cdk.CfnOutput(this, 'DocumentBucketName', {
        value: this.documentBucket.bucketName,
        description: 'S3 Document Bucket Name',
        exportName: `${stackName}-DocumentBucketName`,
      });
    }
    
    if (this.guardrail) {
      new cdk.CfnOutput(this, 'GuardrailArn', {
        value: this.guardrail.guardrailArn,
        description: 'Bedrock Guardrail ARN',
        exportName: `${stackName}-GuardrailArn`,
      });
      
      new cdk.CfnOutput(this, 'GuardrailVersion', {
        value: this.guardrail.guardrailVersion,
        description: 'Bedrock Guardrail Version',
        exportName: `${stackName}-GuardrailVersion`,
      });
    }

    // Output the bot configuration for reference
    new cdk.CfnOutput(this, 'BotId', {
      value: props.botId,
      description: 'Bot ID',
      exportName: `${stackName}-BotId`,
    });

    new cdk.CfnOutput(this, 'OwnerUserId', {
      value: props.ownerUserId,
      description: 'Bot Owner User ID',
      exportName: `${stackName}-OwnerUserId`,
    });
  }
  
  private getChunkingStrategy(props: BotKnowledgeBaseStackProps): bedrock.ChunkingStrategy {
    switch (props.chunkingStrategy) {
      case 'FIXED_SIZE':
        return bedrock.ChunkingStrategy.FIXED_SIZE;
      case 'HIERARCHICAL':
        return bedrock.ChunkingStrategy.HIERARCHICAL_TITAN;
      case 'SEMANTIC':
        return bedrock.ChunkingStrategy.SEMANTIC;
      case 'NONE':
        return bedrock.ChunkingStrategy.NONE;
      default:
        return bedrock.ChunkingStrategy.FIXED_SIZE;
    }
  }

  private createContentFilters(config: NonNullable<BotKnowledgeBaseStackProps['guardrailConfig']>): bedrock.ContentFilter[] {
    const filters: bedrock.ContentFilter[] = [];
    
    if (config.hateThreshold !== undefined) {
      filters.push({
        type: bedrock.ContentFilterType.HATE,
        inputStrength: this.mapThresholdToStrength(config.hateThreshold),
        outputStrength: this.mapThresholdToStrength(config.hateThreshold),
      });
    }
    
    if (config.insultsThreshold !== undefined) {
      filters.push({
        type: bedrock.ContentFilterType.INSULTS,
        inputStrength: this.mapThresholdToStrength(config.insultsThreshold),
        outputStrength: this.mapThresholdToStrength(config.insultsThreshold),
      });
    }
    
    if (config.sexualThreshold !== undefined) {
      filters.push({
        type: bedrock.ContentFilterType.SEXUAL,
        inputStrength: this.mapThresholdToStrength(config.sexualThreshold),
        outputStrength: this.mapThresholdToStrength(config.sexualThreshold),
      });
    }
    
    if (config.violenceThreshold !== undefined) {
      filters.push({
        type: bedrock.ContentFilterType.VIOLENCE,
        inputStrength: this.mapThresholdToStrength(config.violenceThreshold),
        outputStrength: this.mapThresholdToStrength(config.violenceThreshold),
      });
    }
    
    return filters;
  }
  
  private mapThresholdToStrength(threshold: number): bedrock.ContentFilterStrength {
    if (threshold >= 0.9) return bedrock.ContentFilterStrength.NONE;
    if (threshold >= 0.7) return bedrock.ContentFilterStrength.LOW;
    if (threshold >= 0.5) return bedrock.ContentFilterStrength.MEDIUM;
    return bedrock.ContentFilterStrength.HIGH;
  }
}