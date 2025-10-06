import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as cr from 'aws-cdk-lib/custom-resources';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import { Construct } from 'constructs';
import * as path from 'path';
import { Config } from './config';
import * as fs from 'fs';

export interface FrontendStackProps extends cdk.StackProps {
  config: Config;
  userPool: cognito.UserPool;
  userPoolClient: cognito.UserPoolClient;
  apiGateway: apigateway.RestApi;
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
      bucketName: `${props.config.prefix.toLowerCase()}website-${this.account}`.toLowerCase(),
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
    });
    
    // Create Origin Access Control (OAC)
    const originAccessControl = new cloudfront.S3OriginAccessControl(this, 'OAC', {
      description: 'OAC for Foundry website bucket',
    });

    // Create CloudFront Function to rewrite /prod/* paths to /* before sending to origin
    const apiPathRewriteFunction = new cloudfront.Function(this, 'ApiPathRewriteFunction', {
      functionName: `${props.config.prefix}ApiPathRewrite`,
      code: cloudfront.FunctionCode.fromFile({
        filePath: path.join(__dirname, 'edge-functions/api-path-rewrite.js'),
      }),
      runtime: cloudfront.FunctionRuntime.JS_2_0,
      comment: 'Strips /prod prefix from API requests before sending to API Gateway',
    });

    // Create CloudFront Function to prevent error page substitution for API calls
    const apiErrorFunction = new cloudfront.Function(this, 'ApiErrorFunction', {
      functionName: `${props.config.prefix}ApiErrorHandler`,
      code: cloudfront.FunctionCode.fromFile({
        filePath: path.join(__dirname, 'edge-functions/api-error-handler.js'),
      }),
      runtime: cloudfront.FunctionRuntime.JS_2_0,
      comment: 'Prevents error page substitution for API requests',
    });

    // Create CloudFront distribution with proper caching for SPA
    this.distribution = new cloudfront.Distribution(this, 'Distribution', {
      comment: 'Foundry Frontend Distribution',
      defaultBehavior: {
        origin: origins.S3BucketOrigin.withOriginAccessControl(this.websiteBucket, {
          originAccessControl,
        }),
        viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
        allowedMethods: cloudfront.AllowedMethods.ALLOW_GET_HEAD,
        cachedMethods: cloudfront.CachedMethods.CACHE_GET_HEAD,
        compress: true,
        cachePolicy: new cloudfront.CachePolicy(this, 'SPACachePolicy', {
          cachePolicyName: `${props.config.prefix}SPACachePolicy`,
          comment: 'Cache policy optimized for Single Page Applications',
          defaultTtl: cdk.Duration.hours(24),
          maxTtl: cdk.Duration.days(365),
          minTtl: cdk.Duration.seconds(0),
          headerBehavior: cloudfront.CacheHeaderBehavior.none(),
          queryStringBehavior: cloudfront.CacheQueryStringBehavior.none(),
          cookieBehavior: cloudfront.CacheCookieBehavior.none(),
        }),
      },
      additionalBehaviors: {
        // API calls - proxy to API Gateway (same-origin, no CORS needed)
        // Pattern: /{env}/* where env matches the API Gateway stage (dev, prod, staging, etc.)
        [`/${props.apiGateway.deploymentStage.stageName}/*`]: {
          origin: new origins.HttpOrigin(`${props.apiGateway.restApiId}.execute-api.${this.region}.amazonaws.com`, {
            protocolPolicy: cloudfront.OriginProtocolPolicy.HTTPS_ONLY,
            customHeaders: {
              'X-Debug-Origin': 'API-Gateway-Origin',
            },
            originPath: `/${props.apiGateway.deploymentStage.stageName}`, // Prepend API Gateway stage to rewritten path
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
          originRequestPolicy: cloudfront.OriginRequestPolicy.ALL_VIEWER_EXCEPT_HOST_HEADER,
          allowedMethods: cloudfront.AllowedMethods.ALLOW_ALL,
          functionAssociations: [
            {
              // First: Strip environment prefix from request URI (/{env}/path -> /path)
              function: apiPathRewriteFunction,
              eventType: cloudfront.FunctionEventType.VIEWER_REQUEST,
            },
            {
              // Second: Prevent error page substitution for API errors
              function: apiErrorFunction,
              eventType: cloudfront.FunctionEventType.VIEWER_RESPONSE,
            },
          ],
        },
        // Static assets (JS, CSS, images) - cache aggressively
        '/assets/*': {
          origin: origins.S3BucketOrigin.withOriginAccessControl(this.websiteBucket, {
            originAccessControl,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          cachePolicy: cloudfront.CachePolicy.CACHING_OPTIMIZED,
          compress: true,
        },
        // Config.json - never cache (so deployments update immediately)
        '/config.json': {
          origin: origins.S3BucketOrigin.withOriginAccessControl(this.websiteBucket, {
            originAccessControl,
          }),
          viewerProtocolPolicy: cloudfront.ViewerProtocolPolicy.REDIRECT_TO_HTTPS,
          cachePolicy: cloudfront.CachePolicy.CACHING_DISABLED,
        },
      },
      defaultRootObject: 'index.html',
      errorResponses: [
        {
          httpStatus: 404,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.minutes(5),
        },
        {
          httpStatus: 403,
          responseHttpStatus: 200,
          responsePagePath: '/index.html',
          ttl: cdk.Duration.minutes(5),
        },
      ],
      priceClass: cloudfront.PriceClass.PRICE_CLASS_100,
    });
    
    // Grant CloudFront access to S3 bucket
    this.websiteBucket.addToResourcePolicy(
      new iam.PolicyStatement({
        effect: iam.Effect.ALLOW,
        principals: [new iam.ServicePrincipal('cloudfront.amazonaws.com')],
        actions: ['s3:GetObject'],
        resources: [`${this.websiteBucket.bucketArn}/*`],
        conditions: {
          StringEquals: {
            'AWS:SourceArn': `arn:aws:cloudfront::${this.account}:distribution/${this.distribution.distributionId}`,
          },
        },
      })
    );
    
    // Check for React build directory
    const reactBuildPath = path.join(__dirname, '../../react-frontend/dist');
    
    console.log('🔍 Checking for React build...');
    console.log('React build path:', reactBuildPath);
    
    
    // Create a custom resource Lambda to handle config.json with proper token resolution
    const configLambda = new lambda.Function(this, 'ConfigLambda', {
      runtime: lambda.Runtime.PYTHON_3_11,
      handler: 'index.handler',
      code: lambda.Code.fromInline(`
import json
import boto3
import cfnresponse
import traceback

def handler(event, context):
    try:
        print(f"Event: {json.dumps(event)}")
        s3 = boto3.client('s3')
        
        if event['RequestType'] == 'Delete':
            print("Delete request - skipping config.json deletion")
            cfnresponse.send(event, context, cfnresponse.SUCCESS, {})
            return
            
        # Get properties from event
        bucket_name = event['ResourceProperties']['BucketName']
        config_data = event['ResourceProperties']['ConfigData']
        distribution_id = event['ResourceProperties']['DistributionId']
        
        print(f"Uploading config.json to bucket: {bucket_name}")
        print(f"Config data: {json.dumps(config_data, indent=2)}")
        
        # Upload config.json to S3
        response = s3.put_object(
            Bucket=bucket_name,
            Key='config.json',
            Body=json.dumps(config_data, indent=2),
            ContentType='application/json',
            CacheControl='no-cache'
        )
        
        print(f"S3 upload response: {response}")
        
        # Invalidate CloudFront cache for config.json
        if distribution_id:
            print(f"Invalidating CloudFront cache for distribution: {distribution_id}")
            cloudfront = boto3.client('cloudfront')
            invalidation_response = cloudfront.create_invalidation(
                DistributionId=distribution_id,
                InvalidationBatch={
                    'Paths': {
                        'Quantity': 1,
                        'Items': ['/config.json']
                    },
                    'CallerReference': str(context.aws_request_id)
                }
            )
            print(f"CloudFront invalidation response: {invalidation_response}")
        
        print("Config deployment completed successfully")
        cfnresponse.send(event, context, cfnresponse.SUCCESS, {})
        
    except Exception as e:
        print(f"Error: {str(e)}")
        print(f"Traceback: {traceback.format_exc()}")
        cfnresponse.send(event, context, cfnresponse.FAILED, {})
      `),
      timeout: cdk.Duration.minutes(5),
    });
    
    // Grant permissions to the Lambda
    this.websiteBucket.grantWrite(configLambda);
    configLambda.addToRolePolicy(new iam.PolicyStatement({
      effect: iam.Effect.ALLOW,
      actions: ['cloudfront:CreateInvalidation'],
      resources: ['*'],
    }));
    
    // Create custom resource to deploy config.json
    const configDeployment = new cdk.CustomResource(this, 'ConfigDeployment', {
      serviceToken: configLambda.functionArn,
      properties: {
        BucketName: this.websiteBucket.bucketName,
        DistributionId: this.distribution.distributionId,
        ConfigData: {
          environment: props.config.env,
          userPoolId: props.userPool.userPoolId,
          userPoolClientId: props.userPoolClient.userPoolClientId,
          apiEndpoint: props.apiEndpoint,
          websocketEndpoint: props.websocketEndpoint,
          region: this.region,
        },
      },
    });
    
    // Deploy website content after config is created
    if (fs.existsSync(reactBuildPath) && fs.existsSync(path.join(reactBuildPath, 'index.html'))) {
      console.log('✅ Found React build, deploying...');
      
      // Deploy React build (preserve config.json created by custom resource)
      const reactDeployment = new s3deploy.BucketDeployment(this, 'ReactDeployment', {
        sources: [s3deploy.Source.asset(reactBuildPath)],
        destinationBucket: this.websiteBucket,
        distribution: this.distribution,
        distributionPaths: ['/*'],
        prune: false, // Don't delete config.json created by custom resource
        retainOnDelete: false,
        exclude: ['config.json'], // Explicitly exclude config.json from deployment
      });
      
      // Ensure config is deployed before React app
      reactDeployment.node.addDependency(configDeployment);
      
    } else {
      console.log('❌ No React build found. Run "npm run build" in react-frontend directory first.');
      
      // Deploy minimal placeholder
      const placeholderDeployment = new s3deploy.BucketDeployment(this, 'PlaceholderDeployment', {
        sources: [
          s3deploy.Source.jsonData('index.html', `
            <!DOCTYPE html>
            <html>
            <head><title>Foundry</title></head>
            <body>
              <h1>Foundry - Build Missing</h1>
              <p>Please run "npm run build" in the react-frontend directory and redeploy.</p>
            </body>
            </html>
          `),
        ],
        destinationBucket: this.websiteBucket,
        distribution: this.distribution,
        distributionPaths: ['/*'],
      });
      
      // Ensure config is deployed before placeholder
      placeholderDeployment.node.addDependency(configDeployment);
    }
    
    this.websiteUrl = `https://${this.distribution.distributionDomainName}`;
    
    // Outputs
    new cdk.CfnOutput(this, 'WebsiteURL', {
      value: this.websiteUrl,
      description: 'Foundry Website URL',
      exportName: `${props.config.prefix}WebsiteURL`,
    });
    
    new cdk.CfnOutput(this, 'WebsiteBucketName', {
      value: this.websiteBucket.bucketName,
      description: 'Website S3 Bucket Name',
      exportName: `${props.config.prefix}WebsiteBucketName`,
    });
    
    new cdk.CfnOutput(this, 'CloudFrontDistributionId', {
      value: this.distribution.distributionId,
      description: 'CloudFront Distribution ID',
      exportName: `${props.config.prefix}CloudFrontDistributionId`,
    });
    
    // Debug outputs
    new cdk.CfnOutput(this, 'ApiGatewayId', {
      value: props.apiGateway.restApiId,
      description: 'API Gateway REST API ID',
    });
    
    new cdk.CfnOutput(this, 'ApiGatewayStageName', {
      value: props.apiGateway.deploymentStage.stageName,
      description: 'API Gateway Stage Name',
    });
    
    new cdk.CfnOutput(this, 'ExpectedApiOrigin', {
      value: `${props.apiGateway.restApiId}.execute-api.${this.region}.amazonaws.com`,
      description: 'Expected API Gateway Origin Domain',
    });
  }

}