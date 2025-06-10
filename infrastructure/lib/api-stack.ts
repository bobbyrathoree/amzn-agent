import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as path from 'path';
import * as fs from 'fs';
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
      defaultCorsPreflightOptions: {
        allowOrigins: apigateway.Cors.ALL_ORIGINS,
        allowMethods: apigateway.Cors.ALL_METHODS,
        allowHeaders: [
          'Content-Type',
          'Authorization',
          'X-Amz-Date',
          'X-Api-Key',
          'x-user-id',
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
      identitySource: 'method.request.header.Authorization',
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
    // Bots API (with proper Cognito authorization)
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
      USER_POOL_ID: props.userPool.userPoolId,
      REGION: props.env?.region || 'us-east-1',
    };
    
    // Create a temporary directory with a placeholder file for the Lambda code
    // In a real implementation, this would point to your actual Go Lambda code
    const tempDir = `/tmp/lambda-${id}-${Date.now()}`;
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true });
      fs.writeFileSync(`${tempDir}/bootstrap`, '#!/bin/sh\necho "This is a placeholder Lambda function"');
      fs.chmodSync(`${tempDir}/bootstrap`, 0o755);
    }
    
    // Create the Lambda function
    const fn = new lambda.Function(this, id, {
      runtime: lambda.Runtime.PROVIDED_AL2,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset(tempDir),
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