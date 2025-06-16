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

// Add explicit dependency
storageStack.addDependency(networkStack);

const authStack = new AuthStack(app, `${config.prefix}AuthStack`, {
  env: config.envProps,
  config,
});

// Add explicit dependency
authStack.addDependency(storageStack);

// Skip the auth Lambda stack for now to avoid circular dependencies
// const authLambdaStack = new AuthLambdaStack(app, `${config.prefix}AuthLambdaStack`, {
//   env: config.envProps,
//   config,
//   userPool: authStack.userPool,
// });

// authLambdaStack.addDependency(authStack);

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
  storageBucket: storageStack.storageBucket,
});

// Add explicit dependencies
apiStack.addDependency(networkStack);
apiStack.addDependency(storageStack);
apiStack.addDependency(authStack);
// apiStack.addDependency(authLambdaStack); // Commented out for now

// Create proper frontend stack with React support
const frontendStack = new FrontendStack(app, `${config.prefix}FrontendStack`, {
  env: config.envProps,
  config,
  userPool: authStack.userPool,
  userPoolClient: authStack.userPoolClient,
  apiGateway: apiStack.apiGateway,
  apiEndpoint: apiStack.apiEndpoint,
  websocketEndpoint: apiStack.websocketEndpoint,
});

// Add dependencies
frontendStack.addDependency(authStack);
frontendStack.addDependency(apiStack);

const monitoringStack = new MonitoringStack(app, `${config.prefix}MonitoringStack`, {
  env: config.envProps,
  config,
  apiGateway: apiStack.apiGateway,
  lambdaFunctions: apiStack.lambdaFunctions,
  dynamoTables: storageStack.allTables,
});

// Add explicit dependencies
monitoringStack.addDependency(apiStack);
monitoringStack.addDependency(storageStack);

// Apply tags to all stacks
const tags = {
  Environment: env,
  Project: 'AmazonBuddy',
  ManagedBy: 'CDK',
};

Object.entries(tags).forEach(([key, value]) => {
  cdk.Tags.of(app).add(key, value);
});