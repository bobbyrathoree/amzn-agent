# AWS CDK Infrastructure Design

This document outlines the AWS CDK infrastructure design in TypeScript for the AI Chat platform.

## Project Structure

```
infrastructure/
├── bin/
│   └── app.ts             # Main CDK application entry point
├── lib/
│   ├── api-stack.ts       # API Gateway, WebSocket, Lambda functions
│   ├── auth-stack.ts      # Cognito User Pool, Identity Pool
│   ├── config.ts          # Configuration parameters
│   ├── frontend-stack.ts  # S3 website, CloudFront distribution
│   ├── monitoring-stack.ts # CloudWatch dashboards, alarms
│   ├── network-stack.ts   # VPC, subnets, security groups
│   └── storage-stack.ts   # DynamoDB tables, S3 buckets, OpenSearch
├── .env.example           # Example environment variables
├── .gitignore             # Git ignore patterns
├── cdk.json               # CDK configuration
├── package.json           # NPM dependencies and scripts
├── README.md              # Documentation
└── tsconfig.json          # TypeScript configuration
```

## Stack Implementation Details

### 1. Main CDK Application (`bin/app.ts`)

```typescript
#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { Config } from '../lib/config';
import { NetworkStack } from '../lib/network-stack';
import { StorageStack } from '../lib/storage-stack';
import { AuthStack } from '../lib/auth-stack';
import { ApiStack } from '../lib/api-stack';
import { FrontendStack } from '../lib/frontend-stack';
import { MonitoringStack } from '../lib/monitoring-stack';

// Load environment variables
const app = new cdk.App();
const env = app.node.tryGetContext('env') || 'dev';
const config = new Config(env);

// Create stacks
const networkStack = new NetworkStack(app, `${config.prefix}NetworkStack`, {
  env: config.envProps,
  config,
});

const storageStack = new StorageStack(app, `${config.prefix}StorageStack`, {
  env: config.envProps,
  config,
});

const authStack = new AuthStack(app, `${config.prefix}AuthStack`, {
  env: config.envProps,
  config,
});

const apiStack = new ApiStack(app, `${config.prefix}ApiStack`, {
  env: config.envProps,
  config,
  vpc: networkStack.vpc,
  lambdaSecurityGroup: networkStack.lambdaSecurityGroup,
  userPool: authStack.userPool,
  userPoolClient: authStack.userPoolClient,
  botsTable: storageStack.botsTable,
  conversationsTable: storageStack.conversationsTable,
  messagesTable: storageStack.messagesTable,
});

const frontendStack = new FrontendStack(app, `${config.prefix}FrontendStack`, {
  env: config.envProps,
  config,
  userPool: authStack.userPool,
  userPoolClient: authStack.userPoolClient,
  identityPool: authStack.identityPool,
  apiEndpoint: apiStack.apiEndpoint,
  websocketEndpoint: apiStack.websocketEndpoint,
});

const monitoringStack = new MonitoringStack(app, `${config.prefix}MonitoringStack`, {
  env: config.envProps,
  config,
  apiGateway: apiStack.apiGateway,
  lambdaFunctions: apiStack.lambdaFunctions,
  dynamoTables: storageStack.allTables,
});

// Apply tags to all stacks
const tags = {
  Environment: env,
  Project: 'AIChatPlatform',
  ManagedBy: 'CDK',
};

Object.entries(tags).forEach(([key, value]) => {
  cdk.Tags.of(app).add(key, value);
});
```

### 2. Configuration (`lib/config.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';

export class Config {
  public readonly prefix: string;
  public readonly envProps: cdk.Environment;
  public readonly domainName?: string;
  public readonly certificateArn?: string;
  public readonly lambdaMemory: number;
  public readonly lambdaTimeout: cdk.Duration;
  public readonly dynamoTableProps: { [key: string]: any };
  public readonly apiProps: { [key: string]: any };
  
