import * as cdk from 'aws-cdk-lib';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import { Construct } from 'constructs';
import { Config } from './config';

export interface StorageStackProps extends cdk.StackProps {
  config: Config;
}

export class StorageStack extends cdk.Stack {
  public readonly botsTable: dynamodb.Table;
  public readonly conversationsTable: dynamodb.Table;
  public readonly messagesTable: dynamodb.Table;
  public readonly vaultTable: dynamodb.Table;
  public readonly storageBucket: s3.Bucket;
  public readonly allTables: dynamodb.Table[];
  
  constructor(scope: Construct, id: string, props: StorageStackProps) {
    super(scope, id, props);
    
    // DynamoDB table for bots
    this.botsTable = new dynamodb.Table(this, 'BotsTable', {
      tableName: `${props.config.prefix}Bots`,
      partitionKey: { name: 'ID', type: dynamodb.AttributeType.STRING },
      ...props.config.dynamoTableProps,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: props.config.env === 'prod' },
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
    });
    
    // Add GSI for owner ID to efficiently query bots by owner
    this.botsTable.addGlobalSecondaryIndex({
      indexName: 'OwnerIDIndex',
      partitionKey: { name: 'OwnerID', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });
    
    // DynamoDB table for conversations
    this.conversationsTable = new dynamodb.Table(this, 'ConversationsTable', {
      tableName: `${props.config.prefix}Conversations`,
      partitionKey: { name: 'UserID', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'ID', type: dynamodb.AttributeType.STRING },
      ...props.config.dynamoTableProps,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: props.config.env === 'prod' },
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
    });
    
    // Add GSI for bot ID to efficiently query conversations by bot
    this.conversationsTable.addGlobalSecondaryIndex({
      indexName: 'BotIDIndex',
      partitionKey: { name: 'BotID', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'UpdatedAt', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });
    
    // DynamoDB table for messages
    this.messagesTable = new dynamodb.Table(this, 'MessagesTable', {
      tableName: `${props.config.prefix}Messages`,
      partitionKey: { name: 'ConversationID', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'Timestamp', type: dynamodb.AttributeType.NUMBER },
      ...props.config.dynamoTableProps,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: props.config.env === 'prod' },
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
    });

    // DynamoDB table for API key vault (encrypted storage)
    this.vaultTable = new dynamodb.Table(this, 'VaultTable', {
      tableName: `${props.config.prefix}Vault`,
      partitionKey: { name: 'UserID', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'ServiceID', type: dynamodb.AttributeType.STRING },
      ...props.config.dynamoTableProps,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      pointInTimeRecoverySpecification: { pointInTimeRecoveryEnabled: props.config.env === 'prod' },
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
    });
    
    // S3 bucket for storing files
    this.storageBucket = new s3.Bucket(this, 'StorageBucket', {
      bucketName: `${props.config.prefix.toLowerCase()}storage-${this.account}`.toLowerCase(),
      encryption: s3.BucketEncryption.S3_MANAGED,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      cors: [
        {
          allowedMethods: [s3.HttpMethods.GET, s3.HttpMethods.POST, s3.HttpMethods.PUT],
          allowedOrigins: ['*'], // Update to specific domains in production
          allowedHeaders: ['*'],
          maxAge: 3000,
        },
      ],
      lifecycleRules: [
        {
          id: 'temp-file-cleanup',
          prefix: 'temp/',
          expiration: cdk.Duration.days(1),
        },
      ],
    });
    
    // Collect all tables for monitoring
    this.allTables = [this.botsTable, this.conversationsTable, this.messagesTable, this.vaultTable];
    
    // Export outputs
    new cdk.CfnOutput(this, 'BotsTableName', {
      value: this.botsTable.tableName,
      description: 'Bots table name',
      exportName: `${props.config.prefix}BotsTableName`,
    });
    
    new cdk.CfnOutput(this, 'ConversationsTableName', {
      value: this.conversationsTable.tableName,
      description: 'Conversations table name',
      exportName: `${props.config.prefix}ConversationsTableName`,
    });
    
    new cdk.CfnOutput(this, 'MessagesTableName', {
      value: this.messagesTable.tableName,
      description: 'Messages table name',
      exportName: `${props.config.prefix}MessagesTableName`,
    });
    
    new cdk.CfnOutput(this, 'StorageBucketName', {
      value: this.storageBucket.bucketName,
      description: 'Storage bucket name',
      exportName: `${props.config.prefix}StorageBucketName`,
    });

    new cdk.CfnOutput(this, 'VaultTableName', {
      value: this.vaultTable.tableName,
      description: 'Vault table name',
      exportName: `${props.config.prefix}VaultTableName`,
    });
  }
}