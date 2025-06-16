import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as s3 from 'aws-cdk-lib/aws-s3';
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
  storageBucket: s3.Bucket;
}

export class ApiStack extends cdk.Stack {
  public readonly apiGateway: apigateway.RestApi;
  public readonly websocket: apigatewayv2.WebSocketApi;
  public readonly apiEndpoint: string;
  public readonly websocketEndpoint: string;
  public readonly lambdaFunctions: lambda.Function[] = [];
  
  constructor(scope: Construct, id: string, props: ApiStackProps) {
    super(scope, id, props);
    
    // Create REST API Gateway
    this.apiGateway = new apigateway.RestApi(this, 'RestApi', {
      restApiName: `${props.config.prefix}API`,
      description: 'API for the AI Chat Platform',
      endpointConfiguration: {
        types: [apigateway.EndpointType.REGIONAL]
      },
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
        allowHeaders: [
          'Content-Type',
          'Authorization',
          'X-Amz-Date',
          'X-Api-Key',
          'X-User-ID',
        ],
        allowCredentials: true,
      },
      deployOptions: {
        stageName: props.config.env,
        loggingLevel: apigateway.MethodLoggingLevel.INFO,
        dataTraceEnabled: false,
        metricsEnabled: true,
      },
    });
    
    // Create authorizer
    const authorizer = new apigateway.CognitoUserPoolsAuthorizer(this, 'ApiAuthorizer', {
      cognitoUserPools: [props.userPool],
      identitySource: 'method.request.header.Authorization',
      authorizerName: `${props.config.prefix}CognitoAuthorizer`,
      resultsCacheTtl: cdk.Duration.minutes(5), // Cache results for 5 minutes
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
    
    // Grant comprehensive DynamoDB permissions for all service operations
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'dynamodb:ListTables',
        'dynamodb:DescribeTable',
        'dynamodb:CreateTable',
        'dynamodb:UpdateTable',
        'dynamodb:DeleteTable',
        'dynamodb:DescribeTimeToLive',
        'dynamodb:UpdateTimeToLive',
        'dynamodb:DescribeContinuousBackups',
        'dynamodb:UpdateContinuousBackups',
        'dynamodb:DescribeBackup',
        'dynamodb:CreateBackup',
        'dynamodb:DeleteBackup',
        'dynamodb:RestoreTableFromBackup',
        'dynamodb:RestoreTableToPointInTime',
        'dynamodb:TagResource',
        'dynamodb:UntagResource',
        'dynamodb:ListTagsOfResource',
      ],
      resources: ['*'], // Management operations require wildcard
    }));
    
    // Grant comprehensive item-level DynamoDB permissions
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'dynamodb:PutItem',
        'dynamodb:GetItem',
        'dynamodb:UpdateItem',
        'dynamodb:DeleteItem',
        'dynamodb:BatchGetItem',
        'dynamodb:BatchWriteItem',
        'dynamodb:Query',
        'dynamodb:Scan',
        'dynamodb:ConditionCheckItem',
        'dynamodb:TransactGetItems',
        'dynamodb:TransactWriteItems',
      ],
      resources: [
        props.botsTable.tableArn,
        `${props.botsTable.tableArn}/*`,
        props.conversationsTable.tableArn,
        `${props.conversationsTable.tableArn}/*`,
        props.messagesTable.tableArn,
        `${props.messagesTable.tableArn}/*`,
      ],
    }));
    
    // Grant comprehensive S3 permissions
    props.storageBucket.grantReadWrite(lambdaRole);
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        's3:ListAllMyBuckets',
        's3:GetBucketLocation',
        's3:GetBucketVersioning',
        's3:GetBucketPolicy',
        's3:GetBucketAcl',
        's3:ListBucket',
      ],
      resources: ['*'], // Bucket listing operations
    }));
    
    // Create VPC endpoints for all AWS services our backend uses
    
    // Gateway VPC Endpoints (free)
    const gatewayEndpoints = [
      { name: 'DynamoDB', service: ec2.GatewayVpcEndpointAwsService.DYNAMODB },
      { name: 'S3', service: ec2.GatewayVpcEndpointAwsService.S3 },
    ];
    
    gatewayEndpoints.forEach(endpoint => {
      new ec2.GatewayVpcEndpoint(this, `${endpoint.name}Endpoint`, {
        vpc: props.vpc,
        service: endpoint.service,
        // Gateway endpoints automatically route through route tables, no subnet specification needed
      });
    });
    
    // Interface VPC Endpoints (paid) - Add more comprehensive coverage
    const interfaceEndpoints = [
      { name: 'Bedrock', service: ec2.InterfaceVpcEndpointAwsService.BEDROCK_RUNTIME },
      { name: 'BedrockAgent', service: ec2.InterfaceVpcEndpointAwsService.BEDROCK_AGENT },
      { name: 'BedrockAgentRuntime', service: ec2.InterfaceVpcEndpointAwsService.BEDROCK_AGENT_RUNTIME },
      { name: 'CloudFormation', service: ec2.InterfaceVpcEndpointAwsService.CLOUDFORMATION },
      { name: 'SecretsManager', service: ec2.InterfaceVpcEndpointAwsService.SECRETS_MANAGER },
      { name: 'CloudWatchLogs', service: ec2.InterfaceVpcEndpointAwsService.CLOUDWATCH_LOGS },
      { name: 'Lambda', service: ec2.InterfaceVpcEndpointAwsService.LAMBDA },
      { name: 'IAM', service: ec2.InterfaceVpcEndpointAwsService.IAM },
      { name: 'STS', service: ec2.InterfaceVpcEndpointAwsService.STS },
    ];
    
    interfaceEndpoints.forEach(endpoint => {
      new ec2.InterfaceVpcEndpoint(this, `${endpoint.name}Endpoint`, {
        vpc: props.vpc,
        service: endpoint.service,
        subnets: { subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS },
        securityGroups: [props.lambdaSecurityGroup],
        privateDnsEnabled: true, // Critical for DNS resolution
        // No explicit policy - will use default VPC endpoint policy
      });
    });
    
    // Grant CloudFormation permissions for Knowledge Base stack deployments
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'cloudformation:CreateStack',
        'cloudformation:DescribeStacks',
        'cloudformation:UpdateStack',
        'cloudformation:DeleteStack',
        'cloudformation:ListStackResources',
        'cloudformation:DescribeStackEvents',
        'cloudformation:GetTemplate',
      ],
      resources: [
        `arn:aws:cloudformation:${this.region}:${this.account}:stack/BrChatKbStack*/*`,
        `arn:aws:cloudformation:${this.region}:${this.account}:stack/${props.config.prefix}*/*`,
      ],
    }));

    // Grant comprehensive CloudFormation permissions
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'cloudformation:ListStacks',
        'cloudformation:DescribeStacks',
        'cloudformation:DescribeStackEvents',
        'cloudformation:DescribeStackResources',
        'cloudformation:DescribeStackResource',
        'cloudformation:GetTemplate',
        'cloudformation:ListStackResources',
        'cloudformation:ValidateTemplate',
      ],
      resources: ['*'], // Stack listing and describe operations require wildcard
    }));
    
    // Grant comprehensive IAM permissions for service operations
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'iam:GetRole',
        'iam:GetRolePolicy',
        'iam:ListRolePolicies',
        'iam:ListAttachedRolePolicies',
        'iam:GetUser',
        'iam:GetGroup',
        'iam:ListUsers',
        'iam:ListGroups',
        'sts:GetCallerIdentity',
        'sts:AssumeRole',
      ],
      resources: ['*'],
    }));
    
    // Grant Secrets Manager permissions for database config
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'secretsmanager:GetSecretValue',
      ],
      resources: [
        `arn:aws:secretsmanager:${this.region}:${this.account}:secret:*`,
      ],
    }));
    
    // Grant specific Bedrock Knowledge Base permissions
    lambdaRole.addToPolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'bedrock:ListKnowledgeBases',
        'bedrock:GetKnowledgeBase',
        'bedrock:ListDataSources',
        'bedrock:GetDataSource',
        'bedrock:RetrieveAndGenerate',
        'bedrock:Retrieve',
      ],
      resources: [
        `arn:aws:bedrock:${this.region}:${this.account}:knowledge-base/*`,
        `arn:aws:bedrock:${this.region}:${this.account}:data-source/*`,
      ],
    }));
    
    // Create Lambda functions
    const chatLambda = this.createLambdaFunction('ChatFunction', 'chat', props, lambdaRole);
    const botsLambda = this.createLambdaFunction('BotsFunction', 'bots', props, lambdaRole);
    const knowledgeLambda = this.createLambdaFunction('KnowledgeFunction', 'knowledge', props, lambdaRole);
    
    // Define API routes
    // Bots API (with proper Cognito authorization)
    const botsResource = this.apiGateway.root.addResource('bots');
    this.addCrudEndpoints(botsResource, botsLambda, authorizer);
    
    // Individual bot API
    const botResource = botsResource.addResource('{id}');
    this.addCrudEndpoints(botResource, botsLambda, authorizer, ['GET', 'PUT', 'DELETE']);
    
    // Bot chat endpoints
    const botChatResource = botResource.addResource('chat');
    botChatResource.addMethod('POST', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
    // Bot conversations endpoints (bots lambda routes internally to chat handler)
    const botConversationsResource = botResource.addResource('conversations');
    botConversationsResource.addMethod('GET', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    botConversationsResource.addMethod('POST', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
    // Individual bot conversation endpoint
    const botConversationResource = botConversationsResource.addResource('{conversationId}');
    botConversationResource.addMethod('GET', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    botConversationResource.addMethod('DELETE', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
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
    
    // Knowledge bases API (routed to bots Lambda which handles KB endpoints)
    const knowledgeResource = this.apiGateway.root.addResource('knowledge-bases');
    knowledgeResource.addMethod('GET', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
    // Knowledge base specific endpoints
    const knowledgeIdResource = knowledgeResource.addResource('{id}');
    knowledgeIdResource.addMethod('GET', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
    // Knowledge base validation endpoint
    const knowledgeValidateResource = knowledgeResource.addResource('validate');
    knowledgeValidateResource.addMethod('POST', new apigateway.LambdaIntegration(botsLambda), {
      authorizer,
    });
    
    // Create WebSocket API for streaming
    const connectHandler = this.createLambdaFunction('WebSocketConnectFunction', 'websocket', props, lambdaRole, 'connect');
    const disconnectHandler = this.createLambdaFunction('WebSocketDisconnectFunction', 'websocket', props, lambdaRole, 'disconnect');
    const defaultHandler = this.createLambdaFunction('WebSocketDefaultFunction', 'websocket', props, lambdaRole, 'default');
    
    this.websocket = new apigatewayv2.WebSocketApi(this, 'WebSocketApi', {
      apiName: `${props.config.prefix}WebSocketAPI`,
      connectRouteOptions: {
        integration: new integrations.WebSocketLambdaIntegration('ConnectIntegration', connectHandler),
      },
      disconnectRouteOptions: {
        integration: new integrations.WebSocketLambdaIntegration('DisconnectIntegration', disconnectHandler),
      },
      defaultRouteOptions: {
        integration: new integrations.WebSocketLambdaIntegration('DefaultIntegration', defaultHandler),
      },
    });
    
    // Deploy WebSocket API
    const webSocketStage = new apigatewayv2.WebSocketStage(this, 'WebSocketStage', {
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
      DOCUMENTS_BUCKET: props.storageBucket.bucketName,
      USER_POOL_ID: props.userPool.userPoolId,
      // AWS_REGION is automatically provided by Lambda runtime
    };
    
    // Path to the actual Go Lambda binary
    const lambdaPath = path.join(__dirname, '../../lambda/functions', handlerDir);
    
    // Create the Lambda function
    const fn = new lambda.Function(this, id, {
      runtime: lambda.Runtime.PROVIDED_AL2023,
      architecture: lambda.Architecture.ARM_64,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset(lambdaPath),
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
    authorizer?: apigateway.CognitoUserPoolsAuthorizer,
    methods: string[] = ['GET', 'POST', 'PUT', 'DELETE']
  ) {
    const integration = new apigateway.LambdaIntegration(lambdaFn);
    const methodOptions = authorizer ? { authorizer } : {};
    
    if (methods.includes('GET')) {
      resource.addMethod('GET', integration, methodOptions);
    }
    
    if (methods.includes('POST')) {
      resource.addMethod('POST', integration, methodOptions);
    }
    
    if (methods.includes('PUT')) {
      resource.addMethod('PUT', integration, methodOptions);
    }
    
    if (methods.includes('DELETE')) {
      resource.addMethod('DELETE', integration, methodOptions);
    }
  }
}