  constructor(private readonly env: string) {
    this.prefix = `${env}-AIChatPlatform-`;
    
    this.envProps = {
      account: process.env.CDK_DEFAULT_ACCOUNT,
      region: process.env.CDK_DEFAULT_REGION || 'us-west-2',
    };
    
    // Domain configuration (if available)
    this.domainName = process.env.DOMAIN_NAME;
    this.certificateArn = process.env.CERTIFICATE_ARN;
    
    // Lambda settings
    this.lambdaMemory = env === 'prod' ? 1024 : 512;
    this.lambdaTimeout = env === 'prod' 
      ? cdk.Duration.seconds(60) 
      : cdk.Duration.seconds(30);
    
    // DynamoDB settings
    this.dynamoTableProps = {
      billingMode: env === 'prod' 
        ? cdk.aws_dynamodb.BillingMode.PROVISIONED 
        : cdk.aws_dynamodb.BillingMode.PAY_PER_REQUEST,
      readCapacity: env === 'prod' ? 5 : undefined,
      writeCapacity: env === 'prod' ? 5 : undefined,
    };
    
    // API Gateway settings
    this.apiProps = {
      throttlingBurstLimit: env === 'prod' ? 1000 : 500,
      throttlingRateLimit: env === 'prod' ? 2000 : 1000,
    };
  }
}
```

### 3. Network Stack (`lib/network-stack.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import { Construct } from 'constructs';
import { Config } from './config';

export interface NetworkStackProps extends cdk.StackProps {
  config: Config;
}

export class NetworkStack extends cdk.Stack {
  public readonly vpc: ec2.Vpc;
  public readonly lambdaSecurityGroup: ec2.SecurityGroup;
  
  constructor(scope: Construct, id: string, props: NetworkStackProps) {
    super(scope, id, props);
    
    // Create a VPC for isolation and security
    this.vpc = new ec2.Vpc(this, 'Vpc', {
      maxAzs: 2,
      subnetConfiguration: [
        {
          name: 'Public',
          subnetType: ec2.SubnetType.PUBLIC,
          cidrMask: 24,
        },
        {
          name: 'Private',
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
          cidrMask: 24,
        },
        {
          name: 'Isolated',
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
          cidrMask: 24,
        },
      ],
    });
    
    // Security group for Lambda functions
    this.lambdaSecurityGroup = new ec2.SecurityGroup(this, 'LambdaSecurityGroup', {
      vpc: this.vpc,
      description: 'Security group for Lambda functions',
      allowAllOutbound: true,
    });
    
    // Export outputs
    new cdk.CfnOutput(this, 'VpcId', {
      value: this.vpc.vpcId,
      description: 'VPC ID',
      exportName: `${props.config.prefix}VpcId`,
    });
    
    new cdk.CfnOutput(this, 'LambdaSecurityGroupId', {
      value: this.lambdaSecurityGroup.securityGroupId,
      description: 'Lambda Security Group ID',
      exportName: `${props.config.prefix}LambdaSecurityGroupId`,
    });
  }
}
```

### 4. Storage Stack (`lib/storage-stack.ts`)

```typescript
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
      pointInTimeRecovery: props.config.env === 'prod',
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
      pointInTimeRecovery: props.config.env === 'prod',
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
      pointInTimeRecovery: props.config.env === 'prod',
      encryption: dynamodb.TableEncryption.AWS_MANAGED,
    });
    
    // S3 bucket for storing files
    this.storageBucket = new s3.Bucket(this, 'StorageBucket', {
      bucketName: `${props.config.prefix.toLowerCase()}storage-${this.account}`,
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
    this.allTables = [this.botsTable, this.conversationsTable, this.messagesTable];
    
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
  }
}
```

### 5. Auth Stack (`lib/auth-stack.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import { Config } from './config';

export interface AuthStackProps extends cdk.StackProps {
  config: Config;
}

export class AuthStack extends cdk.Stack {
  public readonly userPool: cognito.UserPool;
  public readonly userPoolClient: cognito.UserPoolClient;
  public readonly identityPool: cognito.CfnIdentityPool;
  public readonly authenticatedRole: iam.Role;
  
