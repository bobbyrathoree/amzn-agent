import * as cdk from 'aws-cdk-lib';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as path from 'path';
import { Construct } from 'constructs';
import { Config } from './config';

export interface AuthStackProps extends cdk.StackProps {
  config: Config;
}

export class AuthStack extends cdk.Stack {
  public readonly userPool: cognito.UserPool;
  public readonly userPoolClient: cognito.UserPoolClient;
  public readonly identityPool: cognito.CfnIdentityPool;
  
  constructor(scope: Construct, id: string, props: AuthStackProps) {
    super(scope, id, props);
    
    // Step 1: Create Lambda functions (no user pool dependencies)
    const preSignupLambda = new lambda.Function(this, 'PreSignupLambda', {
      runtime: lambda.Runtime.PYTHON_3_11,
      handler: 'check-email-domain.handler',
      code: lambda.Code.fromAsset(path.join(__dirname, '../lambda-auth')),
      description: 'Validates email domain during user signup - only @amazon.com allowed',
      timeout: cdk.Duration.seconds(30),
    });
    
    const postConfirmationLambda = new lambda.Function(this, 'PostConfirmationLambda', {
      runtime: lambda.Runtime.PYTHON_3_11,
      handler: 'add-user-to-groups.handler',
      code: lambda.Code.fromAsset(path.join(__dirname, '../lambda-auth')),
      description: 'Automatically adds new users to BotCreators group',
      timeout: cdk.Duration.seconds(30),
      // No environment variables needed - getting user pool ID from event
    });
    
    // Grant permissions to post-confirmation Lambda (no user pool reference yet)
    postConfirmationLambda.addToRolePolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: [
        'cognito-idp:AdminAddUserToGroup',
        'cognito-idp:AdminListGroupsForUser',
        'cognito-idp:ListGroups',
      ],
      resources: ['*'], // Broad permissions for now
    }));
    
    // Step 2: Create user pool with both triggers
    this.userPool = new cognito.UserPool(this, 'UserPool', {
      userPoolName: `${props.config.prefix}UserPool`,
      selfSignUpEnabled: false, // Disabled: Only admin can create users
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
      },
      passwordPolicy: {
        minLength: 8,
        requireLowercase: true,
        requireUppercase: true,
        requireDigits: true,
        requireSymbols: true,
      },
      removalPolicy: cdk.RemovalPolicy.RETAIN,
      lambdaTriggers: {
        preSignUp: preSignupLambda,
        postConfirmation: postConfirmationLambda,
      },
    });
    
    // Step 3: DON'T update the environment - it will cause circular dependency
    // The Lambda will need to figure out the user pool ID another way or use a fixed value
    
    // Create user pool client
    this.userPoolClient = this.userPool.addClient('UserPoolClient', {
      userPoolClientName: `${props.config.prefix}UserPoolClient`,
      authFlows: {
        userPassword: true,
        userSrp: true,
      },
      generateSecret: false, // Important: API Gateway Cognito authorizer doesn't work with client secrets
      accessTokenValidity: cdk.Duration.hours(1),
      idTokenValidity: cdk.Duration.hours(1),
      refreshTokenValidity: cdk.Duration.days(30),
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
    
    // Create identity pool for federated identities
    this.identityPool = new cognito.CfnIdentityPool(this, 'IdentityPool', {
      identityPoolName: `${props.config.prefix}IdentityPool`,
      allowUnauthenticatedIdentities: false,
      cognitoIdentityProviders: [{
        clientId: this.userPoolClient.userPoolClientId,
        providerName: this.userPool.userPoolProviderName,
      }],
    });
    
    // Create roles for authenticated and unauthenticated users
    const authenticatedRole = new iam.Role(this, 'AuthenticatedRole', {
      assumedBy: new iam.FederatedPrincipal('cognito-identity.amazonaws.com', {
        StringEquals: {
          'cognito-identity.amazonaws.com:aud': this.identityPool.ref,
        },
        'ForAnyValue:StringLike': {
          'cognito-identity.amazonaws.com:amr': 'authenticated',
        },
      }),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('AmazonS3ReadOnlyAccess'),
      ],
    });
    
    // Attach roles to identity pool
    new cognito.CfnIdentityPoolRoleAttachment(this, 'IdentityPoolRoleAttachment', {
      identityPoolId: this.identityPool.ref,
      roles: {
        authenticated: authenticatedRole.roleArn,
      },
    });
    
    new cdk.CfnOutput(this, 'IdentityPoolId', {
      value: this.identityPool.ref,
      description: 'Identity Pool ID',
      exportName: `${props.config.prefix}IdentityPoolId`,
    });
  }
}