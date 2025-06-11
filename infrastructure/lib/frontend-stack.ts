import * as cdk from 'aws-cdk-lib';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as cloudfront from 'aws-cdk-lib/aws-cloudfront';
import * as origins from 'aws-cdk-lib/aws-cloudfront-origins';
import * as s3deploy from 'aws-cdk-lib/aws-s3-deployment';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as cr from 'aws-cdk-lib/custom-resources';
import { Construct } from 'constructs';
import * as path from 'path';
import { Config } from './config';
import * as fs from 'fs';

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
      bucketName: `${props.config.prefix.toLowerCase()}website-${this.account}`.toLowerCase(),
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
    });
    
    // Create Origin Access Control (OAC)
    const originAccessControl = new cloudfront.S3OriginAccessControl(this, 'OAC', {
      description: 'OAC for AmazonBuddy website bucket',
    });
    
    // Create CloudFront distribution with proper caching for SPA
    this.distribution = new cloudfront.Distribution(this, 'Distribution', {
      comment: 'AmazonBuddy Frontend Distribution',
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
    
    if (fs.existsSync(reactBuildPath) && fs.existsSync(path.join(reactBuildPath, 'index.html'))) {
      console.log('✅ Found React build, deploying...');
      
      // Deploy React build without config.json first
      new s3deploy.BucketDeployment(this, 'ReactDeployment', {
        sources: [s3deploy.Source.asset(reactBuildPath)],
        destinationBucket: this.websiteBucket,
        distribution: this.distribution,
        distributionPaths: ['/*'],
        prune: true,
        retainOnDelete: false,
      });
      
    } else {
      console.log('❌ No React build found. Run "npm run build" in react-frontend directory first.');
      
      // Deploy minimal placeholder
      new s3deploy.BucketDeployment(this, 'PlaceholderDeployment', {
        sources: [
          s3deploy.Source.jsonData('index.html', `
            <!DOCTYPE html>
            <html>
            <head><title>AmazonBuddy</title></head>
            <body>
              <h1>AmazonBuddy - Build Missing</h1>
              <p>Please run "npm run build" in the react-frontend directory and redeploy.</p>
            </body>
            </html>
          `),
        ],
        destinationBucket: this.websiteBucket,
        distribution: this.distribution,
        distributionPaths: ['/*'],
      });
    }
    
    // Create a custom resource Lambda to handle config.json with proper token resolution
    const configLambda = new lambda.Function(this, 'ConfigLambda', {
      runtime: lambda.Runtime.PYTHON_3_11,
      handler: 'index.handler',
      code: lambda.Code.fromInline(`
import json
import boto3
import cfnresponse

def handler(event, context):
    try:
        s3 = boto3.client('s3')
        
        if event['RequestType'] == 'Delete':
            cfnresponse.send(event, context, cfnresponse.SUCCESS, {})
            return
            
        # Get properties from event
        bucket_name = event['ResourceProperties']['BucketName']
        config_data = event['ResourceProperties']['ConfigData']
        distribution_id = event['ResourceProperties']['DistributionId']
        
        # Upload config.json to S3
        s3.put_object(
            Bucket=bucket_name,
            Key='config.json',
            Body=json.dumps(config_data, indent=2),
            ContentType='application/json'
        )
        
        # Invalidate CloudFront cache for config.json
        if distribution_id:
            cloudfront = boto3.client('cloudfront')
            cloudfront.create_invalidation(
                DistributionId=distribution_id,
                InvalidationBatch={
                    'Paths': {
                        'Quantity': 1,
                        'Items': ['/config.json']
                    },
                    'CallerReference': str(context.aws_request_id)
                }
            )
        
        cfnresponse.send(event, context, cfnresponse.SUCCESS, {})
        
    except Exception as e:
        print(f"Error: {str(e)}")
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
    new cdk.CustomResource(this, 'ConfigDeployment', {
      serviceToken: configLambda.functionArn,
      properties: {
        BucketName: this.websiteBucket.bucketName,
        DistributionId: this.distribution.distributionId,
        ConfigData: {
          environment: props.config.env,
          userPoolId: props.userPool.userPoolId,
          userPoolClientId: props.userPoolClient.userPoolClientId,
          identityPoolId: props.identityPool.ref,
          apiEndpoint: props.apiEndpoint,
          websocketEndpoint: props.websocketEndpoint,
          region: this.region,
        },
      },
    });
    
    this.websiteUrl = `https://${this.distribution.distributionDomainName}`;
    
    // Outputs
    new cdk.CfnOutput(this, 'WebsiteURL', {
      value: this.websiteUrl,
      description: 'AmazonBuddy Website URL',
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
  }
}