  constructor(scope: Construct, id: string, props: AuthStackProps) {
    super(scope, id, props);
    
    // Create user pool
    this.userPool = new cognito.UserPool(this, 'UserPool', {
      userPoolName: `${props.config.prefix}UserPool`,
      selfSignUpEnabled: true,
      signInAliases: {
        email: true,
      },
      autoVerify: {
        email: true,
      },
      standardAttributes: {
        email: {
          required: true,
          mutable: true,
        },
        fullname: {
          required: false,
          mutable: true,
        },
      },
      passwordPolicy: {
        minLength: 8,
        requireLowercase: true,
        requireUppercase: true,
        requireDigits: true,
        requireSymbols: true,
      },
      accountRecovery: cognito.AccountRecovery.EMAIL_ONLY,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });
    
    // Define groups for role-based access
    const adminGroup = new cognito.CfnUserPoolGroup(this, 'AdminGroup', {
      userPoolId: this.userPool.userPoolId,
      groupName: 'Admin',
      description: 'Administrators with full access',
    });
    
    const botCreatorsGroup = new cognito.CfnUserPoolGroup(this, 'BotCreatorsGroup', {
      userPoolId: this.userPool.userPoolId,
      groupName: 'BotCreators',
      description: 'Users who can create bots',
    });
    
    // Create user pool client
    this.userPoolClient = this.userPool.addClient('UserPoolClient', {
      userPoolClientName: `${props.config.prefix}UserPoolClient`,
      authFlows: {
        userPassword: true,
        userSrp: true,
      },
      preventUserExistenceErrors: true,
      refreshTokenValidity: cdk.Duration.days(30),
      accessTokenValidity: cdk.Duration.hours(1),
      idTokenValidity: cdk.Duration.hours(1),
      supportedIdentityProviders: [
        cognito.UserPoolClientIdentityProvider.COGNITO,
      ],
    });
    
    // Create identity pool
    this.identityPool = new cognito.CfnIdentityPool(this, 'IdentityPool', {
      identityPoolName: `${props.config.prefix}IdentityPool`,
      allowUnauthenticatedIdentities: false,
      cognitoIdentityProviders: [
        {
          clientId: this.userPoolClient.userPoolClientId,
          providerName: this.userPool.userPoolProviderName,
        },
      ],
    });
    
    // Define authenticated role
    this.authenticatedRole = new iam.Role(this, 'AuthenticatedRole', {
      assumedBy: new iam.FederatedPrincipal(
        'cognito-identity.amazonaws.com',
        {
          StringEquals: {
            'cognito-identity.amazonaws.com:aud': this.identityPool.ref,
          },
          'ForAnyValue:StringLike': {
            'cognito-identity.amazonaws.com:amr': 'authenticated',
          },
        },
        'sts:AssumeRoleWithWebIdentity'
      ),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonBedrockReadOnly'),
      ],
    });
    
    // Attach role to identity pool
    new cognito.CfnIdentityPoolRoleAttachment(this, 'IdentityPoolRoleAttachment', {
      identityPoolId: this.identityPool.ref,
      roles: {
        authenticated: this.authenticatedRole.roleArn,
      },
      roleMappings: {
        userPool: {
          identityProvider: `cognito-idp.${this.region}.amazonaws.com/${this.userPool.userPoolId}:${this.userPoolClient.userPoolClientId}`,
          type: 'Token',
          ambiguousRoleResolution: 'AuthenticatedRole',
        },
      },
    });
    
    // Export outputs
    new cdk.CfnOutput(this, 'UserPoolId', {
      value: this.userPool.userPoolId,
      description: 'User Pool ID',
      exportName: `${props.config.prefix}UserPoolId`,
    });
    
    new cdk.CfnOutput(this, 'UserPoolClientId', {
      value: this.userPoolClient.userPoolClientId,
      description: 'User Pool Client ID',
      exportName: `${props.config.prefix}UserPoolClientId`,
    });
    
    new cdk.CfnOutput(this, 'IdentityPoolId', {
      value: this.identityPool.ref,
      description: 'Identity Pool ID',
      exportName: `${props.config.prefix}IdentityPoolId`,
    });
  }
}
```

### 6. API Stack (`lib/api-stack.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as path from 'path';
import { Construct } from 'constructs';
import { Config } from './config';

export interface ApiStackProps extends cdk.StackProps {
  config: Config;
  vpc: ec2.Vpc;
  lambdaSecurityGroup: ec2.SecurityGroup;
  userPool: cognito.UserPool;
  userPoolClient: cognito.UserPoolClient;
  botsTable: dynamodb.Table;
  conversationsTable: dynamodb.Table;
  messagesTable: dynamodb.Table;
}

export class ApiStack extends cdk.Stack {
  public readonly apiGateway: apigateway.RestApi;
  public readonly websocket: apigateway.WebSocketApi;
  public readonly apiEndpoint: string;
  public readonly websocketEndpoint: string;
  public readonly lambdaFunctions: lambda.Function[] = [];
  
  constructor(scope: Construct, id: string, props: ApiStackProps) {
    super(scope, id, props);
    
    // Create REST API Gateway
    this.apiGateway = new apigateway.RestApi(this, 'RestApi', {
      restApiName: `${props.config.prefix}API`,
      description: 'API for the AI Chat Platform',
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
        allowHeaders: [
          'Content-Type',
          'Authorization',
          'X-Amz-Date',
          'X-Api-Key',
        ],
        allowCredentials: true,
      },
      deployOptions: {
        stageName: props.config.env,
        loggingLevel: apigateway.MethodLoggingLevel.INFO,
        dataTraceEnabled: true,
        metricsEnabled: true,
      },
    });
    
    // Create authorizer
    const authorizer = new apigateway.CognitoUserPoolsAuthorizer(this, 'ApiAuthorizer', {
      cognitoUserPools: [props.userPool],
      identitySources: ['method.request.header.Authorization'],
    });
    
    // Create Lambda role with necessary permissions
    const lambdaRole = new iam.Role(this, 'LambdaRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaVPCAccessExecutionRole'),
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonBedrockFullAccess'),
      ],
    });
    
    // Grant DynamoDB permissions
    props.botsTable.grantReadWriteData(lambdaRole);
    props.conversationsTable.grantReadWriteData(lambdaRole);
    props.messagesTable.grantReadWriteData(lambdaRole);
    
    // Create Lambda functions
    const chatLambda = this.createLambdaFunction('ChatFunction', 'chat', props, lambdaRole);
    const botsLambda = this.createLambdaFunction('BotsFunction', 'bots', props, lambdaRole);
    const knowledgeLambda = this.createLambdaFunction('KnowledgeFunction', 'knowledge', props, lambdaRole);
    
    // Define API routes
    // Bots API
    const botsResource = this.apiGateway.root.addResource('bots');
    this.addCrudEndpoints(botsResource, botsLambda, authorizer);
    
    // Individual bot API
    const botResource = botsResource.addResource('{id}');
    this.addCrudEndpoints(botResource, botsLambda, authorizer, ['GET', 'PUT', 'DELETE']);
    
    // Chat API
    const chatResource = this.apiGateway.root.addResource('chat');
    chatResource.addMethod('POST', new apigateway.LambdaIntegration(chatLambda), {
      authorizer,
    });
    
    // Conversations API
    const conversationsResource = this.apiGateway.root.addResource('conversations');
    conversationsResource.addMethod('GET', new apigateway.LambdaIntegration(chatLambda), {
      authorizer,
    });
    
    const conversationResource = conversationsResource.addResource('{id}');
    conversationResource.addMethod('GET', new apigateway.LambdaIntegration(chatLambda), {
      authorizer,
    });
    conversationResource.addMethod('DELETE', new apigateway.LambdaIntegration(chatLambda), {
      authorizer,
    });
    
    // Knowledge bases API
    const knowledgeResource = this.apiGateway.root.addResource('knowledge-bases');
    knowledgeResource.addMethod('GET', new apigateway.LambdaIntegration(knowledgeLambda), {
      authorizer,
    });
    
    // Create WebSocket API for streaming
    this.websocket = new apigateway.WebSocketApi(this, 'WebSocketApi', {
      apiName: `${props.config.prefix}WebSocketAPI`,
      connectRouteOptions: {
        integration: new apigateway.WebSocketLambdaIntegration('ConnectIntegration', 
          this.createLambdaFunction('WebSocketConnectFunction', 'websocket', props, lambdaRole, 'connect')
        ),
      },
      disconnectRouteOptions: {
        integration: new apigateway.WebSocketLambdaIntegration('DisconnectIntegration', 
          this.createLambdaFunction('WebSocketDisconnectFunction', 'websocket', props, lambdaRole, 'disconnect')
        ),
      },
      defaultRouteOptions: {
        integration: new apigateway.WebSocketLambdaIntegration('DefaultIntegration', 
          this.createLambdaFunction('WebSocketDefaultFunction', 'websocket', props, lambdaRole, 'default')
        ),
      },
    });
    
    // Deploy WebSocket API
    const webSocketStage = new apigateway.WebSocketStage(this, 'WebSocketStage', {
      webSocketApi: this.websocket,
      stageName: props.config.env,
      autoDeploy: true,
    });
    
    // Set API endpoints
    this.apiEndpoint = this.apiGateway.url;
    this.websocketEndpoint = webSocketStage.url;
    
    // Export outputs
    new cdk.CfnOutput(this, 'RestApiEndpoint', {
      value: this.apiEndpoint,
      description: 'REST API Endpoint',
      exportName: `${props.config.prefix}RestApiEndpoint`,
    });
    
    new cdk.CfnOutput(this, 'WebSocketEndpoint', {
      value: this.websocketEndpoint,
      description: 'WebSocket API Endpoint',
      exportName: `${props.config.prefix}WebSocketEndpoint`,
    });
  }
  
  private createLambdaFunction(
    id: string, 
    handlerDir: string, 
    props: ApiStackProps, 
    role: iam.Role, 
    handlerSuffix = 'main'
  ): lambda.Function {
    // Set environment variables
    const environment = {
      ENVIRONMENT: props.config.env,
      BOTS_TABLE: props.botsTable.tableName,
      CONVERSATIONS_TABLE: props.conversationsTable.tableName,
      MESSAGES_TABLE: props.messagesTable.tableName,
      USER_POOL_ID: props.userPool.userPoolId,
      REGION: props.env?.region || 'us-east-1',
    };
    
    // Create the Lambda function
    const fn = new lambda.Function(this, id, {
      runtime: lambda.Runtime.GO_1_X,
      handler: `bootstrap`,
      code: lambda.Code.fromAsset(path.join(__dirname, '../../go-lambda-backend/cmd/functions', handlerDir), {
        bundling: {
          image: lambda.Runtime.GO_1_X.bundlingImage,
          command: [
            'bash', '-c', [
              'apt-get update',
              'apt-get install -y zip',
              'go build -o bootstrap',
              'zip -r /asset-output/function.zip bootstrap',
            ].join(' && '),
          ],
          user: 'root',
        },
      }),
      memorySize: props.config.lambdaMemory,
      timeout: props.config.lambdaTimeout,
      environment,
      role,
      vpc: props.vpc,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
      },
      securityGroups: [props.lambdaSecurityGroup],
      logRetention: logs.RetentionDays.ONE_WEEK,
    });
    
    // Add to the list of functions for monitoring
    this.lambdaFunctions.push(fn);
    
    return fn;
  }
  
  private addCrudEndpoints(
    resource: apigateway.Resource, 
    lambdaFn: lambda.Function, 
    authorizer: apigateway.CognitoUserPoolsAuthorizer,
    methods: string[] = ['GET', 'POST', 'PUT', 'DELETE']
  ) {
    const integration = new apigateway.LambdaIntegration(lambdaFn);
    
    if (methods.includes('GET')) {
      resource.addMethod('GET', integration, { authorizer });
    }
    
    if (methods.includes('POST')) {
      resource.addMethod('POST', integration, { authorizer });
    }
    
    if (methods.includes('PUT')) {
      resource.addMethod('PUT', integration, { authorizer });
    }
    
    if (methods.includes('DELETE')) {
      resource.addMethod('DELETE', integration, { authorizer });
    }
  }
}
```

### 7. Frontend Stack (`lib/frontend-stack.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as acm from 'aws-cdk-lib/aws-certificatemanager';
import * as iam from 'aws-cdk-lib/aws-iam';
import { Construct } from 'constructs';
import * as path from 'path';
import { Config } from './config';

export interface FrontendStackProps extends cdk.StackProps {
  config: Config;
  userPool: cognito.UserPool;
  userPoolClient: cognito.UserPoolClient;
  identityPool: cognito.CfnIdentityPool;
  apiEndpoint: string;
  websocketEndpoint: string;
}

export class FrontendStack extends cdk.Stack {
  public readonly websiteBucket: s3.Bucket;
  public readonly distribution: cloudfront.Distribution;
  public readonly websiteUrl: string;
  
  constructor(scope: Construct, id: string, props: FrontendStackProps) {
    super(scope, id, props);
    
    // Create S3 bucket for website hosting
    this.websiteBucket = new s3.Bucket(this, 'WebsiteBucket', {
      bucketName: `${props.config.prefix.toLowerCase()}website-${this.account}`,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      cors: [
        {
          allowedMethods: [s3.HttpMethods.GET, s3.HttpMethods.HEAD],
          allowedOrigins: ['*'],
          allowedHeaders: ['*'],
        },
      ],
    });
    
    // Create Origin Access Identity
    const originAccessIdentity = new cloudfront.OriginAccessIdentity(this, 'OAI', {
      comment: 'OAI for website bucket',
    });
    
    // Grant access to the bucket
    this.websiteBucket.grantRead(originAccessIdentity);
    
    // Determine if custom domain is configured
    let certificate: acm.ICertificate | undefined;
    let domainNames: string[] | undefined;
    
    if (props.config.domainName && props.config.certificateArn) {
      certificate = acm.Certificate.fromCertificateArn(
        this, 'Certificate', props.config.certificateArn
      );
      domainNames = [props.config.domainName];
    }
    
    // Create CloudFront distribution
    this.distribution = new cloudfront.Distribution(this, 'Distribution', {
      defaultBehavior: {
        origin: new origins.S3Origin(this.websiteBucket, {
          originAccessIdentity,
        }),
        compress: true,
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD,
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
      },
      defaultRootObject: 'index.html',
      errorResponses: [
        {
          httpStatus: 404,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
        },
      ],
      domainNames,
      certificate,
      minimumProtocolVersion: cloudfront.SecurityPolicyProtocol.TLS_V1_2_2021,
    });
    
    // Create custom error response for SPAs
    // This ensures that routes like /bots/123 work with client-side routing
    const cfnDistribution = this.distribution.node.defaultChild as cloudfront.CfnDistribution;
    cfnDistribution.addPropertyOverride('DistributionConfig.CustomErrorResponses', [
      {
        ErrorCode: 403,
        ResponseCode: 200,
        ResponsePagePath: '/index.html',
        ErrorCachingMinTTL: 10,
      },
    ]);
    
    // Generate config file for frontend
    const configFileContent = JSON.stringify({
      environment: props.config.env,
      userPoolId: props.userPool.userPoolId,
      userPoolClientId: props.userPoolClient.userPoolClientId,
      identityPoolId: props.identityPool.ref,
      apiEndpoint: props.apiEndpoint,
      websocketEndpoint: props.websocketEndpoint,
      region: props.env?.region || 'us-west-2',
    }, null, 2);
    
    // Deploy website assets to S3
    new s3deploy.BucketDeployment(this, 'WebsiteDeployment', {
      sources: [
        s3deploy.Source.asset(path.join(__dirname, '../../frontend/out')),
        s3deploy.Source.data('config.json', configFileContent),
      ],
      destinationBucket: this.websiteBucket,
      distribution: this.distribution,
      distributionPaths: ['/*'],
    });
    
    // Set website URL based on domain configuration
    this.websiteUrl = props.config.domainName || `https://${this.distribution.distributionDomainName}`;
    
    // Export outputs
    new cdk.CfnOutput(this, 'WebsiteUrl', {
      value: this.websiteUrl,
      description: 'Website URL',
      exportName: `${props.config.prefix}WebsiteUrl`,
    });
    
    new cdk.CfnOutput(this, 'BucketName', {
      value: this.websiteBucket.bucketName,
      description: 'Website Bucket Name',
      exportName: `${props.config.prefix}WebsiteBucketName`,
    });
    
    new cdk.CfnOutput(this, 'DistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront Distribution ID',
      exportName: `${props.config.prefix}DistributionId`,
    });
  }
}
```

### 8. Monitoring Stack (`lib/monitoring-stack.ts`)

```typescript
import * as cdk from 'aws-cdk-lib';
import * as cloudwatch from 'aws-cdk-lib/aws-cloudwatch';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as sns from 'aws-cdk-lib/aws-sns';
import * as subscriptions from 'aws-cdk-lib/aws-sns-subscriptions';
import { Construct } from 'constructs';
import { Config } from './config';

export interface MonitoringStackProps extends cdk.StackProps {
  config: Config;
  apiGateway: apigateway.RestApi;
  lambdaFunctions: lambda.Function[];
  dynamoTables: dynamodb.Table[];
}

export class MonitoringStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props: MonitoringStackProps) {
    super(scope, id, props);
    
    // Create SNS topic for alerts
    const alertTopic = new sns.Topic(this, 'AlertTopic', {
      displayName: `${props.config.prefix}Alerts`,
      topicName: `${props.config.prefix}Alerts`,
    });
    
    // Add email subscription if provided
    if (process.env.ALERT_EMAIL) {
      alertTopic.addSubscription(
        new subscriptions.EmailSubscription(process.env.ALERT_EMAIL)
      );
    }
    
    // Create dashboard
    const dashboard = new cloudwatch.Dashboard(this, 'Dashboard', {
      dashboardName: `${props.config.prefix}Dashboard`,
    });
    
    // Add API Gateway metrics to dashboard
    const apiMetrics = this.createApiGatewayWidgets(props.apiGateway);
    dashboard.addWidgets(...apiMetrics);
    
    // Add Lambda metrics to dashboard
    const lambdaMetrics = this.createLambdaWidgets(props.lambdaFunctions);
    dashboard.addWidgets(...lambdaMetrics);
    
    // Add DynamoDB metrics to dashboard
    const dynamoMetrics = this.createDynamoWidgets(props.dynamoTables);
    dashboard.addWidgets(...dynamoMetrics);
    
    // Create alarms for critical services
    
    // API Gateway 5xx errors alarm
    const api5xxErrorAlarm = new cloudwatch.Alarm(this, 'Api5xxErrorAlarm', {
      metric: props.apiGateway.metricServerError({
        period: cdk.Duration.minutes(1),
        statistic: 'sum',
      }),
      evaluationPeriods: 3,
      threshold: 5,
      alarmDescription: 'API Gateway is returning 5xx errors',
      actionsEnabled: true,
    });
    
    api5xxErrorAlarm.addAlarmAction(new cloudwatch.SnsAction(alertTopic));
    
    // Lambda error rate alarms
    props.lambdaFunctions.forEach((fn, index) => {
      const errorAlarm = new cloudwatch.Alarm(this, `LambdaErrorAlarm-${index}`, {
        metric: fn.metricErrors({
          period: cdk.Duration.minutes(1),
        }),
        evaluationPeriods: 3,
        threshold: 5,
        alarmDescription: `Lambda function ${fn.functionName} has a high error rate`,
        actionsEnabled: true,
      });
      
      errorAlarm.addAlarmAction(new cloudwatch.SnsAction(alertTopic));
    });
    
    // DynamoDB throttling alarms
    props.dynamoTables.forEach((table, index) => {
      const readThrottleAlarm = new cloudwatch.Alarm(this, `DynamoReadThrottleAlarm-${index}`, {
        metric: table.metricReadThrottleEvents({
          period: cdk.Duration.minutes(5),
          statistic: 'sum',
        }),
        evaluationPeriods: 3,
        threshold: 10,
        alarmDescription: `DynamoDB table ${table.tableName} has read throttling events`,
        actionsEnabled: true,
      });
      
      readThrottleAlarm.addAlarmAction(new cloudwatch.SnsAction(alertTopic));
    });
    
    // Export outputs
    new cdk.CfnOutput(this, 'DashboardURL', {
      value: `https://${this.region}.console.aws.amazon.com/cloudwatch/home?region=${this.region}#dashboards:name=${dashboard.dashboardName}`,
      description: 'Dashboard URL',
      exportName: `${props.config.prefix}DashboardURL`,
    });
    
    new cdk.CfnOutput(this, 'AlertTopicArn', {
      value: alertTopic.topicArn,
      description: 'Alert Topic ARN',
      exportName: `${props.config.prefix}AlertTopicArn`,
    });
  }
  
  private createApiGatewayWidgets(api: apigateway.RestApi): cloudwatch.IWidget[] {
    return [
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Requests',
        left: [
          api.metricCount({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Latency',
        left: [
          api.metricLatency({ statistic: 'avg', period: cdk.Duration.minutes(1) }),
          api.metricLatency({ statistic: 'p90', period: cdk.Duration.minutes(1) }),
          api.metricLatency({ statistic: 'p99', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'API Gateway - Errors',
        left: [
          api.metricClientError({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
          api.metricServerError({ statistic: 'sum', period: cdk.Duration.minutes(1) }),
        ],
        width: 12,
      }),
    ];
  }
  
  private createLambdaWidgets(functions: lambda.Function[]): cloudwatch.IWidget[] {
    const widgets: cloudwatch.IWidget[] = [];
    
    if (functions.length === 0) {
      return widgets;
    }
    
    // Invocations widget
    const invocationsMetrics: cloudwatch.IMetric[] = [];
    const errorsMetrics: cloudwatch.IMetric[] = [];
    const durationMetrics: cloudwatch.IMetric[] = [];
    
    functions.forEach(fn => {
      invocationsMetrics.push(fn.metricInvocations({
        statistic: 'sum',
        period: cdk.Duration.minutes(1),
      }));
      
      errorsMetrics.push(fn.metricErrors({
        statistic: 'sum',
        period: cdk.Duration.minutes(1),
      }));
      
      durationMetrics.push(fn.metricDuration({
        statistic: 'avg',
        period: cdk.Duration.minutes(1),
      }));
    });
    
    widgets.push(
      new cloudwatch.GraphWidget({
        title: 'Lambda - Invocations',
        left: invocationsMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Lambda - Errors',
        left: errorsMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'Lambda - Duration',
        left: durationMetrics,
        width: 12,
      })
    );
    
    return widgets;
  }
  
  private createDynamoWidgets(tables: dynamodb.Table[]): cloudwatch.IWidget[] {
    const widgets: cloudwatch.IWidget[] = [];
    
    if (tables.length === 0) {
      return widgets;
    }
    
    const readCapacityMetrics: cloudwatch.IMetric[] = [];
    const writeCapacityMetrics: cloudwatch.IMetric[] = [];
    const throttleMetrics: cloudwatch.IMetric[] = [];
    
    tables.forEach(table => {
      readCapacityMetrics.push(table.metricConsumedReadCapacityUnits({
        statistic: 'sum',
        period: cdk.Duration.minutes(5),
      }));
      
      writeCapacityMetrics.push(table.metricConsumedWriteCapacityUnits({
        statistic: 'sum',
        period: cdk.Duration.minutes(5),
      }));
      
      throttleMetrics.push(
        table.metricReadThrottleEvents({
          statistic: 'sum',
          period: cdk.Duration.minutes(5),
        }),
        table.metricWriteThrottleEvents({
          statistic: 'sum',
          period: cdk.Duration.minutes(5),
        })
      );
    });
    
    widgets.push(
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Read Capacity',
        left: readCapacityMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Write Capacity',
        left: writeCapacityMetrics,
        width: 12,
      }),
      new cloudwatch.GraphWidget({
        title: 'DynamoDB - Throttle Events',
        left: throttleMetrics,
        width: 12,
      })
    );
    
    return widgets;
  }
}
```

## Required Environment Variables

| Variable            | Description                              | Required?  |
|---------------------|------------------------------------------|------------|
| CDK_DEFAULT_ACCOUNT | AWS account ID for deployment            | Yes        |
| CDK_DEFAULT_REGION  | AWS region for deployment                | Yes        |
| DOMAIN_NAME         | Custom domain for CloudFront distribution| No         |
| CERTIFICATE_ARN     | ACM certificate ARN for HTTPS            | No         |
| ALERT_EMAIL         | Email address to receive alerts          | No         |

## Deployment Commands

1. Install dependencies:
   ```
   npm install
   ```

2. Bootstrap CDK (if not already done):
   ```
   npx cdk bootstrap
   ```

3. Deploy to development environment:
   ```
   npx cdk deploy --all --context env=dev
   ```

4. Deploy to production environment:
   ```
   npx cdk deploy --all --context env=prod
   ```

5. Destroy the stacks:
   ```
   npx cdk destroy --all --context env=dev
   ```

## Security Considerations

1. All data is encrypted at rest:
   - S3 buckets use SSE-S3 encryption
   - DynamoDB tables use AWS managed encryption
   - CloudWatch logs are encrypted

2. All data is encrypted in transit:
   - API Gateway endpoints use HTTPS
   - CloudFront distribution enforces HTTPS
   - WebSocket connections use WSS

3. Authentication and authorization:
   - Cognito User Pools for authentication
   - API Gateway authorizers for endpoint protection
   - IAM roles with least privilege principle

4. Network security:
   - Lambda functions run in private subnets
   - Security groups control network access
   - S3 buckets block public access

5. Monitoring and alerting:
   - CloudWatch dashboards for visibility
   - CloudWatch alarms for critical metrics
   - SNS topics for alert notifications